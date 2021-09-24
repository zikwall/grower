package internal

import (
	"fmt"
	_const "github.com/zikwall/grower/pkg/const"
	"github.com/zikwall/grower/pkg/storage"
)

type MockStorage struct{}

func (mc MockStorage) Write(topic _const.Topic, partition _const.Partition, message _const.Message) {
	fmt.Println(topic, partition, message)
}

func (mc MockStorage) NewTopic(_ _const.Topic, _ ...int) error {
	return nil
}

func (mc MockStorage) HasTopic(_ _const.Topic) bool {
	return true
}

func (mc MockStorage) DeleteTopic(_ _const.Topic) error {
	return nil
}

func (mc MockStorage) Read(_ _const.Topic, _ _const.Partition, _, _ int64) ([]_const.Message, error) {
	return nil, nil
}

func (mc MockStorage) Close() error {
	return nil
}

func (mc MockStorage) Descriptor(_ _const.Topic, _ _const.Partition) storage.Descriptor {
	return nil
}
