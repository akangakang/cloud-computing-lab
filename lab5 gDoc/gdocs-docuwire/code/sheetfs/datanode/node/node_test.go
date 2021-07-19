package node

import (
	stdctx "context"
	"errors"
	"fmt"
	"github.com/fourstring/sheetfs/datanode/config"
	"github.com/fourstring/sheetfs/datanode/server"
	fs_rpc "github.com/fourstring/sheetfs/protocol"
	"github.com/go-zookeeper/zk"
	. "github.com/smartystreets/goconvey/convey"
	"log"
	"testing"
	"time"
)

type testNode struct {
	node *DataNode
	cfg  *DataNodeConfig
}

func (t *testNode) RPC() *server.Server {
	return t.node.rpcsrv
}

func newTestNode(id string, port uint, caddr string, name string, serverList []string, electionPrefix string) (*testNode, error) {
	cfg := &DataNodeConfig{
		NodeID:           id,
		Port:             port,
		ForClientAddr:    caddr,
		ZookeeperServers: serverList,
		DataDirPath:      config.DIR_DATA_PATH,
		ZookeeperTimeout: config.ElectionTimeout,
		ElectionZnode:    config.ElectionZnodePrefix + name,
		ElectionPrefix:   electionPrefix,
		ElectionAck:      config.ElectionAckPrefix + name,
		KafkaServer:      config.KafkaServer,
		KafkaTopic:       config.KafkaTopicPrefix + name,
	}
	mnode, err := NewDataNode(cfg)
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

func newTestNodesSet(startPort uint, num int, electionServers []string) (map[string]*testNode, error) {
	set := map[string]*testNode{}
	electionPrefix := "4da1fce7-d3f8-42dd-965d-4c3311661202-n_"
	for i := 0; i < num; i++ {
		port := startPort + uint(i)
		id := fmt.Sprintf("dnode%d", i)
		caddr := fmt.Sprintf("127.0.0.1:%d", port)
		n, err := newTestNode(id, port, caddr, "node1", electionServers, electionPrefix)
		if err != nil {
			return nil, err
		}
		set[caddr] = n
	}
	return set, nil
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

func populatePrimary(primary *testNode, id uint64, offset uint64, version uint64, data []byte) {

	_, err := primary.RPC().DeleteChunk(stdctx.Background(), &fs_rpc.DeleteChunkRequest{
		Id: id,
	})
	So(err, ShouldBeNil)

	time.Sleep(1 * time.Second) // wait for journal replication
	rep, err := primary.RPC().WriteChunk(stdctx.Background(), &fs_rpc.WriteChunkRequest{
		Id:      id,
		Offset:  offset,
		Padding: " ",
		Size:    config.BLOCK_SIZE,
		Version: version,
		Data:    data,
	})
	So(err, ShouldBeNil)
	So(rep.Status, ShouldEqual, fs_rpc.Status_OK)
}

func verifySecondary(secondary *testNode, id uint64, offset uint64, version uint64, data []byte) {
	rep, err := secondary.RPC().ReadChunk(stdctx.Background(), &fs_rpc.ReadChunkRequest{
		Id:      id,
		Offset:  offset,
		Size:    config.BLOCK_SIZE,
		Version: version, // read will not increase the version
	})
	So(err, ShouldBeNil)
	So(rep.Status, ShouldEqual, fs_rpc.Status_OK)
	So(rep.Data[:len(data)], ShouldResemble, data)
}

func TestDataNodeReplication(t *testing.T) {
	chunkId := uint64(1)
	offset := uint64(0)
	version := uint64(1) // first version return by master should be 1
	data := []byte("this is the test data.")
	var electionServers = []string{
		"127.0.0.1:2181",
		"127.0.0.1:2182",
		"127.0.0.1:2183",
	}

	Convey("Construct test nodes", t, func() {
		// construct node set
		nodesSet, err := newTestNodesSet(9375, 3, electionServers)
		So(err, ShouldBeNil)

		// connect to servers
		zkConn, _, err := zk.Connect(electionServers, config.ElectionTimeout)
		So(err, ShouldBeNil)

		// get the primary node and secondary node list
		primary, secondaries, err := checkPrimaryNode(zkConn, nodesSet, config.ElectionAckPrefix+"node1", 20)
		So(err, ShouldBeNil)
		log.Printf("Select primary: %s\n", primary.cfg.NodeID)

		// write a chunk in primary and check secondary
		populatePrimary(primary, chunkId, offset, version, data)
		time.Sleep(1 * time.Second) // wait for journal replication
		for _, secondary := range secondaries {
			verifySecondary(secondary, chunkId, offset, version, data)
		}
	})
}
