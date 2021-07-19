package main

import (
	"flag"
	"github.com/fourstring/sheetfs/master/config"
	"github.com/fourstring/sheetfs/master/node"
	"log"
	"strings"
)

var port = flag.Uint("p", 0, "port to listen on")
var forClientAddress = flag.String("a", "", "address for client to connect to this node")
var nodeId = flag.String("i", "", "ID of this node")
var zkServers = flag.String("zkservers", "", "comma separated list of zookeeper servers")
var electionZnode = flag.String("elznode", "", "path of znode for election")
var electionAck = flag.String("elack", "", "path of znode for acknowledge primary")
var kafkaServer = flag.String("kfserver", "", "address of kafka server")
var kafkaTopic = flag.String("kftopic", "", "name of kafka topic to rw journals")
var dataNodeGroups = flag.String("dngroups", "", "comma separated list of datanode groupss")

func parseCommaList(l string) []string {
	return strings.Split(l, ",")
}

func main() {
	flag.Parse()
	db, err := connectDB(*nodeId)
	if err != nil {
		log.Fatal(err)
	}
	cfg := &node.MasterNodeConfig{
		NodeID:             *nodeId,
		Port:               *port,
		ForClientAddr:      *forClientAddress,
		ZookeeperServers:   parseCommaList(*zkServers),
		ZookeeperTimeout:   config.ElectionTimeout,
		ElectionZnode:      *electionZnode,
		ElectionPrefix:     config.ElectionPrefix,
		ElectionAck:        *electionAck,
		KafkaServer:        *kafkaServer,
		KafkaTopic:         *kafkaTopic,
		DB:                 db,
		CheckpointInterval: config.CheckpointInterval,
		DataNodeGroups:     parseCommaList(*dataNodeGroups),
	}
	log.Printf("%v\n", cfg)
	mnode, err := node.NewMasterNode(cfg)
	if err != nil {
		log.Fatal(err)
	}
	err = mnode.Run()
	if err != nil {
		log.Fatal(err)
	}
}
