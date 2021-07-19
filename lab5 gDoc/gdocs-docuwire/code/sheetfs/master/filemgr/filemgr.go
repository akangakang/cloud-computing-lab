package filemgr

import (
	"context"
	"github.com/fourstring/sheetfs/common_journal"
	"github.com/fourstring/sheetfs/master/datanode_alloc"
	"github.com/fourstring/sheetfs/master/filemgr/file_errors"
	"github.com/fourstring/sheetfs/master/filemgr/mgr_entry"
	"github.com/fourstring/sheetfs/master/journal/checkpoint"
	"github.com/fourstring/sheetfs/master/journal/journal_entry"
	"github.com/fourstring/sheetfs/master/sheetfile"
	fs_rpc "github.com/fourstring/sheetfs/protocol"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
	"sync"
	"time"
)

/*
FileManager
Represents a top-level directory of SheetFiles stored in the filesystem.
There is only one level of directory, mapping filename to SheetFile directly, or
raise errors for invalid filename.

FileManager exposes both filename-oriented and fd-oriented API at the same time.
The relationship between them is Unix-alike. In other words, applications need to
Open a file with its filename, then a fd will be returned. Subsequently operations
must be invoked by a fd. The management of fds is also Unix-alike: multiple fds can
be pointed to the same underlying SheetFile. The only difference is that there is
only a global fd namespace, not one for per client.

FileManager is responsible for implementing almost all APIs provided to outer applications.
It's a goroutine-safe data structure. To maximize concurrency, it exploits a two-level
locking strategy. It acquires itself RWMutex, mu, to lookup directory Entries or fd table
safely. Then it release mu immediately, and rely on SheetFile.mu to guarantee goroutine-safe
access to specific SheetFiles.

As a directory, FileManager should be persisted during checkpointing. Besides, it manages
all SheetFiles at the same time, so it should not only persist itself, but also persist those
files' metadata as a helper. It also plays as an in-memory cache for filesystem metadata of those
files, loading them into memory on-demand. So a gorm connection is required.
*/
type FileManager struct {
	mu sync.RWMutex
	// All directory entries in the directory. All of them should be loaded into memory once.
	Entries map[string]*mgr_entry.MapEntry
	// Maps filename to a already opened SheetFile. This map is fulfilled on-demand. If a SheetFile
	// is not being opened currently, it's not presented in the map.
	Opened map[string]*sheetfile.SheetFile
	// Maps a fd to a opened filename. Multiple fds are allowed to be pointed to the same file. So
	// their entries in this map will contain same filename.
	Fds map[uint64]string
	// Next available fd to be allocated to respond a Open or Create file operation.
	nextFd        uint64
	db            *gorm.DB
	alloc         *datanode_alloc.DataNodeAllocator
	journalWriter *common_journal.Writer
	logger        *zap.Logger
}

func (f *FileManager) writeJournal(jEntry *journal_entry.MasterEntry) error {
	if f.journalWriter != nil {
		buf, err := proto.Marshal(jEntry)
		if err != nil {
			return err
		}
		err = f.journalWriter.CommitEntry(context.TODO(), buf)
		return err
	}
	return nil
}

/*
allocFd
allocate a new fd to respond a Open or Create file operation. It simply return nextFd and increase
it by 1 currently.
*/
func (f *FileManager) allocFd() uint64 {
	fd := f.nextFd
	f.nextFd += 1
	return fd
}

/*
openFile
Open an existing SheetFile and allocate a fd which can be used to subsequently
access to this file. If this file is not loaded into memory, do it on-demand.

@para
	filename

@return
	uint64: fd to access this file if filename is valid
	error:
		*errors.FileNotFoundError if the filename is invalid or file has been
		recycled.
*/
func (f *FileManager) openFile(filename string) (uint64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// Lookup in directory entries to check validity of filename.
	entry, ok := f.Entries[filename]
	// If there is no such an entry, or the entry has been marked as recycled,
	// the Open operation is invalid.
	if !ok || entry.Recycled {
		return 0, file_errors.NewFileNotFoundError(filename)
	}
	openedFile, ok := f.Opened[filename]
	if !ok {
		// Load file metadata into memory from sqlite on-demand.
		openedFile = sheetfile.LoadSheetFile(f.db, f.alloc, filename)
		f.Opened[filename] = openedFile
	}
	fd := f.allocFd()
	// Add new allocated fd to fd table, points to the filename of opened file.
	f.Fds[fd] = filename
	return fd, nil
}

/*
getFileByFd
@para
	fd

@return
	*SheetFile: pointer of opened file
	error:
		*errors.FdNotFoundError if the fd is invalid
*/
func (f *FileManager) getFileByFd(fd uint64) (*sheetfile.SheetFile, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	filename, ok := f.Fds[fd]
	if !ok {
		return nil, file_errors.NewFdNotFoundError(fd)
	}
	return f.Opened[filename], nil
}

/*
CreateSheet
Creates a new SheetFile and open it immediately.

@para
	filename

@return
	uint64: fd of the newly created file.
	error:
		*errors.FileExistsError if there has been a file with the same filename.
		Although the existing file has been recycled, creating a file with the
		same filename is not allowed.
		errors raised during sheetfile.CreateSheetFile
*/
func (f *FileManager) CreateSheet(filename string) (uint64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.Entries[filename]
	if ok {
		return 0, file_errors.NewFileExistsError(filename)
	}
	/*
		TODO: refactor sheetfile creation to a two-stage one, allocate newCell and newChunk first,
		And then actually create newCell, newChunk in DB. Such a design is more fit for journaling,
		enabling journaling really comes first.
	*/
	sheet, newCell, newChunk, err := sheetfile.CreateSheetFile(f.db, f.alloc, filename)
	if err != nil {
		return 0, err
	}
	newEntry := &mgr_entry.MapEntry{
		FileName:       filename,
		CellsTableName: sheetfile.GetCellTableName(filename),
		Recycled:       false,
	}
	err = f.writeJournal(&journal_entry.MasterEntry{
		XCell:    journal_entry.FromSheetCell(newCell),
		XChunk:   journal_entry.FromSheetChunk(newChunk),
		XFileMap: journal_entry.FromMgrEntry(newEntry),
	})
	if err != nil {
		return 0, err
	}
	f.Entries[filename] = newEntry
	// Add the new file to opened table and allocate an fd right after creation.
	f.Opened[filename] = sheet
	fd := f.allocFd()
	f.Fds[fd] = filename
	return fd, nil
}

/*
OpenSheet
@para
	filename

@return
	uint64: fd to access this file if filename is valid
	error:
		*errors.FileNotFoundError if the filename is invalid or file has been
		recycled.
*/
func (f *FileManager) OpenSheet(filename string) (uint64, error) {
	fd, err := f.openFile(filename)
	if err != nil {
		return 0, err
	}
	return fd, nil
}

/*
RecycleSheet
Mark a file as recycled and record RecycledAt. Do nothing if the filename is invalid.

The semantic of this method is Unix-alike too. It won't prevent applications who
have opened this file from editing this file, but later Open or Create operations
will fail.
*/
func (f *FileManager) RecycleSheet(filename string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	entry, ok := f.Entries[filename]
	if !ok {
		return nil
	}
	var tempEntry mgr_entry.MapEntry
	tempEntry = *entry
	tempEntry.Recycled = true
	tempEntry.RecycledAt = time.Now()
	err := f.writeJournal(&journal_entry.MasterEntry{
		XCell:    journal_entry.FromEmptySheetCell(),
		XChunk:   journal_entry.FromEmptyChunk(),
		XFileMap: journal_entry.FromMgrEntry(&tempEntry),
	})
	if err != nil {
		return err
	}
	f.Entries[filename] = &tempEntry

	return nil
}

/*
ResumeSheet
Mark a file as not recycled, so it can be Opened afterwards.
*/
func (f *FileManager) ResumeSheet(filename string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	entry, ok := f.Entries[filename]
	if !ok {
		return nil
	}
	var tempEntry mgr_entry.MapEntry
	tempEntry = *entry
	tempEntry.Recycled = false
	err := f.writeJournal(&journal_entry.MasterEntry{
		XCell:    journal_entry.FromEmptySheetCell(),
		XChunk:   journal_entry.FromEmptyChunk(),
		XFileMap: journal_entry.FromMgrEntry(&tempEntry),
	})
	if err != nil {
		return err
	}
	f.Entries[filename] = &tempEntry
	return nil
}

/*
Monitor
Continuously monitoring all files marked as recycled, and if some file has
exceeded configured retain time period, delete it forever.
*/
func (f *FileManager) Monitor() {
	// TODO
}

/*
ReadSheet
Read all contents of a SheetFile. This method returns all Chunk of a sheet,
applications should then contact with DataNodes to fetch actual content of
those Chunks.

@para
	fd

@returns
	[]*sheetfile.Chunk: List of all Chunks in a file.
	error:
		*errors.FdNotFoundError if the fd is invalid
*/
func (f *FileManager) ReadSheet(fd uint64) ([]*sheetfile.Chunk, error) {
	file, err := f.getFileByFd(fd)
	if err != nil {
		return nil, err
	}
	return file.GetAllChunks(), nil
}

/*
ReadFileCell
Read Cell located at (row, col) in a file pointed by fd.

@para
	fd, row, col

@return
	*Cell, *Chunk: snapshots of corresponding Cell and Chunk
	error:
		*errors.FdNotFoundError if the fd is invalid
		*errors.CellNotFoundError if row, col passed in is invalid.
*/
func (f *FileManager) ReadFileCell(fd uint64, row, col uint32) (*sheetfile.Cell, *sheetfile.Chunk, error) {
	file, err := f.getFileByFd(fd)
	if err != nil {
		return nil, nil, err
	}
	cell, dataChunk, err := file.GetCellChunk(row, col)
	if err != nil {
		return nil, nil, err
	}
	return cell, dataChunk, nil
}

/*
WriteFileCell
Write Cell located at (row, col) in a file pointed by fd. Create Cell if not existed.

@para
	fd, row, col

@return
	*Cell, *Chunk: snapshots of corresponding Cell and Chunk
	error:
		*errors.FdNotFoundError if the fd is invalid
		*errors.NoDataNodeError if there is no DataNode registered.
*/
func (f *FileManager) WriteFileCell(fd uint64, row, col uint32) (*sheetfile.Cell, *sheetfile.Chunk, error) {
	file, err := f.getFileByFd(fd)
	if err != nil {
		return nil, nil, err
	}
	cell, dataChunk, err := file.WriteCellChunk(row, col, f.db)
	if err != nil {
		return nil, nil, err
	}
	// TODO: refactor SheetFile.WriteCellChunk to a two-stage one.
	// Currently, we can only assume that Kafka is highly-available due to lack of time.
	_ = f.writeJournal(&journal_entry.MasterEntry{
		XCell:    journal_entry.FromSheetCell(cell),
		XChunk:   journal_entry.FromSheetChunk(dataChunk),
		XFileMap: journal_entry.FromEmptyMgrEntry(),
	})
	return cell, dataChunk, nil
}

/*
GetAllSheets
List all sheets stored in the directory.

@return
	[]*fs_rpc.Sheet: a slice of protobuf fs_rpc.Sheet model, contains
	filename and recycled field.
*/
func (f *FileManager) GetAllSheets() []*fs_rpc.Sheet {
	f.mu.RLock()
	defer f.mu.RUnlock()

	pbSheets := make([]*fs_rpc.Sheet, len(f.Entries))
	i := 0
	for _, entry := range f.Entries {
		pbSheets[i] = &fs_rpc.Sheet{
			Filename: entry.FileName,
			Recycled: entry.Recycled,
		}
		i += 1
	}
	return pbSheets
}

/*
Persistent
Flush the MapEntry and SheetFile data stored in a FileManager to sqlite in
a transaction.

@return
	error: error during the persistent transaction.
*/
func (f *FileManager) Persistent() error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	err := f.db.Transaction(func(tx *gorm.DB) error {
		for _, entry := range f.Entries {
			tx.Save(entry)
		}
		for _, file := range f.Opened {
			err := file.Persistent(tx)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

/*
LoadFileManager
Load all MapEntry from database and construct FileManager.Entries. Opened file
table and fd table may be recovered from journal.

This method should only be used to load checkpoints in sqlite.

@para
	db: a gorm connection. It can be a transaction

@return
	*FileManager
*/
func LoadFileManager(db *gorm.DB, alloc *datanode_alloc.DataNodeAllocator, writer *common_journal.Writer) *FileManager {
	fm := &FileManager{
		Entries:       map[string]*mgr_entry.MapEntry{},
		Opened:        map[string]*sheetfile.SheetFile{},
		Fds:           map[uint64]string{},
		nextFd:        0,
		db:            db,
		alloc:         alloc,
		journalWriter: writer,
	}
	var entries []*mgr_entry.MapEntry
	db.Find(&entries)
	for _, entry := range entries {
		fm.Entries[entry.FileName] = entry
	}
	logger, err := zap.NewDevelopment(zap.Fields(zap.String("source", "FileManager")))
	if err == nil {
		fm.logger = logger
	}
	return fm
}

func (f *FileManager) handleJournalMapEntry(mapEntry *journal_entry.FileMapEntry) {
	original, ok := f.Entries[mapEntry.Filename]
	if !ok {
		switch mapEntry.TargetState {
		case journal_entry.State_PRESENT:
			e := &mgr_entry.MapEntry{}
			journal_entry.ToMgrEntry(e, mapEntry)
			f.Entries[mapEntry.Filename] = e
		case journal_entry.State_ABSENT:
			// Do nothing
		}
	} else {
		switch mapEntry.TargetState {
		case journal_entry.State_PRESENT:
			journal_entry.ToMgrEntry(original, mapEntry)
		case journal_entry.State_ABSENT:
			delete(f.Entries, mapEntry.Filename)
		}
	}
}

func (f *FileManager) handleChunkEntry(file *sheetfile.SheetFile, chunk *journal_entry.ChunkEntry) {
	if originalChunk, ok := file.Chunks[chunk.Id]; !ok {
		switch chunk.TargetState {
		case journal_entry.State_PRESENT:
			c := &sheetfile.Chunk{}
			journal_entry.ToSheetChunk(c, chunk)
			file.Chunks[chunk.Id] = c
		case journal_entry.State_ABSENT:
			// Do nothing
		}
	} else {
		switch chunk.TargetState {
		case journal_entry.State_PRESENT:
			journal_entry.ToSheetChunk(originalChunk, chunk)
		case journal_entry.State_ABSENT:
			delete(file.Chunks, chunk.Id)
		}
	}
}

func (f *FileManager) ensureCellChunkConsistency(file *sheetfile.SheetFile, cell *sheetfile.Cell) {
	chunk := file.Chunks[cell.ChunkID]
	for i := 0; i < len(chunk.Cells); i++ {
		if originalCell := chunk.Cells[i]; originalCell.CellID == cell.CellID {
			if originalCell != cell {
				chunk.Cells[i] = cell
			}
			return
		}
	}
	chunk.Cells = append(chunk.Cells, cell)
}

func (f *FileManager) handleCellEntry(file *sheetfile.SheetFile, cell *journal_entry.CellEntry) error {
	if originalCell, ok := file.Cells[cell.CellId]; !ok {
		switch cell.TargetState {
		case journal_entry.State_PRESENT:
			err := sheetfile.CreateCellTableIfNotExists(f.db, cell.SheetName)
			if err != nil {
				return err
			}
			c := &sheetfile.Cell{}
			journal_entry.ToSheetCell(c, cell)
			file.Cells[cell.CellId] = c
			f.ensureCellChunkConsistency(file, c)
		case journal_entry.State_ABSENT:
			// Do nothing
		}
	} else {
		switch cell.TargetState {
		case journal_entry.State_PRESENT:
			journal_entry.ToSheetCell(originalCell, cell)
			f.ensureCellChunkConsistency(file, originalCell)
		case journal_entry.State_ABSENT:
			delete(file.Cells, cell.CellId)
		}
	}
	return nil
}

func (f *FileManager) HandleMasterEntry(entry *journal_entry.MasterEntry) error {
	if mapEntry := entry.GetMapEntry(); mapEntry != nil {
		f.handleJournalMapEntry(mapEntry)
	}
	cell, chunk := entry.GetCell(), entry.GetChunk()
	if cell == nil && chunk == nil {
		return nil
	}
	if cell == nil || chunk == nil {
		return journal_entry.NewInvalidJournalEntryError(entry)
	}
	if cell.ChunkId != chunk.Id || cell.TargetState != chunk.TargetState {
		return journal_entry.NewInvalidJournalEntryError(entry)
	}
	// After MapEntry recovery above, if this entry represents a file creation, the corresponding
	// should have been added to f.Entries, or if this entry is an operation on an existing file,
	// its MapEntry has already in f.Entries too. So if it's unable to find such an MapEntry until
	// now, this journal entry is invalid.
	if _, ok := f.Entries[cell.SheetName]; !ok {
		return journal_entry.NewInvalidJournalEntryError(entry)
	}
	file, ok := f.Opened[cell.SheetName]
	if !ok {
		file = sheetfile.LoadSheetFile(f.db, f.alloc, cell.SheetName)
		f.Opened[cell.SheetName] = file
	}
	f.handleChunkEntry(file, chunk)
	err := f.handleCellEntry(file, cell)

	return err
}

func (f *FileManager) DoCheckpoint() {
	f.mu.RLock()
	defer f.mu.RUnlock()
	offset, err := f.journalWriter.Checkpoint(context.Background())
	if err != nil {
		f.logger.Error("error when journaling checkpoint.", zap.Error(err))
		return
	}
	err = f.Persistent()
	if err != nil {
		if f.logger != nil {
			f.logger.Error("error when checkpointing.", zap.Error(err))
		}
		return
	}
	err = checkpoint.RecordCheckpoint(f.db, offset)
	if err != nil {
		if f.logger != nil {
			f.logger.Error("error when persist checkpoint.", zap.Error(err))
		}
	}
}
