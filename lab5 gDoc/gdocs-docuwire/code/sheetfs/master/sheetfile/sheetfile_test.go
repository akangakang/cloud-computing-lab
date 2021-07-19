package sheetfile

import (
	ctx "context"
	"github.com/fourstring/sheetfs/master/config"
	"github.com/fourstring/sheetfs/master/datanode_alloc"
	"github.com/fourstring/sheetfs/master/filemgr/file_errors"
	"github.com/fourstring/sheetfs/tests"
	. "github.com/smartystreets/goconvey/convey"
	"sync"
	"sync/atomic"
	"testing"
	"text/template"
)

func TestSheetFile_DynamicPersistent(t *testing.T) {
	Convey("Create test database", t, func() {
		db, err := tests.GetTestDB(&Chunk{})
		So(err, ShouldBeNil)
		Convey("Create test chunks", func() {
			chunk0 := &Chunk{
				DataNode: "0",
				Version:  0,
			}
			chunk1 := &Chunk{
				DataNode: "1",
				Version:  0,
			}
			db.Create(chunk0)
			db.Create(chunk1)
			Convey("Test dynamic table name", func() {
				sheet0 := &SheetFile{
					Chunks: map[uint64]*Chunk{chunk0.ID: chunk0},
					Cells: map[int64]*Cell{
						GetCellID(0, 0): {
							CellID:    GetCellID(0, 0),
							Offset:    0,
							Size:      0,
							ChunkID:   chunk0.ID,
							SheetName: "sheet0",
						},
						GetCellID(0, 1): {
							CellID:    GetCellID(0, 1),
							Offset:    config.MaxBytesPerCell,
							Size:      0,
							ChunkID:   chunk0.ID,
							SheetName: "sheet0",
						},
					},
					filename: "sheet0",
				}
				sheet1 := &SheetFile{
					Chunks: map[uint64]*Chunk{chunk1.ID: chunk1},
					Cells: map[int64]*Cell{
						GetCellID(0, 0): {
							CellID:    GetCellID(0, 0),
							Offset:    0,
							Size:      0,
							ChunkID:   chunk1.ID,
							SheetName: "sheet1",
						},
						GetCellID(0, 1): {
							CellID:    GetCellID(0, 1),
							Offset:    config.MaxBytesPerCell,
							Size:      0,
							ChunkID:   chunk1.ID,
							SheetName: "sheet1",
						},
					},
					filename: "sheet1",
				}
				err = sheet0.persistentStructure(db)
				So(err, ShouldBeNil)
				err = sheet1.persistentStructure(db)
				So(err, ShouldBeNil)
				err = sheet0.Persistent(db)
				So(err, ShouldBeNil)
				err = sheet1.Persistent(db)
				So(err, ShouldBeNil)
				sheet0Cells := GetSheetCellsAll(db, "sheet0")
				So(len(sheet0Cells), ShouldEqual, 2)
				sheet1Cells := GetSheetCellsAll(db, "sheet1")
				So(len(sheet1Cells), ShouldEqual, 2)
			})
		})
	})
}

func TestSheetFile_addCellToLastAvailable(t *testing.T) {
	Convey("Construct test chunk and file", t, func() {
		db, err := tests.GetTestDB(&Chunk{})
		So(err, ShouldBeNil)
		chunk := &Chunk{DataNode: "1", Version: 0, Cells: []*Cell{}}
		chunk.Persistent(db)
		sheet := &SheetFile{
			Chunks: map[uint64]*Chunk{
				chunk.ID: chunk,
			},
			Cells:              map[int64]*Cell{},
			LastAvailableChunk: chunk,
		}
		Convey("Add cells to chunk", func() {
			sheet.addCellToLastAvailable(0, 0, config.MaxBytesPerCell)
			sheet.addCellToLastAvailable(1, 1, config.MaxBytesPerCell)
			sheet.addCellToLastAvailable(2, 2, config.MaxBytesPerCell)
			So(sheet.LastAvailableChunk.isAvailable(config.MaxBytesPerCell), ShouldEqual, true)
			sheet.addCellToLastAvailable(3, 3, config.MaxBytesPerCell)
			So(sheet.LastAvailableChunk.isAvailable(config.MaxBytesPerCell), ShouldEqual, false)
			So(sheet.LastAvailableChunk.Version, ShouldEqual, 4)
		})
	})
}

func TestCreateSheetFile(t *testing.T) {
	Convey("Get test db", t, func() {
		db, err := tests.GetTestDB(&Chunk{})
		So(err, ShouldBeNil)
		alloc := datanode_alloc.NewDataNodeAllocator()
		Convey("Create sheetfile when no datanode registered", func() {
			_, _, _, err := CreateSheetFile(db, alloc, "errfile")
			So(err, ShouldBeError, &datanode_alloc.NoDataNodeError{})
		})
		Convey("Add a datanode", func() {
			alloc.AddDataNode("node1")
			backup_create_tmpl := create_tmpl
			Convey("Create SheetFile using ill-formed SQL", func() {
				create_tmpl, err = template.New("ill-formed SQL").Parse("ill-formed SQL {{ .Name}}")
				So(err, ShouldBeNil)
				_, _, _, err = CreateSheetFile(db, alloc, "ill-file")
				So(err, ShouldBeError)
			})
			create_tmpl = backup_create_tmpl
			Convey("Create sheetfile and verify invariants", func() {
				file, _, _, err := CreateSheetFile(db, alloc, "sheet0")
				So(err, ShouldBeNil)
				So(len(file.Cells), ShouldEqual, 1)
				So(len(file.Chunks), ShouldEqual, 1)
				So(file.LastAvailableChunk, ShouldBeNil)
				metaCell := file.Cells[config.SheetMetaCellID]
				// metaChunk is the first Chunk in testing DB, so its ID is 1
				metaChunk := file.Chunks[1]
				// check metaCell
				So(metaCell.CellID, ShouldEqual, config.SheetMetaCellID)
				So(metaCell.Size, ShouldEqual, config.BytesPerChunk)
				So(metaCell.Offset, ShouldEqual, 0)
				// check metaChunk
				So(metaChunk.Version, ShouldEqual, 0)
				So(metaChunk.isAvailable(config.MaxBytesPerCell), ShouldEqual, false)
				// check relationship between metaCell and metaChunk
				So(metaCell.ChunkID, ShouldEqual, metaChunk.ID)
				So(len(metaChunk.Cells), ShouldEqual, 1)
				So(metaChunk.Cells[0].ID, ShouldEqual, metaCell.ID)
			})
		})
	})
}

func TestSheetFile_GetCellChunk(t *testing.T) {
	Convey("Create test file and datanode", t, func() {
		db, err := tests.GetTestDB(&Chunk{})
		So(err, ShouldBeNil)
		alloc := datanode_alloc.NewDataNodeAllocator()
		alloc.AddDataNode("node1")
		file, _, _, err := CreateSheetFile(db, alloc, "sheet0")
		So(err, ShouldBeNil)
		Convey("Get non-exist cell", func() {
			cell, chunk, err := file.GetCellChunk(0, 0)
			So(cell, ShouldBeNil)
			So(chunk, ShouldBeNil)
			So(err, ShouldBeError, file_errors.NewCellNotFoundError(0, 0))
		})
		Convey("Get MetaCell", func() {
			cell, chunk, err := file.GetCellChunk(config.SheetMetaCellRow, config.SheetMetaCellCol)
			So(err, ShouldBeNil)
			So(cell.IsMeta(), ShouldBeTrue)
			So(cell.Size, ShouldEqual, config.BytesPerChunk)
			So(cell.ChunkID, ShouldEqual, chunk.ID)
		})
	})
}

func TestSheetFile_WriteCellChunk_GetAllChunks(t *testing.T) {
	Convey("Create test file and datanode", t, func() {
		db, err := tests.GetTestDB(&Chunk{})
		So(err, ShouldBeNil)
		alloc := datanode_alloc.NewDataNodeAllocator()
		alloc.AddDataNode("node1")
		file, _, _, err := CreateSheetFile(db, alloc, "sheet0")
		So(err, ShouldBeNil)
		Convey("Write to MetaCell", func() {
			cell, chunk, err := file.WriteCellChunk(config.SheetMetaCellRow, config.SheetMetaCellCol, db)
			So(err, ShouldBeNil)
			So(chunk.Version, ShouldEqual, 1)
			So(cell.IsMeta(), ShouldBeTrue)
			So(cell.ChunkID, ShouldEqual, chunk.ID)
		})
		Convey("Write to non-exist cell", func() {
			// First write will create a chunk due to no LastAvailable
			cell, chunk, err := file.WriteCellChunk(0, 0, db)
			So(err, ShouldBeNil)
			So(*cell, shouldBeSameCell, Cell{
				CellID:  0,
				Offset:  0,
				Size:    config.MaxBytesPerCell,
				ChunkID: chunk.ID,
			})
			So(file.LastAvailableChunk.ID, ShouldEqual, chunk.ID)
			So(chunk.Version, ShouldEqual, 1)
			// fulfill newly allocated chunk
			for i := uint32(1); i < 4; i++ {
				cell, chunk, err = file.WriteCellChunk(i, i, db)
				So(err, ShouldBeNil)
				So(*cell, shouldBeSameCell, Cell{
					CellID:  GetCellID(i, i),
					Offset:  uint64(i) * config.MaxBytesPerCell,
					Size:    config.MaxBytesPerCell,
					ChunkID: chunk.ID,
				})
				So(chunk.Version, ShouldEqual, uint64(i)+1)
			}
			So(file.LastAvailableChunk.ID, ShouldEqual, chunk.ID)
			// This write should make file to allocate a new Chunk again
			last_chunk := chunk
			cell, chunk, err = file.WriteCellChunk(4, 4, db)
			So(err, ShouldBeNil)
			So(*cell, shouldBeSameCell, Cell{
				CellID:  GetCellID(4, 4),
				Offset:  0,
				Size:    config.MaxBytesPerCell,
				ChunkID: chunk.ID,
			})
			So(file.LastAvailableChunk.ID, ShouldNotEqual, last_chunk.ID)
			So(file.LastAvailableChunk.ID, ShouldEqual, chunk.ID)
			Convey("Test GetAllChunks", func() {
				chunks := file.GetAllChunks()
				So(len(chunks), ShouldEqual, 3)
			})
		})
	})
}

// TODO: change assertions here to config.MaxCellsPerChunk-agnostic
func TestLoadSheetFile(t *testing.T) {
	Convey("Create and persist test file", t, func() {
		db, err := tests.GetTestDB(&Chunk{})
		So(err, ShouldBeNil)
		alloc := datanode_alloc.NewDataNodeAllocator()
		alloc.AddDataNode("node1")
		file, _, _, err := CreateSheetFile(db, alloc, "sheet0")
		So(err, ShouldBeNil)
		for i := uint32(0); i < 10; i++ {
			_, _, err := file.WriteCellChunk(i, i, db)
			So(err, ShouldBeNil)
		}
		err = file.Persistent(db)
		So(err, ShouldBeNil)
		file = LoadSheetFile(db, alloc, "sheet0")
		// 10 normal cell and 1 MetaCell
		So(len(file.Cells), ShouldEqual, 11)
		So(len(file.Chunks), ShouldEqual, 4)
		for i := uint64(1); i <= 4; i++ {
			chunk := file.Chunks[i]
			switch i {
			case 1:
				So(len(chunk.Cells), ShouldEqual, 1)
				So(chunk.Cells[0].IsMeta(), ShouldEqual, true)
				So(chunk.isAvailable(config.MaxBytesPerCell), ShouldEqual, false)
				continue
			case 4:
				So(len(chunk.Cells), ShouldEqual, 2)
				So(chunk.isAvailable(config.MaxBytesPerCell), ShouldEqual, true)
			default:
				So(len(chunk.Cells), ShouldEqual, 4)
				So(chunk.isAvailable(config.MaxBytesPerCell), ShouldEqual, false)
			}
			for _, cell := range chunk.Cells {
				So(cell.ChunkID, ShouldEqual, chunk.ID)
			}
		}
		So(file.LastAvailableChunk.ID, ShouldEqual, 4)
	})
}

func TestSheetFile_Concurrency1(t *testing.T) {
	Convey("Create test file", t, func() {
		db, err := tests.GetTestDB(&Chunk{})
		So(err, ShouldBeNil)
		alloc := datanode_alloc.NewDataNodeAllocator()
		alloc.AddDataNode("node1")
		file, _, _, err := CreateSheetFile(db, alloc, "sheet0")
		Convey("Write to cells concurrently", func(c C) {
			// record expected Version after operation
			expectedVersions := map[uint64]*uint64{}
			var wg sync.WaitGroup
			startRow, endRow := 0, 20
			startCol, endCol := 0, 20
			// +1 for MetaCell
			maxChunks := uint64(tests.DivRoundUp((endRow-startRow)*(endCol-startCol), config.MaxCellsPerChunk)) + 1
			for i := uint64(1); i < maxChunks+1; i++ {
				expectedVersions[i] = new(uint64)
			}
			worker := func() {
				defer wg.Done()
				for i := 0; i < 100; i++ {
					row := uint32(tests.RandInt(startRow, endRow))
					col := uint32(tests.RandInt(startCol, endCol))
					_, chunk, err := file.WriteCellChunk(row, col, db)
					c.So(err, ShouldBeNil)
					atomic.AddUint64(expectedVersions[chunk.ID], 1)
				}
			}
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go worker()
			}
			wg.Wait()
			for _, chunk := range file.Chunks {
				So(chunk.Version, ShouldEqual, *expectedVersions[chunk.ID])
			}
		})
	})
}

func TestSheetFile_Concurrency2(t *testing.T) {
	Convey("Create test file", t, func() {
		db, err := tests.GetTestDB(&Chunk{})
		So(err, ShouldBeNil)
		alloc := datanode_alloc.NewDataNodeAllocator()
		alloc.AddDataNode("node1")
		file, _, _, err := CreateSheetFile(db, alloc, "sheet0")
		Convey("Read and write concurrently", func(c C) {
			// record expected Version after operation
			expectedVersions := map[uint64]*uint64{}
			totalCells := 0
			var rwg, wwg sync.WaitGroup
			var mu sync.RWMutex
			readerCtx, cancelReaders := ctx.WithCancel(ctx.Background())
			startRow, endRow := 0, 20
			startCol, endCol := 0, 20
			// +1 for MetaCell
			maxChunks := uint64(tests.DivRoundUp((endRow-startRow)*(endCol-startCol), config.MaxCellsPerChunk)) + 1

			for i := uint64(1); i < maxChunks+1; i++ {
				expectedVersions[i] = new(uint64)
			}
			// For MetaCell
			*expectedVersions[1] = 1

			reader := func() {
				defer rwg.Done()
				for {
					select {
					case <-readerCtx.Done():
						return
					default:
						row := uint32(tests.RandInt(startRow, endRow))
						col := uint32(tests.RandInt(startCol, endCol))
						_, chunk, err := file.GetCellChunk(row, col)
						_, ok := err.(*file_errors.CellNotFoundError)
						if ok {
							continue
						}
						c.So(err, ShouldBeNil)
						// Writers can preempt here, so when reader gets mu,
						// Some writers may have written to this cell again,
						// increased version number
						mu.RLock()
						c.So(chunk.Version, ShouldBeLessThanOrEqualTo, *expectedVersions[chunk.ID])
						mu.RUnlock()

						chunks := file.GetAllChunks()
						mu.RLock()
						// <= same as above
						// +1 for MetaCell
						c.So(len(chunks), ShouldBeLessThanOrEqualTo, tests.DivRoundUp(totalCells, config.MaxCellsPerChunk)+1)
						mu.RUnlock()
					}
				}
			}

			writer := func(row uint32) {
				defer wwg.Done()
				for i := 0; i < endCol-startCol; i++ {
					mu.Lock()
					_, chunk, err := file.WriteCellChunk(row, uint32(i), db)
					c.So(err, ShouldBeNil)
					totalCells += 1
					*expectedVersions[chunk.ID] += 1
					mu.Unlock()
				}
			}
			for i := 0; i < endRow-startRow; i++ {
				wwg.Add(1)
				go writer(uint32(i))
			}
			for i := 0; i < 20; i++ {
				rwg.Add(1)
				go reader()
			}
			wwg.Wait()
			cancelReaders()
			rwg.Wait()
		})
	})
}
