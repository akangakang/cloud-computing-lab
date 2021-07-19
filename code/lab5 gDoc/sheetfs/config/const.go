package config

import (
	"hash/crc32"
	"time"
)

const (
	BLOCK_SIZE             = 2 << 10
	FILE_SIZE              = BLOCK_SIZE << 2
	VERSION_START_LOCATION = BLOCK_SIZE << 2
	SPECIAL_ID             = ^uint64(0)
	DataNodeAckPrefix      = "/datanode_election_ack_"
	MasterAck              = "/master_election_ack"
	AckTimeout             = 1 * time.Second
	ACK_MOST_TIMES         = 5
	WRITE_LOG_FLAG         = uint64(1)
	DELETE_LOG_FLAG        = uint64(2)
)

var ElectionServer = []string{
	"127.0.0.1:2181",
	"127.0.0.1:2182",
	"127.0.0.1:2183",
}
var KafkaServer = "127.0.0.1:9093"
var KafkaTopicPrefix = "datanode_journal_"
var Crc32q = crc32.MakeTable(0xD5828281)
