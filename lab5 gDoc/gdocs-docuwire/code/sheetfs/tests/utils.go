package tests

import (
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"math/rand"
	"time"
)

func GetTestDB(automigrates ...interface{}) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(automigrates...)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func GetPersistTestDB(dbName string, automigrates ...interface{}) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("%s.db", dbName)), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(automigrates...)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func RandInt(a, b int) int {
	rand.Seed(time.Now().UnixNano())
	return a + rand.Intn(b-a)
}

func DivRoundUp(n, d int) int {
	return (n + (d - 1)) / d
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStr(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
