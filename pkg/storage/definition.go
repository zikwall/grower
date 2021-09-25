package storage

import (
	"github.com/zikwall/grower/pkg/const"
)

type Storage interface {
	Descriptor(topic _const.Topic, partition _const.Partition) Descriptor
	Write(topic _const.Topic, partition _const.Partition, message _const.Message)
	NewTopic(topic _const.Topic, partitions ...int) error
	HasTopic(topic _const.Topic) bool
	DeleteTopic(topic _const.Topic) error
	Close() error
}

type Descriptor interface {
	Write(messages []_const.Message)
	Read(from, to int64) ([]_const.Message, error)
	Close() error
}
