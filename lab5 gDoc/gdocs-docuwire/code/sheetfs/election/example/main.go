package main

import (
	"flag"
	"fmt"
	"github.com/fourstring/sheetfs/election"
	"log"
	"time"
)

var electionZnode = "/test_election"
var electionPrefix = "4da1fce7-d3f8-42dd-965d-4c3311661202-n_"
var electionAck = "/test_election_ack"
var electionServers = []string{
	"127.0.0.1:2181",
	"127.0.0.1:2182",
	"127.0.0.1:2183",
}
var id = flag.String("i", "", "ID of proposer")

func main() {
	flag.Parse()
	e, err := election.NewElector(electionServers, 1*time.Second, electionZnode, electionPrefix, electionAck)
	proposal, err := e.CreateProposal()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s: My proposal is %s\n", *id, proposal)
	for {
		success, watch, notify, err := e.TryBeLeader()
		if err != nil {
			log.Fatal(err)
		}
		if success {
			break
		}
		fmt.Printf("%s: I'm secondary and watching %s\n", *id, watch)
		done := false
		for !done {
			select {
			case <-notify:
				done = true
			default:
				/*
					Do works of a secondary node here.
				*/
			}
		}
	}
	/*
		MUST complete all preparation required to serve requests before AckLeader!
	*/
	fmt.Printf("%s: I'm primary!\n", *id)
	err = e.AckLeader(*id)
	if err != nil {
		log.Fatal(err)
	}
	for {
		time.Sleep(1 * time.Minute)
	}
}
