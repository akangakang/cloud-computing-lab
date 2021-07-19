package checkpoint

import (
	"errors"
	"github.com/fourstring/sheetfs/master/model"
	"gorm.io/gorm"
)

type Checkpoint struct {
	model.Model
	StartOffset int64
}

func getCheckpointInDB(db *gorm.DB) *Checkpoint {
	var ckpt Checkpoint
	result := db.First(&ckpt, 1)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		ckpt.ID = 1
		ckpt.StartOffset = 0
		db.Create(&ckpt)
	}
	return &ckpt
}

func RecordCheckpoint(db *gorm.DB, newStartOffset int64) error {
	ckpt := getCheckpointInDB(db)
	ckpt.StartOffset = newStartOffset
	db.Save(ckpt)
	return nil
}

func ReadCheckpoint(db *gorm.DB) int64 {
	ckpt := getCheckpointInDB(db)
	return ckpt.StartOffset
}
