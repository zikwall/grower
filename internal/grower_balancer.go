package internal

import (
	_const "github.com/zikwall/grower/pkg/const"
	"math"
	"sync"
	"time"
)

func (g *Grower) balancer(topic _const.Topic, partitions _const.Partition) {
	g.wg.Add(1)

	periodicChecker := time.NewTicker(10_000 * time.Millisecond)

	go func() {
		defer func() {
			g.state.mu.Lock()
			delete(g.state.consumers, topic)
			delete(g.state.waits, topic)
			delete(g.state.offsets, topic)
			g.state.mu.Unlock()

			close(g.messagePool[topic])
			delete(g.messagePool, topic)

			g.wg.Done()
		}()

		for {
			select {
			case <-g.ctx.Done():
				return
			case change := <-g.subscriberChanges:
				g.reBalance(topic, partitions, change)
			case <-periodicChecker.C:
				if !g.storage.HasTopic(topic) {
					return
				}
			}
		}
	}()
}

// reBalance rebalances the consumers when adding or removing listening
//
// an example:
// create_topic("rainbow_topic_name", 2) -> topic with two partitions
//
// subscribe("rainbow_topic_name", "soap_consumer_group", () => on receive messages callback) -> uuid_one
// -> reBalance
// --> state = map[uuid_one: [1, 2]]
//
// subscribe("rainbow_topic_name", "soap_consumer_group", () => on receive messages callback) -> uuid_two
// -> reBalance
// --> state = map[uuid_one: [1], uuid_two: [2]]
//
func (g *Grower) reBalance(topic _const.Topic, partitions _const.Partition, change Change) {
	g.state.mu.RLock()
	consumersSnapshot, isExistConsumerGroup := g.state.consumers[topic][change.Group]
	waitSnapshot := g.state.waits[topic][change.Group]
	g.state.mu.RUnlock()

	// Добавляем новую группу слушателей, состояния:
	if !isExistConsumerGroup {
		g.state.mu.Lock()

		g.state.consumers[topic][change.Group] = map[_const.ConsumerUUID][]int{}
		g.state.offsets[topic][change.Group] = map[_const.Partition]int64{}
		g.state.waits[topic][change.Group] = &sync.WaitGroup{}

		consumersSnapshot = g.state.consumers[topic][change.Group]
		waitSnapshot = g.state.waits[topic][change.Group]
		g.state.mu.Unlock()
	}

	// Добавляем ожидание для перебалансировки слушателей
	waitSnapshot.Add(1)

	switch change.Direction {
	case GetOut:
		delete(consumersSnapshot, change.UUID)
		// если у нас не осталось подписчиков подчищаем ресурсы группы
		// в будущем надо будет сохранить офсеты группы подписчиков, иначе при заходе под этой же группой
		// будет происходить не правильное чтение, а именно все начнется заного
		if len(consumersSnapshot) == 0 {
			g.state.mu.Lock()
			delete(g.state.consumers[topic], change.Group)
			delete(g.state.offsets[topic], change.Group)
			delete(g.state.waits[topic], change.Group)
			g.state.mu.Unlock()
		}
	case GetIn:
		consumersSnapshot[change.UUID] = []int{}

		g.state.mu.Lock()
		g.state.offsets[topic][change.Group] = map[_const.Partition]int64{}
		g.state.mu.Unlock()
	}

	partitionSnapshot := map[_const.Partition]struct{}{}

	// Освобождаем все занятые партиции
	for i := 1; i <= partitions; i++ {
		partitionSnapshot[i] = struct{}{}
	}

	// очищаем существующие партиции из прослушивания
	for consumerUUID := range consumersSnapshot {
		consumersSnapshot[consumerUUID] = []int{}
	}

	// Считаем, сколько партиций приходится на одного слушателя.
	// Далее равномерно распределяем слушателей по партициям
	partsForOne := distributionPartitions(partitions, len(consumersSnapshot))
	for consumerUUID := range consumersSnapshot {
		for part := range partitionSnapshot {
			// если слушатель уже "полный", переходим к заполнению следующего
			if len(consumersSnapshot[consumerUUID]) >= partsForOne {
				break
			}

			// линкуем слушателя и партицию
			consumersSnapshot[consumerUUID] = append(consumersSnapshot[consumerUUID], part)
			// удаляем свободную партицию
			delete(partitionSnapshot, part)
		}
	}

	// Заменяем состояния
	g.state.mu.Lock()
	g.state.consumers[topic][change.Group] = consumersSnapshot
	g.state.mu.Unlock()

	// снимаем ожидание перебалансировки слушателей
	waitSnapshot.Done()
}

func distributionPartitions(partitions, consumers int) int {
	partOneConsumer := float64(partitions) / float64(consumers)
	return int(math.Round(partOneConsumer + 0.49))
}
