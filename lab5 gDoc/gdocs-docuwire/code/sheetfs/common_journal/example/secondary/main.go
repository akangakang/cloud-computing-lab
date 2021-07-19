package main

import (
	"context"
	"flag"
	"github.com/fourstring/sheetfs/common_journal"
	journal_example "github.com/fourstring/sheetfs/common_journal/example"
	"log"
	"time"
)

var nextEntryOffset = flag.Int64("n", 0, "offset of next entry to be consumed")

func fakePersist(offset int64) {

}

func main() {
	flag.Parse()
	receiver, err := common_journal.NewReceiver(journal_example.KafkaServer, journal_example.KafkaTopic)
	if err != nil {
		log.Fatal(err)
	}
	err = receiver.SetOffset(*nextEntryOffset)
	/*
		Secondary spins to pull journal entries from kafka and applies them here.
	*/
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for {
		msg, ckpt, err := receiver.FetchEntry(ctx)
		if err != nil {
			log.Printf("Secondary: May be I'm cancelled! err: %s\n", err)
			break
		}
		if ckpt != nil {
			log.Printf("Secondary: receive a Checkpoint entry: %s\n", ckpt)
			/*
				Real secondary should perform checkpoint operations here and persist
				ckpt.NextEntryOffset where the node start to recover data through journal entries
				after crash.
			*/
			fakePersist(ckpt.NextEntryOffset)
		} else {
			log.Printf("Secondary: receive a general entry:%s\n", string(msg))
		}
	}
}
