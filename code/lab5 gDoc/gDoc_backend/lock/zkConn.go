package lock

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

var ZkConn *zk.Conn
var RWMutex sync.RWMutex

func ZkConnInit() {
	hosts := []string{"192.168.12.140:2182"}
	// 连接zk
	var err error

	ZkConn, _, err = zk.Connect(hosts, time.Second*5)

	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("zookeeper connect succ")

	println(ZkConn.Server())
}

func ZkLockFile(filename string) (lock_file *zk.Lock) {
	filepath := "/" + filename
	lock_file = zk.NewLock(ZkConn, filepath, zk.WorldACL(zk.PermAll))
	err_file_lock := lock_file.Lock()
	if err_file_lock != nil {
		panic(err_file_lock)
	}
	fmt.Printf("[Lock file Succ]:%s\n", filepath)

	return
}

func ZkLockCell(row int, column int, filename string) (lock_cell *zk.Lock) {
	lock_file := ZkLockFile(filename)

	cellpath := "/" + filename + "-" + strconv.Itoa(row) + "-" + strconv.Itoa(column)
	lock_cell = zk.NewLock(ZkConn, cellpath, zk.WorldACL(zk.PermAll))
	err_cell_lock := lock_cell.Lock()
	if err_cell_lock != nil {
		panic(err_cell_lock)
	}
	fmt.Printf("[Lock cell Succ]:%s\n", cellpath)

	ZkUnlockFile(lock_file)
	return
}

func ZkUnlockCell(lock_cell *zk.Lock) {
	lock_cell.Unlock()
	fmt.Printf("[Unlock cell Succ]\n")
}

func ZkUnlockFile(lock_file *zk.Lock) {
	lock_file.Unlock()
	fmt.Printf("[Unlock file Succ]\n")
}
