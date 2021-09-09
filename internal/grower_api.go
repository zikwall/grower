package internal

import (
	"errors"
	_const "github.com/zikwall/grower/pkg/const"
)

func (g *Grower) CreateTopic(topic _const.Topic, partitions _const.Partition) error {
	if topic == "" {
		return errors.New("topic name can not be empty")
	}

	if partitions == 0 {
		partitions = 1
	}

	return g.storage.NewTopic(topic, partitions)
}
