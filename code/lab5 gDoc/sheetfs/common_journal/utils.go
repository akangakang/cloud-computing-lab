package common_journal

import (
	"context"
	"errors"
	"github.com/go-zookeeper/zk"
	"github.com/segmentio/kafka-go"
	"time"
)

/*
Returns a Context which will be asynchronously cancelled by a channel emitting
zk.Event. Such a context can be used to cancel blocking FetchEntry operations
when a secondary node becomes a primary one.
*/
func NewZKEventCancelContext(ctx context.Context, notify <-chan zk.Event) context.Context {
	c, cancel := context.WithCancel(ctx)
	go func() {
		<-notify
		cancel()
	}()
	return c
}

func ensureJournalTopicCreated(server, topic string) error {
	client := &kafka.Client{
		Addr:    kafka.TCP(server),
		Timeout: 10 * time.Second,
	}

	_, err := client.CreateTopics(context.Background(), &kafka.CreateTopicsRequest{
		Addr: kafka.TCP(server),
		Topics: []kafka.TopicConfig{
			{
				Topic:             topic,
				NumPartitions:     1,
				ReplicationFactor: 1,
				ConfigEntries:     []kafka.ConfigEntry{},
			},
		},
		ValidateOnly: false,
	})
	if err != nil && !errors.Is(err, kafka.TopicAlreadyExists) {
		return err
	}
	return nil
}
