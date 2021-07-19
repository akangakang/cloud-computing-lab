package model

import (
	"gorm.io/gorm"
	"time"
)

type Model struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
