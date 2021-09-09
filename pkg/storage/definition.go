package storage

import (
	"github.com/zikwall/grower/pkg/const"
)

type Storage interface {
	Write(topic _const.Topic, partition _const.Partition, message _const.Message)
	NewTopic(topic _const.Topic, partitions ...int) error
	Close() error
}
