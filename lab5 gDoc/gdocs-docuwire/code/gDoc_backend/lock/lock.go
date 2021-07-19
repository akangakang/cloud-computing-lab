package lock

import (
	"strconv"
	"sync"

	"github.com/astaxie/beego"
)

type CellLockMap map[string]*sync.Mutex

var (
	FileLockMap   = make(map[string]CellLockMap)
	FileLock      sync.Mutex
	FileRWLockMap = make(map[string]*sync.RWMutex)
)

const (
	META_ROW = ^uint32(0)
	META_COL = ^uint32(0)
)

/*
 * before create a new file, func AddFileLock should be called
 */
func AddFileLock(filename string) {
	FileLock.Lock()
	if _, ok := FileLockMap[filename]; !ok {
		FileLockMap[filename] = make(map[string]*sync.Mutex)
	}

	if _, ok := FileRWLockMap[filename]; !ok {
		FileRWLockMap[filename] = &sync.RWMutex{}
	}
	FileLock.Unlock()
}

/*
 * when delete a file, func DeleteFileLock should be called
 */
func DeleteFileLock(filename string) {
	FileLock.Lock()
	if _, ok := FileLockMap[filename]; ok {
		delete(FileLockMap, filename)
	}
	if _, ok := FileRWLockMap[filename]; ok {
		delete(FileRWLockMap, filename)
	}
	FileLock.Unlock()
}

func LockCell(filename string, row uint32, column uint32) {
	cell := strconv.FormatUint(uint64(row), 10) + "-" + strconv.FormatUint(uint64(column), 10)
	beego.Debug("[LockCell] RLock", filename, cell)
	if _, ok := FileRWLockMap[filename]; !ok {
		AddFileLock(filename)
	}
	FileRWLockMap[filename].RLock()

	if cellLock, ok := FileLockMap[filename][cell]; ok {
		cellLock.Lock()
	} else {
		FileLock.Lock()
		FileLockMap[filename][cell] = &sync.Mutex{}
		FileLock.Unlock()
		FileLockMap[filename][cell].Lock()
	}

	beego.Debug("[LockCell]", filename, cell)
}

func UnlockCell(filename string, row uint32, column uint32) {
	cell := strconv.FormatUint(uint64(row), 10) + "-" + strconv.FormatUint(uint64(column), 10)
	FileLockMap[filename][cell].Unlock()
	FileRWLockMap[filename].RUnlock()
	beego.Debug("[UnlockCell]", filename, cell)
}

func LockMeta(filename string) {
	LockCell(filename, META_ROW, META_COL)
}

func UnlockMeta(filename string) {
	UnlockCell(filename, META_ROW, META_COL)
}

func LockFile(filename string) {
	beego.Debug("[LockFile]", filename)
	if _, ok := FileRWLockMap[filename]; !ok {
		AddFileLock(filename)
	}
	FileRWLockMap[filename].Lock()
}

func UnlockFile(filename string) {
	beego.Debug("[UnlockFile]", filename)
	if _, ok := FileRWLockMap[filename]; !ok {
		AddFileLock(filename)
	}
	FileRWLockMap[filename].Unlock()
}
