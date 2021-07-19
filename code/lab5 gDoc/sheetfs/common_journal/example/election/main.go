package main

import (
	"context"
	"errors"
	"flag"
	"github.com/fourstring/sheetfs/common_journal"
	journal_example "github.com/fourstring/sheetfs/common_journal/example"
	"github.com/fourstring/sheetfs/election"
	"github.com/fourstring/sheetfs/tests"
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
var num = flag.Int("n", 5, "Number of messages to produce if this node become primary")
var ckpt = flag.Bool("c", false, "whether to make a checkpoint after produce {num} messages")
var nextEntryOffset = flag.Int64("o", 0, "offset of next entry to be consumed")

func handleMsg(msg []byte, ckpt *common_journal.Checkpoint) {
	if ckpt != nil {
		log.Printf("%s: receive a Checkpoint entry: %s\n", *id, ckpt)
		/*
			Real secondary should perform checkpoint operations here and persist
			ckpt.NextEntryOffset where the node start to recover data through journal entries
			after crash.
		*/
	} else {
		log.Printf("%s: receive a general entry: %s\n", *id, string(msg))
	}
}

func main() {
	flag.Parse()
	e, err := election.NewElector(electionServers, 1*time.Second, electionZnode, electionPrefix, electionAck)
	proposal, err := e.CreateProposal()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s: My proposal is %s\n", *id, proposal)
	receiver, err := common_journal.NewReceiver(journal_example.KafkaServer, journal_example.KafkaTopic)
	if err != nil {
		log.Fatal(err)
	}
	err = receiver.SetOffset(*nextEntryOffset)
	if err != nil {
		log.Fatal(err)
	}
	for {
		success, watch, notify, err := e.TryBeLeader()
		if err != nil {
			log.Fatal(err)
		}
		if success {
			break
		}
		log.Printf("%s: I'm secondary and watching %s\n", *id, watch)
		ctx := common_journal.NewZKEventCancelContext(context.Background(), notify)
		/*
			Generally, a secondary node should invoke receiver.FetchEntry to blocking fetch and
			applies entries until it realized that it has become a primary node.
		*/
		for {
			msg, ckpt, err := receiver.FetchEntry(ctx)
			// Sleep intentionally here to show forwarding all remaining entries when become a primary node.
			time.Sleep(1 * time.Second)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					break
				} else {
					log.Fatal(err)
				}
			}
			handleMsg(msg, ckpt)
		}
	}
	/*
		MUST complete all preparation required to serve requests before AckLeader!
		Forwarding all journal entries here using receiver.TryFetchMessage until a NoMoreMessageError is returned
	*/
	log.Printf("%s: I'm primary! Start fast forwarding now!\n", *id)
	for {
		msg, ckpt, err := receiver.TryFetchEntry(context.Background())
		if err != nil {
			// New primary has consumed all remaining messages.
			if errors.Is(err, &common_journal.NoMoreMessageError{}) {
				break
			} else {
				log.Fatal(err)
			}
		}
		handleMsg(msg, ckpt)
	}
	log.Printf("%s: I have done all preparations! Start acking as a primary.", *id)
	err = e.AckLeader(*id)
	if err != nil {
		log.Fatal(err)
	}
	/*
		Do primary works here
	*/
	writer, err := common_journal.NewWriter(journal_example.KafkaServer, journal_example.KafkaTopic)
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < *num; i++ {
		content := tests.RandStr(10)
		err = writer.CommitEntry(context.Background(), []byte(content))
		log.Printf("%s: committed entry %s\n", *id, content)
		if err != nil {
			log.Fatal(err)
		}
	}
	if *ckpt {
		log.Printf("%s: I'm checkpointing!", *id)
		writer.PrepareCheckpoint()
		newStartOffset, err := writer.Checkpoint(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%s: checkpoint successfully, and newStartOffset=%d\n", *id, newStartOffset)
		writer.ExitCheckpoint()
	}
	log.Printf("%s: All works done, please kill me.", *id)
	for {
		time.Sleep(1 * time.Minute)
	}
}
