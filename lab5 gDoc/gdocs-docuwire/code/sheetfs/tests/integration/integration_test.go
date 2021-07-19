package integration

import (
	stdctx "context"
	"fmt"
	"github.com/fourstring/sheetfs/fsclient"
	. "github.com/smartystreets/goconvey/convey"
	"os"
	"sync"
	"testing"
	"time"
)

var ctx = stdctx.Background()
var ElectionServers = []string{
	"127.0.0.1:2181",
	"127.0.0.1:2182",
	"127.0.0.1:2183",
}
var MasterAck = "/master-election-ack"
var DatnodeAckPrefix = "/datanode_election_ack_"
var cfg = &fsclient.ClientConfig{
	ZookeeperServers:    ElectionServers,
	ZookeeperTimeout:    10 * time.Second,
	MasterZnode:         MasterAck,
	DataNodeZnodePrefix: DatnodeAckPrefix,
	MaxRetry:            10,
}

func constructData(col uint32, row uint32) []byte {
	return []byte("{\n" +
		"\"c\": " + fmt.Sprint(col) + ",\n" +
		"\"r\": " + fmt.Sprint(row) + ",\n" +
		"\"v\": {\n" +
		"\"ct\": {\"fa\": \"General\",\"t\": \"g\"},\n" +
		"\"m\": \"ww\",\n" +
		"\"v\": \"ww\"\n" +
		"}\n" +
		"}")
}

func TestCreate(t *testing.T) {
	Convey("Start test servers", t, func() {
		c, err := fsclient.NewClient(cfg)
		So(err, ShouldBeNil)
		Convey("Create test file", func() {
			_, err := c.Create(ctx, "test file")
			So(err, ShouldEqual, nil)
		})
	})
}

func TestOpen(t *testing.T) {
	Convey("Start test servers", t, func() {
		c, err := fsclient.NewClient(cfg)
		So(err, ShouldBeNil)
		Convey("Open exist test file", func() {
			c.Create(ctx, "test file")
			_, err := c.Open(ctx, "test file")
			So(err, ShouldEqual, nil)
		})

		Convey("Open non-exist test file", func() {
			file, err := c.Open(ctx, "non-exist file")
			So(err, ShouldNotBeNil)
			So(file, ShouldEqual, nil)
		})
	})
}

func TestReadAndWrite(t *testing.T) {
	Convey("Start test servers", t, func() {
		c, err := fsclient.NewClient(cfg)
		So(err, ShouldBeNil)

		// var file File
		Convey("Read empty file after create", func() {
			file, err := c.Create(ctx, "test file")
			So(err, ShouldBeNil)
			read, _, _ := file.Read(ctx) // must call this before write

			header := []byte("{\"celldata\": []}")
			So(read[:len(header)], ShouldResemble, header)

			// read := make([]byte, 1024)
			b := []byte("this is test")

			size, err := file.WriteAt(ctx, b, 0, 0, " ")
			So(size, ShouldEqual, len(b))
			So(err, ShouldBeNil)

			size, err = file.ReadAt(ctx, read, 0, 0)
			So(read[:len(b)], ShouldResemble, b)
			So(size, ShouldEqual, 2048)
			So(err, ShouldBeNil)
		})
	})
}

func TestComplicatedReadAndWrite(t *testing.T) {
	Convey("Start test servers", t, func() {
		c, err := fsclient.NewClient(cfg)
		So(err, ShouldBeNil)

		// var file File
		Convey("Read empty file after create", func() {
			file, err := c.Create(ctx, "test file")
			So(err, ShouldBeNil)
			file.Read(ctx) // must call this before write
			// read := make([]byte, 1024)
			for row := 0; row < 10; row++ {
				for col := 0; col < 10; col++ {
					b := constructData(uint32(row), uint32(col))
					file.WriteAt(ctx, b, uint32(row), uint32(col), " ")
				}
			}
			_, size, err := file.Read(ctx) // must call this before write
			So(err, ShouldBeNil)
			So(size, ShouldBeGreaterThanOrEqualTo, 100/4*8192)
		})
	})
}

func TestConcurrentWrite(t *testing.T) {
	Convey("Start test servers", t, func() {
		c, err := fsclient.NewClient(cfg)
		So(err, ShouldBeNil)

		// var file File
		Convey("Read empty file after create", func(conveyC C) {
			file, err := c.Create(ctx, "test file")
			So(err, ShouldBeNil)
			file.Read(ctx) // must call this before write

			// read := make([]byte, 1024)
			var wg sync.WaitGroup
			for row := 0; row < 10; row++ {
				for col := 0; col < 10; col++ {
					row := row
					col := col
					wg.Add(1)
					go func() {
						b := constructData(uint32(row), uint32(col))
						file.WriteAt(ctx, b, uint32(row), uint32(col), " ")
						wg.Done()
					}()
				}
			}

			wg.Wait()
			read, size, err := file.Read(ctx) // must call this before write
			So(len(read), ShouldEqual, size)
			So(err, ShouldBeNil)
		})
	})
}

func TestHandleCrash(t *testing.T) {
	Convey("Start test servers", t, func() {
		c, err := fsclient.NewClient(cfg)
		So(err, ShouldBeNil)

		// var file File
		Convey("Read empty file after create", func(conveyC C) {
			file, err := c.Create(ctx, "test file")
			So(err, ShouldBeNil)
			file.Read(ctx) // must call this before write

			// read := make([]byte, 1024)
			var wg sync.WaitGroup
			for row := 0; row < 5; row++ {
				for col := 0; col < 10; col++ {
					row := row
					col := col
					wg.Add(1)
					go func() {
						b := constructData(uint32(row), uint32(col))
						file.WriteAt(ctx, b, uint32(row), uint32(col), " ")
						wg.Done()
					}()
				}
			}
			wg.Wait()
			os.Stdout.Write([]byte("Please crash datanodes!\n"))
			os.Stdout.Sync()
			buf := make([]byte, 1)
			os.Stdin.Read(buf)
			for row := 5; row < 10; row++ {
				for col := 0; col < 10; col++ {
					row := row
					col := col
					wg.Add(1)
					go func() {
						b := constructData(uint32(row), uint32(col))
						file.WriteAt(ctx, b, uint32(row), uint32(col), " ")
						wg.Done()
					}()
				}
			}
			wg.Wait()
			os.Stdout.Write([]byte("Please crash datanodes and master!\n"))
			os.Stdout.Sync()
			buf = make([]byte, 1)
			os.Stdin.Read(buf)
			read, size, err := file.Read(ctx) // must call this before write
			So(len(read), ShouldEqual, size)
			So(err, ShouldBeNil)
			So(size, ShouldBeGreaterThanOrEqualTo, 100/4*8192)
		})
	})
}
