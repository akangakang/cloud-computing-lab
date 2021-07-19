package sheetfile

import (
	"github.com/fourstring/sheetfs/master/config"
	"github.com/fourstring/sheetfs/master/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

/*
Chunk
Represent a fixed-size block of data stored on some DataNode.
The size of a Chunk is given by config.BytesPerChunk.

A Version is maintained by MasterNode and DataNode separately. Latest
Version of a Chunk is stored in MasterNode, and the actual Version is placed
on DataNode. Version is necessary for serializing write operations to a Chunk.
When a client issues a write operation, MasterNode will increase the Version by
1 and return it to client. Client must send both data to write and the Version
to DataNode which actually stores the Chunk. This operation success iff version
in request is equal to Version in DataNode plus 1, by which we achieve serialization
of write operations.
Version can also be utilized to select correct replication of a Chunk when quorums
were introduced.

As to other metadata datastructures, Chunk should be maintained in memory, with the
aid of journaling to tolerate fault, and flushed to sqlite during checkpointing only.
*/
type Chunk struct {
	model.Model
	DataNode string
	Version  uint64
	Cells    []*Cell
}

/*
isAvailable
Returns true if c is available to store a new Cell with given size.
*/
func (c *Chunk) isAvailable(size uint64) bool {
	used := uint64(0)
	for _, cell := range c.Cells {
		used += cell.Size
	}
	remains := config.BytesPerChunk - used
	return size <= remains
}

/*
Persistent
Flush Chunk data in memory into sqlite. But Chunk.Cells is not taken into consideration
because dynamic table names are applied. They should be persisted manually.
This method should be used only for checkpointing, and is supposed to be called
in a transaction for atomicity.
*/
func (c *Chunk) Persistent(tx *gorm.DB) {
	tx.Omit(clause.Associations).Clauses(clause.OnConflict{UpdateAll: true}).Create(c)
}

/*
Snapshot
Returns a *Chunk points to the copy of c.
See SheetFile for the necessity of Snapshot.

@return
	*Chunk points to the copy of c.
*/
func (c *Chunk) Snapshot() *Chunk {
	var nc Chunk
	nc = *c
	for i, cell := range c.Cells {
		nc.Cells[i] = cell.Snapshot()
	}
	return &nc
}

/*
loadChunkForFile
Load a chunk for a sheet with given id from sqlite. And preload all Cells simultaneously.
This function do not check id passed in, so it's not exported. Caller should
check against id.

@para
	tx: a gorm connection, it can be a transaction.
	filename
	id: Chunk.ID

@return
	*Chunk
*/
func loadChunkForFile(tx *gorm.DB, filename string, id uint64) *Chunk {
	var c Chunk
	tx.Preload("Cells", func(db *gorm.DB) *gorm.DB {
		return db.Table(GetCellTableName(filename))
	}).First(&c, id)
	return &c
}
