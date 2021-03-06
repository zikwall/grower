package internal

import (
	"context"
	_const "github.com/zikwall/grower/pkg/const"
	"time"
)

const reclaimInterval = time.Millisecond * 150
const batchSize = 15
const optimizedPartitionsSize = 10

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
			case <-ctx.Done():
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
				var offsetsSnapshot map[_const.Partition]int64
				var optimizedUpdate bool

				g.state.mu.RLock()
				waitSnapshot := g.state.waits[topic][group]
				g.state.mu.RUnlock()

				// если идет процесс перебалансировки - ожидаем снятия блокировки (окончания перебалансировки)
				waitSnapshot.Wait()

				g.state.mu.RLock()
				offsetsSnapshot = make(map[_const.Partition]int64, optimizedPartitionsSize)

				// получаем все офсеты для прилинкованных партиций слушателя
				partitionsSnapshot := g.state.consumers[topic][group][uuid]
				for _, partition := range partitionsSnapshot {
					offsetsSnapshot[partition] = g.state.offsets[topic][group][partition]
				}
				g.state.mu.RUnlock()

				for _, partition := range partitionsSnapshot {
					offset := offsetsSnapshot[partition] + batchSize
					messages, err := g.storage.Read(topic, partition, offsetsSnapshot[partition]+1, offset)

					if err != nil {
						// send error to chan of errors
						continue
					}

					if len(messages) > 0 {
						ch <- messages

						// выравниваем смещение в партиции на последнее сообщение
						if int64(len(messages)) < batchSize {
							offset = offset - batchSize + int64(len(messages))
						}
						offsetsSnapshot[partition] = offset
						optimizedUpdate = true
					}
				}

				if optimizedUpdate {
					// commit offset for partitions in consumer group for direct consumer
					g.state.mu.Lock()
					for partition, offset := range offsetsSnapshot {
						g.state.offsets[topic][group][partition] = offset
					}
					g.state.mu.Unlock()
				}
			}
		}
	}()

	// return unsubscribe function
	return func() {
		cancel()
	}
}
