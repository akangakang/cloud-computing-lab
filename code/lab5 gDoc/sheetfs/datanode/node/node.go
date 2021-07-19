package node

import (
	"context"
	"errors"
	"fmt"
	"github.com/fourstring/sheetfs/common_journal"
	"github.com/fourstring/sheetfs/datanode/server"
	"github.com/fourstring/sheetfs/election"
	fs_rpc "github.com/fourstring/sheetfs/protocol"
	"google.golang.org/grpc"
	"log"
	"net"
	"time"
)

type DataNodeConfig struct {
	NodeID           string
	Port             uint
	ForClientAddr    string
	DataDirPath      string
	ZookeeperServers []string
	ZookeeperTimeout time.Duration
	ElectionZnode    string
	ElectionPrefix   string
	ElectionAck      string
	KafkaServer      string
	KafkaTopic       string
}

type DataNode struct {
	jWriter  *common_journal.Writer
	receiver *common_journal.Receiver
	elector  *election.Elector
	port     uint
	cAddr    string
	rpcsrv   *server.Server
}

func NewDataNode(config *DataNodeConfig) (*DataNode, error) {
	d := &DataNode{
		port:  config.Port,
		cAddr: config.ForClientAddr,
	}
	elector, err := election.NewElector(config.ZookeeperServers, config.ZookeeperTimeout, config.ElectionZnode, config.ElectionPrefix, config.ElectionAck)
	if err != nil {
		return nil, err
	}
	d.elector = elector

	jw, err := common_journal.NewWriter(config.KafkaServer, config.KafkaTopic)
	if err != nil {
		return nil, err
	}
	d.jWriter = jw

	receiver, err := common_journal.NewReceiver(config.KafkaServer, config.KafkaTopic)
	if err != nil {
		return nil, err
	}
	d.receiver = receiver

	rpcsrv := server.NewServer(config.DataDirPath+config.NodeID, jw)
	d.rpcsrv = rpcsrv

	return d, nil
}

func (d *DataNode) RunAsSecondary() error {
	_, err := d.elector.CreateProposal()
	if err != nil {
		return err
	}
	for {
		success, _, notify, err := d.elector.TryBeLeader()
		if err != nil {
			log.Fatal(err)
		}
		if success {
			break
		}
		ctx := common_journal.NewZKEventCancelContext(context.Background(), notify)
		/*
			Generally, a secondary node should invoke receiver.FetchEntry to blocking fetch and
			applies entries until it realized that it has become a primary node.
		*/
		for {
			msg, _, err := d.receiver.FetchEntry(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					break
				} else {
					log.Fatal(err)
				}
			}
			err = d.rpcsrv.HandleMsg(msg)
			if err != nil {
				return err
			}
		}
	}
	for {
		msg, _, err := d.receiver.TryFetchEntry(context.Background())
		if err != nil {
			// New primary has consumed all remaining messages.
			if errors.Is(err, &common_journal.NoMoreMessageError{}) {
				break
			} else {
				log.Fatal(err)
			}
		}
		err = d.rpcsrv.HandleMsg(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *DataNode) RunAsPrimary() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", d.port))
	if err != nil {
		return err
	}

	s := grpc.NewServer()
	fs_rpc.RegisterDataNodeServer(s, d.rpcsrv)

	// ABANDONED NOW: register data node
	/*
		connZK, _, err := zk.Connect(config.ElectionServers, config.ElectionTimeout)
		if err != nil {
			log.Fatalf("Connect masternode server failed.")
		}

		masterAddr, _, err := connZK.Get(config.MasterAck)
		if err != nil {
			log.Fatalf("Get master address from commection failed.")
		}

		connDial, _ := grpc.Dial(string(masterAddr))
		var masterClient = fs_rpc.NewMasterNodeClient(connDial)

		rep, err := masterClient.RegisterDataNode(context.Background(),
			&fs_rpc.RegisterDataNodeRequest{Addr: d.cAddr})
		if err != nil {
			log.Fatalf("%s", err)
		}
		if rep.Status != fs_rpc.Status_OK {
			log.Fatalf("Register failed.")
		}
	*/

	// ack leader
	err = d.elector.AckLeader(d.cAddr)
	if err != nil {
		return err
	}
	err = s.Serve(lis)
	if err != nil {
		return err
	}
	return nil
}

func (d *DataNode) Run() error {
	err := d.RunAsSecondary()
	if err != nil {
		return err
	}
	return d.RunAsPrimary()
}
