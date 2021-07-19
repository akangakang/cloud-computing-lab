package mgr_entry

import (
	"gorm.io/gorm"
	"time"
)

/*
MapEntry
Represents a 'directory entry' of FileManager. Every entry maps a FileName
to a CellsTableName which is the name of sqlite table storing Cells
of the mapped SheetFile.

Recycled is a flag indicates that whether the mapped file has been moved to
'recycle bin' or not. When a file is recycled, time of this operation is recorded
in the RecycledAt field. Recycled files will be permanently after a period of time,
before they are deleted, they can be resumed.
*/
type MapEntry struct {
	gorm.Model
	FileName       string `gorm:"index"`
	CellsTableName string
	Recycled       bool
	RecycledAt     time.Time
}
