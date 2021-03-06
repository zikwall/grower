package internal

import (
	"errors"
	_const "github.com/zikwall/grower/pkg/const"
	"sync"
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

	go g.balancer(topic, partitions)

	g.messagePool[topic] = make(chan _const.Message, 1)

	g.state.mu.Lock()
	g.state.consumers[topic] = map[_const.Group]map[_const.ConsumerUUID][]int{}
	g.state.offsets[topic] = map[_const.Group]map[_const.Partition]int64{}
	g.state.waits[topic] = map[_const.Group]*sync.WaitGroup{}
	g.state.mu.Unlock()

	g.listeners = append(g.listeners, NewListener(
		g.storage, g.messagePool[topic], topic, partitions),
	)

	return nil
}

func (g *Grower) DeleteTopic(topic _const.Topic) error {
	return g.storage.DeleteTopic(topic)
}

func (g *Grower) Write(topic _const.Topic, message _const.Message) {
	g.messagePool[topic] <- message
}
