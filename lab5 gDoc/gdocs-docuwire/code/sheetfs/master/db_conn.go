package main

import (
	"fmt"
	"github.com/fourstring/sheetfs/master/filemgr/mgr_entry"
	"github.com/fourstring/sheetfs/master/journal/checkpoint"
	"github.com/fourstring/sheetfs/master/sheetfile"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func connectDB(nodeId string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("%s.db", nodeId)), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&mgr_entry.MapEntry{}, &sheetfile.Chunk{}, &checkpoint.Checkpoint{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
