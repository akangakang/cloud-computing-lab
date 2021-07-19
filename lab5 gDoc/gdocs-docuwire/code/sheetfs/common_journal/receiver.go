package common_journal

import (
	"context"
	"fmt"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

/*
Receiver wraps around a kafka.Reader, offering methods for a secondary node
to receive and replicate journal entries.
*/
type Receiver struct {
	entriesReader              *kafka.Reader
	entriesKey, checkpointsKey []byte
}

/*
Initialize a Receiver.

This function tries to create a single partition topic with given name, if it exists,
do nothing, and if other errors happened during creation, return it.

@param
	server: address to a Kafka server.
	topic: Kafka topic name used to store messages.

@return
	error: not nil if failed to ensure the topic is created.
*/
func NewReceiver(server, topic string) (*Receiver, error) {
	err := ensureJournalTopicCreated(server, topic)
	if err != nil {
		return nil, err
	}
	er := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{server},
		Topic:     topic,
		Partition: 0,
		MinBytes:  1,
		MaxBytes:  1e6,
	})
	return &Receiver{
		entriesReader:  er,
		entriesKey:     []byte(fmt.Sprintf("%s-entries", topic)),
		checkpointsKey: []byte(fmt.Sprintf("%s-ckpts", topic)),
	}, nil
}

/*
Blocking until fetch an entry from Kafka successfully or an error raised. This method should be
used when a secondary node pulls entries and applies them continuously as a secondary one.

When a message got fetched, this method checks whether the message is a Checkpoint or not. If
so, it will Unmarshall this message and return an unmarshalled *Checkpoint. If not, it just
returns raw byte content of the message. Applications should parse those bytes by themselves.

@param
	ctx: Context used to cancel operation asynchronously.

@return
	[]byte: raw bytes content of journal entry if success
	*Checkpoint: points to an unmarshalled Checkpoint object if success and the entry fetched
	is a Checkpoint entry.
	error: not nil if failed to fetch message or unmarshall Checkpoint,
*/
func (r *Receiver) FetchEntry(ctx context.Context) ([]byte, *Checkpoint, error) {
	msg, err := r.entriesReader.ReadMessage(ctx)
	if err != nil {
		return nil, nil, err
	}
	if string(msg.Key) == string(r.checkpointsKey) {
		ckpt := &Checkpoint{}
		err := proto.Unmarshal(msg.Value, ckpt)
		if err != nil {
			return nil, nil, err
		}
		return msg.Value, ckpt, nil
	}
	return msg.Value, nil, nil
}

/*
Fetch a journal entry from kafka.  If there are more messages to be consumed in kafka, it will
block until fetch a message successfully or fail to do that. And it will return immediately with
a *NoMoreMessageError if there is no more message in kafka currently by computing lag of internal
kafka.Reader. This error can be regarded as an indication that a secondary node has performed all
necessary preparations to turn into a primary node.

@param
	ctx: Context used to cancel operation asynchronously.

@return
	[]byte: raw bytes content of journal entry if success
	*Checkpoint: points to an unmarshalled Checkpoint object if success and the entry fetched
	is a Checkpoint entry.
	error: not nil if failed to fetch message or unmarshall Checkpoint,
*/
func (r *Receiver) TryFetchEntry(ctx context.Context) ([]byte, *Checkpoint, error) {
	lag, err := r.entriesReader.ReadLag(ctx)
	if err != nil {
		return nil, nil, err
	}
	if lag == 0 {
		return nil, nil, &NoMoreMessageError{}
	}
	return r.FetchEntry(ctx)
}

/*
Set offset of wrapped kafka.Reader. A secondary node can call this method to
set current offset to a checkpoint, skipping replicated and persisted jouranl
entries.
*/
func (r *Receiver) SetOffset(offset int64) error {
	return r.entriesReader.SetOffset(offset)
}
