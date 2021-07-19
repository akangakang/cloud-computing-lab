package journal_entry

import (
	"github.com/fourstring/sheetfs/master/filemgr/mgr_entry"
	"github.com/fourstring/sheetfs/master/sheetfile"
	"time"
)

func FromSheetCell(c *sheetfile.Cell) *MasterEntry_Cell {
	return &MasterEntry_Cell{Cell: &CellEntry{
		TargetState: State_PRESENT,
		CellId:      c.CellID,
		Offset:      c.Offset,
		Size:        c.Size,
		ChunkId:     c.ChunkID,
		SheetName:   c.SheetName,
	}}
}

func FromEmptySheetCell() *MasterEntry_E1 {
	return &MasterEntry_E1{E1: &Empty{}}
}

func ToSheetCell(scell *sheetfile.Cell, e *CellEntry) {
	scell.CellID = e.CellId
	scell.Offset = e.Offset
	scell.Size = e.Size
	scell.ChunkID = e.ChunkId
	scell.SheetName = e.SheetName
}

func FromSheetChunk(c *sheetfile.Chunk) *MasterEntry_Chunk {
	return &MasterEntry_Chunk{Chunk: &ChunkEntry{
		TargetState: State_PRESENT,
		Id:          c.ID,
		Version:     c.Version,
		Datanode:    c.DataNode,
	}}
}

func FromEmptyChunk() *MasterEntry_E2 {
	return &MasterEntry_E2{E2: &Empty{}}
}

func ToSheetChunk(schunk *sheetfile.Chunk, e *ChunkEntry) {
	schunk.ID = e.Id
	schunk.Version = e.Version
	schunk.DataNode = e.Datanode
}

func FromMgrEntry(mentry *mgr_entry.MapEntry) *MasterEntry_MapEntry {
	return &MasterEntry_MapEntry{MapEntry: &FileMapEntry{
		TargetState:       State_PRESENT,
		Filename:          mentry.FileName,
		CellsTableName:    mentry.CellsTableName,
		Recycled:          mentry.Recycled,
		RecycledTimestamp: mentry.RecycledAt.UnixNano(),
	}}
}

func FromEmptyMgrEntry() *MasterEntry_E3 {
	return &MasterEntry_E3{E3: &Empty{}}
}

func ToMgrEntry(mentry *mgr_entry.MapEntry, e *FileMapEntry) {
	mentry.FileName = e.Filename
	mentry.CellsTableName = e.CellsTableName
	mentry.Recycled = e.Recycled
	mentry.RecycledAt = time.Unix(0, e.RecycledTimestamp)
}
