package sheetfile

import (
	"fmt"
	"github.com/fourstring/sheetfs/master/config"
	"github.com/fourstring/sheetfs/tests"
	. "github.com/smartystreets/goconvey/convey"
	"math"
	"testing"
)

func shouldBeSameCell(actual interface{}, expected ...interface{}) string {
	ac, ok := actual.(Cell)
	if !ok {
		return "actual not a Cell!"
	}
	ec, ok := expected[0].(Cell)
	if !ok {
		return "expected not a Cell!"
	}
	if ac.Size == ec.Size && ac.CellID == ec.CellID && ac.ChunkID == ec.ChunkID && ac.Offset == ec.Offset {
		return ""
	} else {
		return fmt.Sprintf("actual %v, expected %v", ac, ec)
	}
}

func TestGetCellID(t *testing.T) {
	Convey("Should compute cell ID correctly", t, func() {
		So(GetCellID(0, 0), ShouldEqual, 0)
		So(GetCellID(math.MaxUint32, math.MaxUint32), ShouldEqual, config.SheetMetaCellID)
		So(GetCellID(0x7eadbeef, 0x7eadbaaf), ShouldEqual, int64(0x7eadbeef7eadbaaf))
	})
}

func TestCellTableName(t *testing.T) {
	Convey("Construct testing cell", t, func() {
		filename := "test1"
		c1 := NewCell(0, 0, 0, 0, filename)
		Convey("Test cell table name", func() {
			So(GetCellTableName(filename), ShouldEqual, "cells_test1")
			So(c1.TableName(), ShouldEqual, "cells_test1")
		})
	})
}

func TestCell_Snapshot(t *testing.T) {
	Convey("Construct testing cell", t, func() {
		filename := "test1"
		c1 := NewCell(0, 0, 0, 0, filename)
		Convey("Test snapshot", func() {
			s := c1.Snapshot()
			So(s, ShouldNotEqual, c1)
			So(*s, shouldBeSameCell, *c1)
		})
	})
}

func TestCell_Persistent(t *testing.T) {
	Convey("Get test db", t, func() {
		db, err := tests.GetTestDB()
		So(err, ShouldBeNil)
		err = CreateCellTableIfNotExists(db, "sheet0")
		So(err, ShouldBeNil)
		err = db.AutoMigrate(&Chunk{})
		So(err, ShouldBeNil)
		Convey("create test chunk", func() {
			chunk := Chunk{}
			db.Save(&chunk)
			Convey("Create and persist test cell", func() {
				cell := NewCell(0, 0, 0, chunk.ID, "sheet0")
				cell.Persistent(db)
				Convey("Find test cell from db", func() {
					var c1 Cell
					db.Table(GetCellTableName("sheet0")).First(&c1, cell.ID)
					So(c1, shouldBeSameCell, *cell)
				})
			})
		})
	})
}

func TestCell_IsMeta(t *testing.T) {
	Convey("Construct test cells", t, func() {
		cell := NewCell(0, 0, 0, 0, "sheet0")
		metaCell := NewCell(GetCellID(config.SheetMetaCellRow, config.SheetMetaCellCol), 0, 0, 0, "sheet0")
		Convey("test IsMeta", func() {
			So(cell.IsMeta(), ShouldEqual, false)
			So(metaCell.IsMeta(), ShouldEqual, true)
		})
	})
}
