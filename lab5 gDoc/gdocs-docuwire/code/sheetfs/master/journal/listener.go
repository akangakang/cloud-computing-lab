package journal

import (
	"context"
	"errors"
	"github.com/fourstring/sheetfs/common_journal"
	"github.com/fourstring/sheetfs/election"
	"github.com/fourstring/sheetfs/master/filemgr"
	"github.com/fourstring/sheetfs/master/journal/checkpoint"
	entry2 "github.com/fourstring/sheetfs/master/journal/journal_entry"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

type ListenerConfig struct {
	NodeID      string
	Elector     *election.Elector
	KafkaServer string
	KafkaTopic  string
	FileManager *filemgr.FileManager
	DB          *gorm.DB
}

type Listener struct {
	elector  *election.Elector
	receiver *common_journal.Receiver
	fm       *filemgr.FileManager
	db       *gorm.DB
	logger   *zap.Logger
}

func NewListener(config *ListenerConfig) (*Listener, error) {
	receiver, err := common_journal.NewReceiver(config.KafkaServer, config.KafkaTopic)
	if err != nil {
		return nil, err
	}
	logger, err := zap.NewDevelopment(zap.Fields(
		zap.String("source", "Listener"),
		zap.String("NodeID", config.NodeID),
	))
	if err != nil {
		return nil, err
	}
	return &Listener{
		elector:  config.Elector,
		receiver: receiver,
		fm:       config.FileManager,
		db:       config.DB,
		logger:   logger,
	}, nil
}

func (l *Listener) RunAsSecondary() error {
	defer l.logger.Sync()

	ckptOffset := checkpoint.ReadCheckpoint(l.db)
	err := l.receiver.SetOffset(ckptOffset)

	if err != nil {
		l.logger.Error("error when loading checkpoint offset.", zap.Error(err))
		return err
	}
	for {
		success, watch, notify, err := l.elector.TryBeLeader()
		if err != nil {
			l.logger.Error("error when run as secondary.", zap.Error(err))
			return err
		}
		if success {
			break
		}
		l.logger.Debug("run as secondary.", zap.String("watch", watch))
		ctx := common_journal.NewZKEventCancelContext(context.Background(), notify)
		for {
			msg, ckpt, err := l.receiver.FetchEntry(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					break
				}
				l.logger.Error("error when run as secondary.", zap.Error(err))
				return err
			}
			err = l.handleJournal(msg, ckpt)
			if err != nil {
				l.logger.Error("error when handling journal.", zap.Error(err))
				return err
			}
			_ = l.logger.Sync()
		}
	}
	for {
		msg, ckpt, err := l.receiver.TryFetchEntry(context.Background())
		if err != nil {
			if errors.Is(err, &common_journal.NoMoreMessageError{}) {
				break
			}
			l.logger.Error("error when fast forwarding remaining journal.", zap.Error(err))
			return err
		}
		err = l.handleJournal(msg, ckpt)
		if err != nil {
			l.logger.Error("error when fast forwarding remaining journal.", zap.Error(err))
			return err
		}
	}
	return nil
}

func (l *Listener) handleCheckpoint(ckpt *common_journal.Checkpoint) error {
	err := l.fm.Persistent()
	if err != nil {
		return err
	}
	return checkpoint.RecordCheckpoint(l.db, ckpt.NextEntryOffset)
}

func (l *Listener) handleJournal(entry []byte, ckpt *common_journal.Checkpoint) error {
	defer l.logger.Sync()
	if ckpt != nil {
		err := l.handleCheckpoint(ckpt)
		if err != nil {
			l.logger.Error("error when recording checkpoint.", zap.Error(err))
			return err
		}
	} else {
		var masterEntry entry2.MasterEntry
		err := proto.Unmarshal(entry, &masterEntry)
		if err != nil {
			l.logger.Error("error when unmarshalling journal entry.", zap.Error(err))
			return err
		}
		err = l.fm.HandleMasterEntry(&masterEntry)
		if err != nil {
			l.logger.Error("error when applying master journal entry.", zap.Error(err))
			return err
		}
	}
	return nil
}
