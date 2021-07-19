package common_journal

import (
	stdctx "context"
	"github.com/fourstring/sheetfs/tests"
	. "github.com/smartystreets/goconvey/convey"
	"sync"
	"testing"
)

var kafkaServer = "127.0.0.1:9093"
var ctx = stdctx.Background()

func TestWriteAndReceive(t *testing.T) {
	Convey("write test messages", t, func() {
		entries := []string{"111", "222", "333", "444", "555"}
		topic := tests.RandStr(10)
		receiver, err := NewReceiver(kafkaServer, topic)
		So(err, ShouldBeNil)
		_, _, err = receiver.TryFetchEntry(ctx)
		So(err, ShouldBeError, &NoMoreMessageError{})
		writer, err := NewWriter(kafkaServer, topic)
		So(err, ShouldBeNil)
		for _, entry := range entries {
			err := writer.CommitEntry(ctx, []byte(entry))
			So(err, ShouldBeNil)
		}
		Convey("read test messages concurrently", func(c C) {
			var wg sync.WaitGroup
			for i := 0; i < 2; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					receiver, err := NewReceiver(kafkaServer, topic)
					c.So(err, ShouldBeNil)
					for i := 0; i < len(entries); i++ {
						msg, _, err := receiver.TryFetchEntry(ctx)
						if i <= len(entries)-1 {
							c.So(err, ShouldBeNil)
							c.So(string(msg), ShouldResemble, entries[i])
						} else {
							c.So(err, ShouldBeError, &NoMoreMessageError{})
						}
					}
				}()
			}
			wg.Wait()
		})
	})
}

func TestCheckpoint(t *testing.T) {
	Convey("write test messages", t, func() {
		entries := []string{"111", "222", "333", "444", "555"}
		topic := tests.RandStr(10)
		writer, err := NewWriter(kafkaServer, topic)
		So(err, ShouldBeNil)
		for _, entry := range entries {
			err := writer.CommitEntry(ctx, []byte(entry))
			So(err, ShouldBeNil)
		}
		writer.PrepareCheckpoint()
		offset, err := writer.Checkpoint(ctx)
		writer.ExitCheckpoint()
		So(err, ShouldBeNil)
		So(offset, ShouldEqual, len(entries)+1)
		Convey("read test messages concurrently", func(c C) {
			var wg sync.WaitGroup
			for i := 0; i < 2; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					receiver, err := NewReceiver(kafkaServer, topic)
					c.So(err, ShouldBeNil)
					for i := 0; i < len(entries)+1; i++ {
						msg, ckpt, err := receiver.TryFetchEntry(ctx)
						if i <= len(entries)-1 {
							c.So(err, ShouldBeNil)
							c.So(ckpt, ShouldBeNil)
							c.So(string(msg), ShouldResemble, entries[i])
						} else if i == len(entries) {
							c.So(err, ShouldBeNil)
							c.So(ckpt, ShouldNotBeNil)
							c.So(ckpt.LastEntryOffset, ShouldEqual, len(entries)-1)
							c.So(ckpt.NextEntryOffset, ShouldEqual, len(entries)+1)
							err := receiver.SetOffset(ckpt.NextEntryOffset)
							c.So(err, ShouldBeNil)
						} else {
							c.So(err, ShouldBeError, &NoMoreMessageError{})
						}
					}
					_, _, err = receiver.TryFetchEntry(ctx)
					c.So(err, ShouldBeError, &NoMoreMessageError{})
				}()
			}
			wg.Wait()
		})
	})
}
