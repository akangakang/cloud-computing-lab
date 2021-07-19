package node

import (
	"fmt"
	"github.com/fourstring/sheetfs/common_journal"
	"github.com/fourstring/sheetfs/election"
	"github.com/fourstring/sheetfs/master/datanode_alloc"
	"github.com/fourstring/sheetfs/master/filemgr"
	"github.com/fourstring/sheetfs/master/journal"
	"github.com/fourstring/sheetfs/master/server"
	fs_rpc "github.com/fourstring/sheetfs/protocol"
	"google.golang.org/grpc"
	"gorm.io/gorm"
	"net"
	"time"
)

type MasterNodeConfig struct {
	NodeID             string
	Port               uint
	ForClientAddr      string
	ZookeeperServers   []string
	ZookeeperTimeout   time.Duration
	ElectionZnode      string
	ElectionPrefix     string
	ElectionAck        string
	KafkaServer        string
	KafkaTopic         string
	DB                 *gorm.DB
	CheckpointInterval time.Duration
	DataNodeGroups     []string
}

type MasterNode struct {
	db           *gorm.DB
	fm           *filemgr.FileManager
	jWriter      *common_journal.Writer
	listener     *journal.Listener
	elector      *election.Elector
	port         uint
	cAddr        string
	alloc        *datanode_alloc.DataNodeAllocator
	ckptInterval time.Duration
	rpcsrv       *server.Server
}

func NewMasterNode(config *MasterNodeConfig) (*MasterNode, error) {
	m := &MasterNode{
		db:           config.DB,
		port:         config.Port,
		cAddr:        config.ForClientAddr,
		ckptInterval: config.CheckpointInterval,
	}
	elector, err := election.NewElector(config.ZookeeperServers, config.ZookeeperTimeout, config.ElectionZnode, config.ElectionPrefix, config.ElectionAck)
	if err != nil {
		return nil, err
	}
	m.elector = elector

	m.alloc = datanode_alloc.NewDataNodeAllocatorWithGroups(config.DataNodeGroups)

	jw, err := common_journal.NewWriter(config.KafkaServer, config.KafkaTopic)
	if err != nil {
		return nil, err
	}
	m.jWriter = jw
	m.fm = filemgr.LoadFileManager(config.DB, m.alloc, jw)

	lis, err := journal.NewListener(&journal.ListenerConfig{
		NodeID:      config.NodeID,
		Elector:     elector,
		KafkaServer: config.KafkaServer,
		KafkaTopic:  config.KafkaTopic,
		FileManager: m.fm,
		DB:          m.db,
	})

	if err != nil {
		return nil, err
	}
	m.listener = lis

	return m, nil
}

func (m *MasterNode) RunAsSecondary() error {
	_, err := m.elector.CreateProposal()
	if err != nil {
		return err
	}
	return m.listener.RunAsSecondary()
}

func (m *MasterNode) RunAsPrimary() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", m.port))
	if err != nil {
		return err
	}
	srv, err := server.NewServer(m.fm, m.alloc)
	if err != nil {
		return err
	}
	m.rpcsrv = srv
	s := grpc.NewServer()
	fs_rpc.RegisterMasterNodeServer(s, srv)
	err = m.elector.AckLeader(m.cAddr)
	if err != nil {
		return err
	}
	go func() {
		ticker := time.NewTicker(m.ckptInterval)
		defer ticker.Stop()
		for {
			<-ticker.C
			m.fm.DoCheckpoint()
		}
	}()
	err = s.Serve(lis)
	if err != nil {
		return err
	}
	return nil
}

func (m *MasterNode) Run() error {
	err := m.RunAsSecondary()
	if err != nil {
		return err
	}
	return m.RunAsPrimary()
}
