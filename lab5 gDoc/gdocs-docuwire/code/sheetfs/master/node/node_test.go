package node

import (
	stdctx "context"
	"errors"
	"fmt"
	"github.com/fourstring/sheetfs/master/config"
	"github.com/fourstring/sheetfs/master/filemgr"
	"github.com/fourstring/sheetfs/master/filemgr/mgr_entry"
	"github.com/fourstring/sheetfs/master/journal/checkpoint"
	"github.com/fourstring/sheetfs/master/server"
	"github.com/fourstring/sheetfs/master/sheetfile"
	fs_rpc "github.com/fourstring/sheetfs/protocol"
	"github.com/fourstring/sheetfs/tests"
	"github.com/go-zookeeper/zk"
	. "github.com/smartystreets/goconvey/convey"
	"gorm.io/gorm"
	"log"
	"testing"
	"time"
)

var ckptInterval = 5 * time.Second

func newTestNode(id string, port uint, caddr string) (*testNode, error) {
	db, err := tests.GetPersistTestDB(id, &mgr_entry.MapEntry{}, &sheetfile.Chunk{}, &checkpoint.Checkpoint{})
	if err != nil {
		log.Fatal(err)
	}
	cfg := &MasterNodeConfig{
		NodeID:             id,
		Port:               port,
		ForClientAddr:      caddr,
		ZookeeperServers:   config.ElectionServers,
		ZookeeperTimeout:   config.ElectionTimeout,
		ElectionZnode:      config.ElectionZnode,
		ElectionPrefix:     config.ElectionPrefix,
		ElectionAck:        config.ElectionAck,
		KafkaServer:        config.KafkaServer,
		KafkaTopic:         config.KafkaTopic,
		DB:                 db,
		CheckpointInterval: ckptInterval,
		DataNodeGroups:     []string{"node1"},
	}
	mnode, err := NewMasterNode(cfg)
	if err != nil {
		return nil, err
	}

	go func() {
		err := mnode.Run()
		if err != nil {
			log.Printf("error happens when %s is running: %s\n", id, err)
		}
	}()
	return &testNode{node: mnode, cfg: cfg}, nil
}

type testNode struct {
	node *MasterNode
	cfg  *MasterNodeConfig
}

func (t *testNode) RPC() *server.Server {
	return t.node.rpcsrv
}

func (t *testNode) FM() *filemgr.FileManager {
	return t.node.fm
}

func newTestNodesSet(startPort uint, num int) (map[string]*testNode, error) {
	set := map[string]*testNode{}
	for i := 0; i < num; i++ {
		port := startPort + uint(i)
		id := fmt.Sprintf("mnode%d", i)
		caddr := fmt.Sprintf("127.0.0.1:%d", port)
		n, err := newTestNode(id, port, caddr)
		if err != nil {
			return nil, err
		}
		set[caddr] = n
	}
	return set, nil
}

func newSuccessorTestNode(id string, port uint, caddr string, db *gorm.DB) (*testNode, error) {
	cfg := &MasterNodeConfig{
		NodeID:             id,
		Port:               port,
		ForClientAddr:      caddr,
		ZookeeperServers:   config.ElectionServers,
		ZookeeperTimeout:   config.ElectionTimeout,
		ElectionZnode:      fmt.Sprintf("/%s", tests.RandStr(10)),
		ElectionPrefix:     tests.RandStr(10),
		ElectionAck:        fmt.Sprintf("/%s", tests.RandStr(10)),
		KafkaServer:        config.KafkaServer,
		KafkaTopic:         config.KafkaTopic,
		DB:                 db,
		CheckpointInterval: ckptInterval,
		DataNodeGroups:     []string{"node1"},
	}
	mnode, err := NewMasterNode(cfg)
	if err != nil {
		return nil, err
	}

	go func() {
		err := mnode.Run()
		if err != nil {
			log.Printf("error happens when %s is running: %s\n", id, err)
		}
	}()
	return &testNode{node: mnode, cfg: cfg}, nil
}

func checkPrimaryNode(conn *zk.Conn, nodes map[string]*testNode, ackName string, maxRetry int) (*testNode, []*testNode, error) {
	var primary *testNode
	secondaries := make([]*testNode, 0)
	for i := 0; i < maxRetry; i++ {
		caddr, _, err := conn.Get(ackName)
		if err != nil {
			return nil, nil, err
		}
		if n, ok := nodes[string(caddr)]; ok {
			primary = n
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if primary == nil {
		return nil, nil, errors.New("election failed")
	}
	for _, node := range nodes {
		if node != primary {
			secondaries = append(secondaries, node)
		}
	}
	return primary, secondaries, nil
}

func waitPrimaryAck(conn *zk.Conn, node *testNode) error {
	for {
		caddr, _, err := conn.Get(node.cfg.ElectionAck)
		if err != nil {
			return err
		}
		if string(caddr) == node.cfg.ForClientAddr {
			return nil
		}
	}
}

func shouldBeCompleteSameChunk(actual interface{}, expected ...interface{}) string {
	ac, ok := actual.(*fs_rpc.Chunk)
	if !ok {
		return "actual not a Chunk!"
	}
	ec, ok := expected[0].(*fs_rpc.Chunk)
	if !ok {
		return "expected not a Chunk!"
	}

	if ac.Datanode == ec.Datanode && ac.Version == ec.Version && ac.HoldsMeta == ec.HoldsMeta && ac.Id == ec.Id {
		return ""
	} else {
		return fmt.Sprintf("actual %v, expected %v", ac, ec)
	}
}

func shouldBeCompleteSameCell(actual interface{}, expected ...interface{}) string {
	ac, ok := actual.(*fs_rpc.Cell)
	if !ok {
		return "actual not a Cell!"
	}
	ec, ok := expected[0].(*fs_rpc.Cell)
	if !ok {
		return "expected not a Cell!"
	}

	if ac.Size == ec.Size && ac.Offset == ec.Offset && shouldBeCompleteSameChunk(ac.Chunk, ec.Chunk) == "" {
		return ""
	} else {
		return fmt.Sprintf("actual %v, expected %v", ac, ec)
	}
}

func getTestFilename(no int) string {
	return fmt.Sprintf("sheet%d", no)
}

func populatePrimary(primary *testNode, totalFiles, rowsPerFile, colsPerFile int) {
	cellsPerFile := rowsPerFile * colsPerFile
	fds := make([]uint64, 0)
	for i := 0; i < totalFiles; i++ {
		filename := getTestFilename(i)
		rep, err := primary.RPC().CreateSheet(stdctx.Background(), &fs_rpc.CreateSheetRequest{Filename: filename})
		So(err, ShouldBeNil)
		So(rep.Status, ShouldEqual, fs_rpc.Status_OK)
		fds = append(fds, rep.Fd)
	}
	for i := 0; i < totalFiles; i++ {
		for j := 0; j < rowsPerFile; j++ {
			for k := 0; k < colsPerFile; k++ {
				curCellNum := uint64(i*cellsPerFile + j*colsPerFile + k)
				req := &fs_rpc.WriteCellRequest{
					Fd:     fds[i],
					Row:    uint32(j),
					Column: uint32(k),
				}
				rep, err := primary.RPC().WriteCell(stdctx.Background(), req)
				So(err, ShouldBeNil)
				So(rep.Status, ShouldEqual, fs_rpc.Status_OK)
				So(rep.Cell, shouldBeCompleteSameCell, &fs_rpc.Cell{
					Chunk: &fs_rpc.Chunk{
						Id:        11 + curCellNum/config.MaxCellsPerChunk,
						Datanode:  "node1",
						Version:   1 + curCellNum%4,
						HoldsMeta: false,
					},
					Offset: (curCellNum % 4) * config.MaxBytesPerCell,
					Size:   config.MaxBytesPerCell,
				})
			}
		}
	}
	for i := 0; i < totalFiles; i++ {
		if i%2 == 0 {
			filename := getTestFilename(i)
			req := &fs_rpc.RecycleSheetRequest{Filename: filename}
			rep, err := primary.RPC().RecycleSheet(stdctx.Background(), req)
			So(err, ShouldBeNil)
			So(rep.Status, ShouldEqual, fs_rpc.Status_OK)
		}
	}
}

func populateCheckpointSuccessor(succ *testNode, totalFiles int) {
	for i := 0; i < totalFiles; i++ {
		filename := getTestFilename(i)
		succ.FM().Opened[filename] = sheetfile.LoadSheetFile(succ.cfg.DB, succ.node.alloc, filename)
	}
}

func verifySecondary(secondary *testNode, totalFiles, rowsPerFile, colsPerFile int) {
	cellsPerFile := rowsPerFile * colsPerFile
	for i := 0; i < totalFiles; i++ {
		filename := getTestFilename(i)
		e, ok := secondary.FM().Entries[filename]
		So(ok, ShouldBeTrue)
		So(e.FileName, ShouldEqual, filename)
		So(e.Recycled, ShouldEqual, i%2 == 0)
		sheet, ok := secondary.FM().Opened[filename]
		So(ok, ShouldBeTrue)
		for j := 0; j < rowsPerFile; j++ {
			for k := 0; k < colsPerFile; k++ {
				curCellNum := uint64(i*cellsPerFile + j*colsPerFile + k)
				cell, ok := sheet.Cells[sheetfile.GetCellID(uint32(j), uint32(k))]
				So(ok, ShouldBeTrue)
				chunk, ok := sheet.Chunks[cell.ChunkID]
				So(ok, ShouldBeTrue)
				So(cell.SheetName, ShouldEqual, filename)
				So(cell.Size, ShouldEqual, config.MaxBytesPerCell)
				So(cell.Offset, ShouldEqual, (curCellNum%4)*config.MaxBytesPerCell)
				So(chunk.ID, ShouldEqual, 11+curCellNum/config.MaxCellsPerChunk)
				So(chunk.DataNode, ShouldEqual, "node1")
				So(chunk.Version, ShouldEqual, 4)
				So(len(chunk.Cells), ShouldEqual, 4)
			}
		}
	}
}

func TestMasterNodeReplication(t *testing.T) {
	totalFiles := 10
	rowsPerFile := 10
	colsPerFile := 10

	Convey("Construct test nodes", t, func() {
		nodesSet, err := newTestNodesSet(8432, 3)
		So(err, ShouldBeNil)
		zkConn, _, err := zk.Connect(config.ElectionServers, config.ElectionTimeout)
		So(err, ShouldBeNil)
		Convey("check primary", func() {
			primary, secondaries, err := checkPrimaryNode(zkConn, nodesSet, config.ElectionAck, 20)
			So(err, ShouldBeNil)
			log.Printf("Select primary: %s\n", primary.cfg.NodeID)
			populatePrimary(primary, totalFiles, rowsPerFile, colsPerFile)
			time.Sleep(1 * time.Second) // wait for journal replication
			for _, secondary := range secondaries {
				verifySecondary(secondary, totalFiles, rowsPerFile, colsPerFile)
			}
			time.Sleep(2 * ckptInterval) // wait for checkpoint replication
			ckptSuccessor, err := newSuccessorTestNode("ckpt-successor", 18433, "127.0.0.1:18433", secondaries[0].node.db)
			populateCheckpointSuccessor(ckptSuccessor, totalFiles)
			So(err, ShouldBeNil)
			err = waitPrimaryAck(zkConn, ckptSuccessor)
			So(err, ShouldBeNil)
			verifySecondary(ckptSuccessor, totalFiles, rowsPerFile, colsPerFile)
			db, err := tests.GetPersistTestDB("fresh-successor", &mgr_entry.MapEntry{}, &sheetfile.Chunk{}, &checkpoint.Checkpoint{})
			freshSuccessor, err := newSuccessorTestNode("fresh-successor", 18432, "127.0.0.1:18432", db)
			So(err, ShouldBeNil)
			err = waitPrimaryAck(zkConn, freshSuccessor)
			So(err, ShouldBeNil)
			verifySecondary(freshSuccessor, totalFiles, rowsPerFile, colsPerFile)
		})
	})
}
