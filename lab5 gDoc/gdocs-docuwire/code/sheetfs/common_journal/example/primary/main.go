package main

import (
	"context"
	"flag"
	"github.com/fourstring/sheetfs/common_journal"
	journal_example "github.com/fourstring/sheetfs/common_journal/example"
	"github.com/fourstring/sheetfs/tests"
	"log"
)

var num = flag.Int("n", 5, "Number of messages to produce")
var ckpt = flag.Bool("c", false, "whether to make a checkpoint after produce {num} messages")
var ctx = context.Background()

func main() {
	flag.Parse()
	writer, err := common_journal.NewWriter(journal_example.KafkaServer, journal_example.KafkaTopic)
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < *num; i++ {
		content := tests.RandStr(10)
		err := writer.CommitEntry(ctx, []byte(content))
		log.Printf("Primary: committed entry %s\n", content)
		if err != nil {
			log.Fatal(err)
		}
	}
	if *ckpt {
		writer.PrepareCheckpoint()
		/*
			Do real checkpoint operation here before call writer.Checkpoint()
		*/
		newStartOffset, err := writer.Checkpoint(ctx) // primary should persist newStartOffset, and it can
		// start from here to recover data through journal entries.
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Primary: checkpoint successfully, and newStartOffset=%d\n", newStartOffset)
		writer.ExitCheckpoint()
	}
}
