package internal

import (
	"fmt"
	_const "github.com/zikwall/grower/pkg/const"
)

type MockStorage struct{}

func (mc MockStorage) Write(topic _const.Topic, partition _const.Partition, message _const.Message) {
	fmt.Println(topic, partition, message)
}

func (mc MockStorage) NewTopic(_ _const.Topic, _ ...int) error {
	return nil
}

func (mc MockStorage) Close() error {
	return nil
}
