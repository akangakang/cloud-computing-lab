package common_journal

import (
	"context"
	"fmt"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
	"sync"
)

/*
Writer abstracts Kafka operations. The primary node can utilize this type to send
journal entries it produced to Kafka, making corresponding secondary nodes to be
able to replicate them and achieves master-backup fault tolerance.

Writer also provides a version-based Writer.Checkpoint API. It will send a special
checkpoint journal entry to Kafka, which can be used to represent a complete checkpoint
operation conducted by the primary node. After receiving such an entry, a secondary
node should also perform checkpointing too.

Internally, Writer uses two different event keys to represent general journal entries
which is opaque to Writer, and those special Checkpoint entries. Because Kafka only
guarantees that write operations to the same partition of the same topic are ordered,
Writer applies a FixedBalancer to its internal kafka.Writer, which makes the two keys
are always routed to the same partition.
*/
type Writer struct {
	ckptMu                     sync.RWMutex
	cbMu                       sync.Mutex
	lastWriteOffset            int64
	writer                     *kafka.Writer
	entriesKey, checkpointsKey []byte
}

/*
Initialize a Writer.

This function tries to create a single partition topic with given name, if it exists,
do nothing, and if other errors happened during creation, return it.

@param
	server: address to a Kafka server.
	topic: Kafka topic name used to store messages.

@return
	error: not nil if failed to ensure the topic is created.
*/
func NewWriter(server, topic string) (*Writer, error) {
	err := ensureJournalTopicCreated(server, topic)
	if err != nil {
		return nil, err
	}
	w := &Writer{
		entriesKey:     []byte(fmt.Sprintf("%s-entries", topic)),
		checkpointsKey: []byte(fmt.Sprintf("%s-ckpts", topic)),
	}
	ew := &kafka.Writer{
		Addr:      kafka.TCP(server),
		Topic:     topic,
		Balancer:  &kafka.LeastBytes{},
		Async:     false,
		BatchSize: 1,
		Completion: func(messages []kafka.Message, err error) {
			if err != nil || len(messages) == 0 {
				return
			}
			lastMsg := messages[len(messages)-1]
			w.cbMu.Lock()
			defer w.cbMu.Unlock()
			if w.lastWriteOffset < lastMsg.Offset {
				w.lastWriteOffset = lastMsg.Offset
			}
		},
		RequiredAcks: kafka.RequireOne,
	}
	w.writer = ew
	return w, nil
}

/*
Commit a general journal entry to Kafka. Writer employs synchronous mode to write
messages, so this method will block until the new journal entry is written successfully
or exceeds max attempt times.

@param
	ctx: Context used to cancel operation asynchronously.
	entry: data of journal entry.

@return
	error: not nil if failed to write message.
*/
func (w *Writer) CommitEntry(ctx context.Context, entry []byte) error {
	w.ckptMu.RLock()
	defer w.ckptMu.RUnlock()
	err := w.writer.WriteMessages(ctx, kafka.Message{
		Key:   w.entriesKey,
		Value: entry,
	})
	return err
}

/*
There may be many goroutines are using a Writer in a primary node, but when the latter wants
to perform Checkpoint operation, it needs to ensure all pending CommitEntry have finished and
there is no more incoming CommitEntry. This can be achieved by calling PrepareCheckpoint,
this method guarantees that when it returns, all pending CommitEntry are done and no more incoming
ones.

Internally, we use a writer-preferred RWMutex here, to make Checkpoint operation blocks all incoming
CommitEntry, and all CommitEntry don't interfere with each other. PrepareCheckpoint will Lock this
RWMutex, and CommitEntry will RLock it.
*/
func (w *Writer) PrepareCheckpoint() int64 {
	w.ckptMu.Lock()
	return w.lastWriteOffset
}

/*
Unlock the RWMutex, to allow further CommitEntry to be executed.
*/
func (w *Writer) ExitCheckpoint() {
	w.ckptMu.Unlock()
}

/*
Commit a checkpoint entry to Kafka. This method should be invoked by a primary node
who has already performs a checkpoint operation. And an offset is returned by this
method which indicates the offset of the first entry after the new checkpoint entry,
this offset can be used by the primary node to recover from journals after crash.
*/
func (w *Writer) Checkpoint(ctx context.Context) (int64, error) {
	// Fetch latest offset of entries
	ckptOffset := w.lastWriteOffset + 1
	ckpt := Checkpoint{
		LastEntryOffset: ckptOffset - 1, // when there is no message written, it can be -1
		NextEntryOffset: ckptOffset + 1,
	}
	buf, err := proto.Marshal(&ckpt)
	err = w.writer.WriteMessages(ctx, kafka.Message{
		Key:   w.checkpointsKey,
		Value: buf,
	})
	return ckpt.NextEntryOffset, err
}
