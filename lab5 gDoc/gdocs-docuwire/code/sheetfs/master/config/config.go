package config

import (
	"math"
	"time"
)

const (
	MaxCellsPerChunk   = 4
	BytesPerChunk      = uint64(8192)
	MaxBytesPerCell    = BytesPerChunk / MaxCellsPerChunk
	DBName             = "master.db"
	SheetMetaCellRow   = uint32(math.MaxUint32)
	SheetMetaCellCol   = uint32(math.MaxUint32)
	ElectionZnode      = "/master_election"
	ElectionAck        = "/master_election_ack"
	ElectionPrefix     = "a20ffeb5-319a-4e0b-b54d-646fb93d3158-n_"
	ElectionTimeout    = 1 * time.Second
	CheckpointInterval = 1 * time.Minute
)

var SheetMetaCellID = int64(0)
var ElectionServers = []string{
	"127.0.0.1:2181",
	"127.0.0.1:2182",
	"127.0.0.1:2183",
}
var KafkaServer = "127.0.0.1:9093"
var KafkaTopic = "master_journal"

func init() {
	SheetMetaCellID = -1
}
