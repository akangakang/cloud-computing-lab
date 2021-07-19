package fs

import (
	"time"

	"github.com/astaxie/beego"
	"github.com/fourstring/sheetfs/fsclient"
)

var client *fsclient.Client

func getClient() *fsclient.Client {
	var (
		clientConfig *fsclient.ClientConfig
		err          error
	)

	clientConfig = &fsclient.ClientConfig{
		ZookeeperServers:    []string{"zoo1:2181", "zoo2:2181", "zoo3:2181"},
		ZookeeperTimeout:    10 * time.Second,
		MasterZnode:         "/master-ack",
		DataNodeZnodePrefix: "/datanode_election_ack_",
		MaxRetry:            10,
	}

	if client == nil {
		if client, err = fsclient.NewClient(clientConfig); err != nil {
			beego.Error("[FS] Connect to client error", err.Error())
		}
	}

	return client
}
