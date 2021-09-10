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

	if err := g.storage.NewTopic(topic, partitions); err != nil {
		return err
	}

	g.messagePool[topic] = make(chan _const.Message)

	g.state.mu.Lock()
	g.state.consumers[topic] = map[_const.Group]map[_const.ConsumerUUID][]int{}
	g.state.offsets[topic] = map[_const.Group]map[_const.ConsumerUUID]map[_const.Partition]int64{}
	g.state.mu.Unlock()

	g.listeners = append(g.listeners, NewListener(
		g.storage, g.messagePool[topic], topic, partitions),
	)

	return nil
}

func (g *Grower) Write(topic _const.Topic, message _const.Message) {
	g.messagePool[topic] <- message
}
