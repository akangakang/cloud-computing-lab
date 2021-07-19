package server

import (
	goctx "context"
	"fmt"
	"github.com/fourstring/sheetfs/master/config"
	"github.com/fourstring/sheetfs/master/datanode_alloc"
	"github.com/fourstring/sheetfs/master/filemgr"
	"github.com/fourstring/sheetfs/master/filemgr/mgr_entry"
	"github.com/fourstring/sheetfs/master/journal/checkpoint"
	"github.com/fourstring/sheetfs/master/sheetfile"
	fs_rpc "github.com/fourstring/sheetfs/protocol"
	"github.com/fourstring/sheetfs/tests"
	. "github.com/smartystreets/goconvey/convey"
	"sort"
	"testing"
)

var ctx = goctx.Background()

func newTestServer() (*Server, error) {
	db, err := tests.GetTestDB(&mgr_entry.MapEntry{}, &sheetfile.Chunk{}, &checkpoint.Checkpoint{})
	if err != nil {
		return nil, err
	}
	alloc := datanode_alloc.NewDataNodeAllocator()
	alloc.AddDataNode("node1")
	fm := filemgr.LoadFileManager(db, alloc, nil)
	s, err := NewServer(fm, alloc)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func shouldBeSameChunk(actual interface{}, expected ...interface{}) string {
	ac, ok := actual.(*fs_rpc.Chunk)
	if !ok {
		return "actual not a Chunk!"
	}
	ec, ok := expected[0].(*fs_rpc.Chunk)
	if !ok {
		return "expected not a Chunk!"
	}

	if ac.Datanode == ec.Datanode && ac.Version == ec.Version && ac.HoldsMeta == ec.HoldsMeta {
		return ""
	} else {
		return fmt.Sprintf("actual %v, expected %v", ac, ec)
	}
}

func shouldBeSameCell(actual interface{}, expected ...interface{}) string {
	ac, ok := actual.(*fs_rpc.Cell)
	if !ok {
		return "actual not a Cell!"
	}
	ec, ok := expected[0].(*fs_rpc.Cell)
	if !ok {
		return "expected not a Cell!"
	}

	if ac.Size == ec.Size && ac.Offset == ec.Offset && shouldBeSameChunk(ac.Chunk, ec.Chunk) == "" {
		return ""
	} else {
		return fmt.Sprintf("actual %v, expected %v", ac, ec)
	}
}

func TestServer_RegisterDataNode(t *testing.T) {
	Convey("Build test server", t, func() {
		db, err := tests.GetTestDB(&mgr_entry.MapEntry{}, &sheetfile.Chunk{}, &checkpoint.Checkpoint{})
		So(err, ShouldBeNil)
		alloc := datanode_alloc.NewDataNodeAllocator()
		fm := filemgr.LoadFileManager(db, alloc, nil)
		s, err := NewServer(fm, alloc)
		So(err, ShouldBeNil)
		Convey("Call rpc method", func() {
			rep, err := s.RegisterDataNode(ctx, &fs_rpc.RegisterDataNodeRequest{Addr: "node1"})
			So(err, ShouldBeNil)
			So(rep.Status, ShouldEqual, fs_rpc.Status_OK)
			Convey("Allocate data nodes", func() {
				node, err := alloc.AllocateNode()
				So(err, ShouldBeNil)
				So(node, ShouldEqual, "node1")
			})
		})
	})
}

func TestServer_CreateSheet(t *testing.T) {
	Convey("Build test server", t, func() {
		s, err := newTestServer()
		So(err, ShouldBeNil)
		Convey("Create file", func() {
			rep, err := s.CreateSheet(ctx, &fs_rpc.CreateSheetRequest{Filename: "sheet0"})
			So(err, ShouldBeNil)
			So(rep.Status, ShouldEqual, fs_rpc.Status_OK)
			So(rep.Fd, ShouldEqual, 0)
			Convey("Create existed file", func() {
				rep, err := s.CreateSheet(ctx, &fs_rpc.CreateSheetRequest{Filename: "sheet0"})
				So(err, ShouldBeNil)
				So(rep.Status, ShouldEqual, fs_rpc.Status_Exist)
			})
		})
	})
}

func TestServer_ReadCell(t *testing.T) {
	Convey("Build test server", t, func() {
		s, err := newTestServer()
		So(err, ShouldBeNil)
		Convey("Create test file", func() {
			rep, err := s.CreateSheet(ctx, &fs_rpc.CreateSheetRequest{Filename: "sheet0"})
			So(err, ShouldBeNil)
			So(rep.Status, ShouldEqual, fs_rpc.Status_OK)
			Convey("Read test sheet MetaCell", func() {
				rep, err := s.ReadCell(ctx, &fs_rpc.ReadCellRequest{
					Fd:     rep.Fd,
					Row:    config.SheetMetaCellRow,
					Column: config.SheetMetaCellCol,
				})
				So(err, ShouldBeNil)
				So(rep.Status, ShouldEqual, fs_rpc.Status_OK)
				So(rep.Cell, shouldBeSameCell, &fs_rpc.Cell{
					Chunk: &fs_rpc.Chunk{
						Id:        1,
						Datanode:  "node1",
						Version:   0,
						HoldsMeta: true,
					},
					Offset: 0,
					Size:   config.BytesPerChunk,
				})
			})
			Convey("Read non-exist cell", func() {
				rep, err := s.ReadCell(ctx, &fs_rpc.ReadCellRequest{
					Fd:     rep.Fd,
					Row:    0,
					Column: 0,
				})
				So(err, ShouldBeNil)
				So(rep.Status, ShouldEqual, fs_rpc.Status_Invalid)
			})
			Convey("Read non-exits fd", func() {
				rep, err := s.ReadCell(ctx, &fs_rpc.ReadCellRequest{
					Fd:     0xdeafbeef,
					Row:    config.SheetMetaCellRow,
					Column: config.SheetMetaCellCol,
				})
				So(err, ShouldBeNil)
				So(rep.Status, ShouldEqual, fs_rpc.Status_NotFound)
			})
		})
	})
}

func TestServer_WriteCell(t *testing.T) {
	Convey("Build test server", t, func() {
		s, err := newTestServer()
		So(err, ShouldBeNil)
		Convey("Create test file", func() {
			rep, err := s.CreateSheet(ctx, &fs_rpc.CreateSheetRequest{Filename: "sheet0"})
			So(err, ShouldBeNil)
			So(rep.Status, ShouldEqual, fs_rpc.Status_OK)
			Convey("Write to test sheet", func() {
				for i := uint32(0); i < 10; i++ {
					rep, err := s.WriteCell(ctx, &fs_rpc.WriteCellRequest{
						Fd:     rep.Fd,
						Row:    i,
						Column: i,
					})
					So(err, ShouldBeNil)
					So(rep.Status, ShouldEqual, fs_rpc.Status_OK)
					So(rep.Cell, shouldBeSameCell, &fs_rpc.Cell{
						Chunk: &fs_rpc.Chunk{
							Id:        uint64(2 + i/4),
							Datanode:  "node1",
							Version:   uint64(1 + (i % 4)),
							HoldsMeta: false,
						},
						Offset: (uint64(i) % 4) * config.MaxBytesPerCell,
						Size:   config.MaxBytesPerCell,
					})
				}
				Convey("Read cells written", func() {
					for i := uint32(0); i < 10; i++ {
						rep, err := s.ReadCell(ctx, &fs_rpc.ReadCellRequest{
							Fd:     rep.Fd,
							Row:    i,
							Column: i,
						})
						So(err, ShouldBeNil)
						So(rep.Status, ShouldEqual, fs_rpc.Status_OK)
						finalVersion := uint64(4)
						if i >= 8 {
							finalVersion = 2
						}
						So(rep.Cell, shouldBeSameCell, &fs_rpc.Cell{
							Chunk: &fs_rpc.Chunk{
								Id:        uint64(2 + i/4),
								Datanode:  "node1",
								Version:   finalVersion,
								HoldsMeta: false,
							},
							Offset: (uint64(i) % 4) * config.MaxBytesPerCell,
							Size:   config.MaxBytesPerCell,
						})
					}
				})
			})
		})
	})
}

func TestServer_ReadSheet(t *testing.T) {
	Convey("Build test server", t, func() {
		s, err := newTestServer()
		So(err, ShouldBeNil)
		Convey("Create test file", func() {
			rep, err := s.CreateSheet(ctx, &fs_rpc.CreateSheetRequest{Filename: "sheet0"})
			So(err, ShouldBeNil)
			So(rep.Status, ShouldEqual, fs_rpc.Status_OK)
			fd := rep.Fd
			Convey("Write to test sheet", func() {
				for i := uint32(0); i < 10; i++ {
					rep, err := s.WriteCell(ctx, &fs_rpc.WriteCellRequest{
						Fd:     fd,
						Row:    i,
						Column: i,
					})
					So(err, ShouldBeNil)
					So(rep.Status, ShouldEqual, fs_rpc.Status_OK)
				}
				Convey("Read test sheet", func() {
					rep, err := s.ReadSheet(ctx, &fs_rpc.ReadSheetRequest{
						Fd: fd,
					})
					So(err, ShouldBeNil)
					So(rep.Status, ShouldEqual, fs_rpc.Status_OK)
					So(len(rep.Chunks), ShouldEqual, 4)
					for _, chunk := range rep.Chunks {
						switch chunk.Id {
						case 1: // For MetaCell
							So(chunk, shouldBeSameChunk, &fs_rpc.Chunk{
								Id:        1,
								Datanode:  "node1",
								Version:   0,
								HoldsMeta: true,
							})
						case 4:
							So(chunk, shouldBeSameChunk, &fs_rpc.Chunk{
								Id:        4,
								Datanode:  "node1",
								Version:   2,
								HoldsMeta: false,
							})
						default:
							So(chunk, shouldBeSameChunk, &fs_rpc.Chunk{
								Id:        chunk.Id,
								Datanode:  "node1",
								Version:   4,
								HoldsMeta: false,
							})
						}
					}
				})
			})
		})
	})
}

func TestServer_RecycleResumeSheet(t *testing.T) {
	Convey("Build test server", t, func() {
		s, err := newTestServer()
		So(err, ShouldBeNil)
		Convey("Create test file", func() {
			rep, err := s.CreateSheet(ctx, &fs_rpc.CreateSheetRequest{Filename: "sheet0"})
			So(err, ShouldBeNil)
			So(rep.Status, ShouldEqual, fs_rpc.Status_OK)
			Convey("Recycle and resume test file", func() {
				rep, err := s.RecycleSheet(ctx, &fs_rpc.RecycleSheetRequest{Filename: "sheet0"})
				So(err, ShouldBeNil)
				So(rep.Status, ShouldEqual, fs_rpc.Status_OK)
				rep2, err := s.OpenSheet(ctx, &fs_rpc.OpenSheetRequest{Filename: "sheet0"})
				So(err, ShouldBeNil)
				So(rep2.Status, ShouldEqual, fs_rpc.Status_NotFound)
				rep3, err := s.ResumeSheet(ctx, &fs_rpc.ResumeSheetRequest{Filename: "sheet0"})
				So(err, ShouldBeNil)
				So(rep3.Status, ShouldEqual, fs_rpc.Status_OK)
				rep4, err := s.OpenSheet(ctx, &fs_rpc.OpenSheetRequest{Filename: "sheet0"})
				So(err, ShouldBeNil)
				So(rep4.Status, ShouldEqual, fs_rpc.Status_OK)
				So(rep4.Fd, ShouldEqual, 1)
			})
		})
	})
}

// TODO
func TestServer_DeleteSheet(t *testing.T) {
}

func TestServer_ListSheets(t *testing.T) {
	Convey("Build test server", t, func() {
		s, err := newTestServer()
		So(err, ShouldBeNil)
		Convey("Create test files", func() {
			for i := 0; i < 10; i++ {
				filename := fmt.Sprintf("sheet%d", i)
				rep, err := s.CreateSheet(ctx, &fs_rpc.CreateSheetRequest{Filename: filename})
				So(err, ShouldBeNil)
				So(rep.Status, ShouldEqual, fs_rpc.Status_OK)
				So(rep.Fd, ShouldEqual, i)
				if i%2 == 0 {
					rep, err := s.RecycleSheet(ctx, &fs_rpc.RecycleSheetRequest{Filename: filename})
					So(err, ShouldBeNil)
					So(rep.Status, ShouldEqual, fs_rpc.Status_OK)
				}
			}
			Convey("List test files", func() {
				rep, err := s.ListSheets(ctx, &fs_rpc.Empty{})
				So(err, ShouldBeNil)
				So(rep.Status, ShouldEqual, fs_rpc.Status_OK)
				sheets := rep.Sheets
				sort.Slice(sheets, func(i, j int) bool {
					return sheets[i].Filename < sheets[j].Filename
				})
				for i, sheet := range sheets {
					filename := fmt.Sprintf("sheet%d", i)
					So(sheet.Filename, ShouldEqual, filename)
					So(sheet.Recycled, ShouldEqual, i%2 == 0)
				}
			})
		})
	})
}
