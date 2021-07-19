package sheetfile

import (
	"github.com/fourstring/sheetfs/master/config"
	"github.com/fourstring/sheetfs/master/datanode_alloc"
	"github.com/fourstring/sheetfs/master/filemgr/file_errors"
	"gorm.io/gorm"
	"sync"
)

/*
SheetFile
Represents a file containing a sheet.
Every SheetFile is made of lots of Cell. Almost every Cell has
config.MaxBytesPerCell bytes of storage to store its data, and there
is a special one called MetaCell, whose size will be config.BytesPerChunk.
Applications should consider to make use of MetaCell to store data related
to whole sheet. MetaCell can be accessed by (config.SheetMetaCellRow, config.SheetMetaCellCol).

Composed of Cells, SheetFile provides a row/col-oriented API to applications.
Cells is the index of such an API. So Cells must be persistent, so as Chunks storing those
Cells. And as a logical collection of Cells and Chunks, SheetFile should be used as
a helper to persist Cells and Chunks. However, SheetFile itself has not to be kept permanently.
All of its data can be resumed by scanning Cells loaded from database.

Chunks and Cells are maintained in pointers. Sometimes those data will be returned to the
outside world, which implies we can't rely SheetFile.mu on accessing those pointers goroutine-safely.
So all pointers returned will point to a copy, or snapshot of a Chunk or Cell. In other words,
they are a 'goroutine-safe view' of Chunks/Cells at some point.
*/
type SheetFile struct {
	mu sync.RWMutex
	// All Chunks containing all Cells of the SheetFile.
	// Maps ChunkID to *Chunk.
	Chunks map[uint64]*Chunk
	// All Cells in the sheet.
	// Maps CellID to *Cell.
	Cells map[int64]*Cell
	// Keeps track of latest Chunk whose remaining space is capable of storing a new Cell.
	LastAvailableChunk *Chunk

	filename string
	alloc    *datanode_alloc.DataNodeAllocator
}

/*
CreateSheetFile
Create a SheetFile, corresponding sqlite table to store Cells of the SheetFile, MetaCell
and chunk used to store it.
Theoretically, it's not required to flush the metadata of a new file to disk immediately.
However, we make use of sqlite as some kind of alternative to general BTree. So we create
the table here as a workaround.

@para
	db: a gorm connection. It should not be a transaction.(See CreateCellTableIfNotExists)
	filename: filename of new SheetFile

@return
	*SheetFile: pointer of new SheetFile if success, or nil.
	error:
		some error happens while creating cell table.
		*errors.NoDataNodeError: This function must allocate a Chunk for MetaCell, if there
		are no DataNodes for storing this cell, returns NoDateNodeError.
*/
func CreateSheetFile(db *gorm.DB, alloc *datanode_alloc.DataNodeAllocator, filename string) (*SheetFile, *Cell, *Chunk, error) {
	f := &SheetFile{
		Chunks:             map[uint64]*Chunk{},
		Cells:              map[int64]*Cell{},
		LastAvailableChunk: nil,
		filename:           filename,
		alloc:              alloc,
	}
	err := f.persistentStructure(db)
	if err != nil {
		return nil, nil, nil, err
	}
	dataNode, err := alloc.AllocateNode()
	if err != nil {
		return nil, nil, nil, err
	}
	// Create Chunk and MetaCell
	chunk := &Chunk{DataNode: dataNode, Version: 0}
	chunk.Persistent(db)
	metaCell := NewCell(config.SheetMetaCellID, 0, config.BytesPerChunk, chunk.ID, filename)
	chunk.Cells = []*Cell{metaCell}
	// Add MetaCell and Chunk to new file
	f.Chunks[chunk.ID] = chunk
	f.Cells[metaCell.CellID] = metaCell
	return f, metaCell, chunk, nil
}

/*
LoadSheetFile
Load a SheetFile from database. As mentioned above, SheetFile has not to be persisted.
In fact, this function loads all Cells of given filename from database. Afterwards,
this function scans over those cells, adding them to SheetFile.Cells, and their Chunk to
SheetFile.Chunks. Besides, this function also set SheetFile.LastAvailableChunk to the
first Chunk whose isAvailable() is true.

This method should only be used to load checkpoints in sqlite. (See GetSheetCellsAll)

@para
	db: a gorm connection. It can be a transaction.
	filename: The validity of filename won't be checked. Caller should guarantee that
	a valid filename is passed in.

@return
	*SheetFile: pointer of loaded SheetFile.
*/
func LoadSheetFile(db *gorm.DB, alloc *datanode_alloc.DataNodeAllocator, filename string) *SheetFile {
	cells := GetSheetCellsAll(db, filename)
	file := &SheetFile{
		Chunks:   map[uint64]*Chunk{},
		Cells:    map[int64]*Cell{},
		filename: filename,
		alloc:    alloc,
	}
	for _, cell := range cells {
		// SheetName is ignored by gorm, not persist to sqlite
		// However it's necessary to persist cell later
		cell.SheetName = filename
		file.Cells[cell.CellID] = cell
		_, ok := file.Chunks[cell.ChunkID]
		// config.MaxCellsPerChunk cells will be stored in the same Chunk at most.
		// So Chunk of a Cell may have been added to file.Chunks.
		// To avoid expensive database operations, we first check whether cell.ChunkID
		// has been loaded or not.
		if !ok {
			dataChunk := loadChunkForFile(db, filename, cell.ChunkID)
			file.Chunks[cell.ChunkID] = dataChunk
			// SheetFile.WriteCellChunk guarantees that it always fulfill currently
			// available Chunk before allocating a new one. So the first Chunk whose
			// isAvailable() should be the LastAvailableChunk.
			if file.LastAvailableChunk == nil && dataChunk.isAvailable(config.MaxBytesPerCell) {
				file.LastAvailableChunk = dataChunk
			}
		}
	}
	return file
}

/*
GetAllChunks
Returns the Snapshot of all Chunks.

@return
	[]*Chunk: slice of pointers pointing to snapshots of s.Chunks.
*/
func (s *SheetFile) GetAllChunks() []*Chunk {
	s.mu.RLock()
	defer s.mu.RUnlock()
	chunks := make([]*Chunk, len(s.Chunks))
	i := 0
	for _, c := range s.Chunks {
		chunks[i] = c.Snapshot()
		i++
	}
	return chunks
}

/*
GetCellChunk
Lookup Cell located at (row, col) and its Chunk.

@para
	row: row number of Cell
	col: column number of Cell

@return
	*Cell, *Chunk: corresponding snapshots if no error
	error:
		*errors.CellNotFoundError if row, col passed in is invalid.
*/
func (s *SheetFile) GetCellChunk(row, col uint32) (*Cell, *Chunk, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Compute CellID by row, col and lookup cell by CellID.
	cell := s.Cells[GetCellID(row, col)]
	if cell == nil {
		return nil, nil, file_errors.NewCellNotFoundError(row, col)
	}
	return cell.Snapshot(), s.Chunks[cell.ChunkID].Snapshot(), nil
}

/*
getCellOffset
Compute the offset of a cell which will be added to an available Chunk.
Every Chunk has config.MaxCellsPerChunk slots, each slot occupies
config.MaxBytesPerCell bytes, for storing a single Cell.
So the offset is the 0-indexed index of the new Cell in the Chunk times
config.MaxBytesPerCell.
*/
func (s *SheetFile) getCellOffset(chunk *Chunk) uint64 {
	return (uint64(len(chunk.Cells))) * config.MaxBytesPerCell
}

/*
addCellToLastAvailable
Add a new cell with given maximum size located at (row,col) to s.LastAvailableChunk.

@para
	row: row number
	col: column number
	size: maximum size of new Cell

@return
	*Cell: pointer of new Cell
*/
func (s *SheetFile) addCellToLastAvailable(row, col uint32, size uint64) *Cell {
	cell := NewCell(GetCellID(row, col), s.getCellOffset(s.LastAvailableChunk),
		size, s.LastAvailableChunk.ID, s.filename)
	s.Cells[cell.CellID] = cell
	// Add new cell to cells of chunk
	s.LastAvailableChunk.Cells = append(s.LastAvailableChunk.Cells, cell)
	// Increase Version of LastAvailableChunk because new Cell is added.
	s.LastAvailableChunk.Version += 1
	return cell
}

/*
WriteCellChunk
Performs necessary metadata mutations to handle an operation of writing data to a Cell.

@para
	row, col: row number, column number of Cell to write
	tx: a gorm connection, can be a transaction

@return
	*Cell, *Chunk: snapshots of the Cell and its Chunk to be written.
	error:
		*errors.NoDataNodeError if there is no DataNode registered.
*/
func (s *SheetFile) WriteCellChunk(row, col uint32, tx *gorm.DB) (*Cell, *Chunk, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cell := s.Cells[GetCellID(row, col)]
	// Lookup an existing Cell by CellID first
	if cell != nil {
		// For existing Cell, just increase its Chunk version
		dataChunk := s.Chunks[cell.ChunkID]
		dataChunk.Version += 1
		return cell.Snapshot(), dataChunk.Snapshot(), nil
	} else {
		// Generally, the size of the new Cell is config.MaxBytesPerCell. However, for
		// the MetaCell defined by (config.SheetMetaCellRow,config.SheetMetaCellCol),
		// a whole chunk should be granted to store metadata of a sheet.
		newCellSize := config.MaxBytesPerCell
		if row == config.SheetMetaCellRow && col == config.SheetMetaCellCol {
			newCellSize = config.BytesPerChunk
		}
		// For a new Cell, tries to add it to s.LastAvailableChunk
		if s.LastAvailableChunk != nil && s.LastAvailableChunk.isAvailable(newCellSize) {
			// There is a empty slot for the new Cell, add it to s.LastAvailableChunk
			cell = s.addCellToLastAvailable(row, col, newCellSize)
			return cell.Snapshot(), s.LastAvailableChunk.Snapshot(), nil
		} else {
			// s.LastAvailableChunk has been fulfilled, allocate a new Chunk, and
			// makes it s.LastAvailableChunk.
			datanode, err := s.alloc.AllocateNode()
			if err != nil {
				// If there is no DataNode, *errors.NoDataNodeError will be returned.
				return nil, nil, err
			}
			newChunk := &Chunk{
				DataNode: datanode,
				Version:  0,
				Cells:    []*Cell{},
			}
			// newChunk.ID is allocated by sqlite(autoIncrement). So it's required to persist
			// newChunk here to get newChunk.ID.
			newChunk.Persistent(tx)
			// Add the newChunk to Chunks collection of s.
			s.Chunks[newChunk.ID] = newChunk
			// Let newChunk to be s.LastAvailableChunk
			s.LastAvailableChunk = newChunk
			// Re-use logic of addCellToLastAvailable
			cell = s.addCellToLastAvailable(row, col, newCellSize)
			return cell.Snapshot(), newChunk.Snapshot(), nil
		}
	}
}

/*
Persistent
Flush the Cell and Chunk data stored in a SheetFile to sqlite.

@para
	db: a gorm connection. It's supposed to be a transaction.

@return
	error: always nil currently. But potentially errors may be introduced
	in the future.
*/
func (s *SheetFile) Persistent(tx *gorm.DB) error {
	err := tx.Transaction(s.persistentData)
	if err != nil {
		return err
	}
	return nil
}

/*
persistentStructure
Creates the table required to store Cells of a SheetFile.
This method should only be called once when a SheetFile is created.

@para
	db: a gorm connection. Creating a table in a sqlite transaction is
	not allowed, so it can't be a transaction.

@return
	error: errors during creation of the Cell table.
*/
func (s *SheetFile) persistentStructure(db *gorm.DB) error {
	err := CreateCellTableIfNotExists(db, s.filename)
	if err != nil {
		return err
	}
	return nil
}

/*
persistentData
Flush the Cell and Chunk data stored in a SheetFile to sqlite.

@para
	db: a gorm connection. It is supposed to be a transaction.

@return
	error: always nil currently. But potentially errors may be introduced
	in the future.
*/
func (s *SheetFile) persistentData(tx *gorm.DB) error {
	if len(s.Cells) == 0 {
		return nil
	}

	for _, cell := range s.Cells {
		cell.Persistent(tx)
	}
	for _, dataChunk := range s.Chunks {
		dataChunk.Persistent(tx)
	}
	return nil
}
