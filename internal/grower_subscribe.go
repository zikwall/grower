package internal

import (
	"context"
	_const "github.com/zikwall/grower/pkg/const"
	"time"
)

const reclaimInterval = time.Millisecond * 100
const batchSize = 15

const (
	GetOut = iota
	GetIn
)

type Subscriber interface {
	Subscribe(_const.Topic, _const.Group, func(..._const.Message)) func()
}

type Change struct {
	Direction int
	Topic     _const.Topic
	Group     _const.Group
	UUID      int64
}

func (g *Grower) Subscribe(
	topic _const.Topic, group _const.Group, onMessages func(messages ..._const.Message),
) func() {
	ch := make(chan []_const.Message, 1)
	ctx, cancel := context.WithCancel(g.ctx)
	uuid := g.subscriberCreateUUID()
	g.subscriberGetIn(topic, group, uuid)

	go func() {
		for {
			select {
			case <-g.ctx.Done():
				return
			case messages := <-ch:
				onMessages(messages...)
			}
		}
	}()

	go func() {
		defer g.subscriberGetOut(topic, group, uuid)

		ticker := time.NewTicker(reclaimInterval)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				g.state.mu.RLock()
				partitionsSnapshot := g.state.consumers[topic][group][uuid]
				offsetSnapshot := g.state.offsets[topic][group][uuid]
				g.state.mu.RUnlock()

				for _, partition := range partitionsSnapshot {
					offset := offsetSnapshot[partition] + batchSize

					messages, err := g.storage.Read(topic, partition, offsetSnapshot[partition], offset)

					if err != nil {
						// send error to chan of errors
						continue
					}

					if len(messages) > 0 {
						ch <- messages
						offsetSnapshot[partition] = offset
					}
				}

				// commit offset for partitions in consumer group for direct consumer
				g.state.mu.Lock()
				g.state.offsets[topic][group][uuid] = offsetSnapshot
				g.state.mu.Unlock()
			}
		}
	}()

	// return unsubscribe function
	return func() {
		cancel()
	}
}
