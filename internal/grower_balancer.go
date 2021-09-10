package internal

import (
	_const "github.com/zikwall/grower/pkg/const"
	"math"
)

// функция в данный момент работает только "локально",
// состояние предтся вынести за рамки данной процедуры распеределения
func (g *Grower) balancer(topic _const.Topic, partitions _const.Partition) {
	g.wg.Add(1)

	go func() {
		defer g.wg.Done()

		for {
			select {
			case <-g.ctx.Done():
				return
			case change := <-g.subscriberChanges:
				g.reBalance(topic, partitions, change)
			}
		}
	}()
}

func (g *Grower) reBalance(topic _const.Topic, partitions _const.Partition, change Change) {
	g.state.mu.RLock()
	consumersSnapshot, isExistConsumerGroup := g.state.consumers[topic][change.Group]
	g.state.mu.RUnlock()

	// Добавляем новую группу слушателей, состояния:
	if !isExistConsumerGroup {
		g.state.mu.Lock()

		g.state.consumers[topic][change.Group] = map[_const.ConsumerUUID][]int{}
		g.state.offsets[topic][change.Group] = map[_const.ConsumerUUID]map[_const.Partition]int64{}

		consumersSnapshot = g.state.consumers[topic][change.Group]
		g.state.mu.Unlock()
	}

	switch change.Direction {
	case GetOut:
		delete(consumersSnapshot, change.UUID)
	case GetIn:
		consumersSnapshot[change.UUID] = []int{}
	}

	partitionSnapshot := map[_const.Partition]struct{}{}

	// Освобождаем все занятые партиции
	for i := 1; i <= partitions; i++ {
		partitionSnapshot[i] = struct{}{}
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
}

func distributionPartitions(partitions, consumers int) int {
	partOneConsumer := float64(partitions) / float64(consumers)
	return int(math.Round(partOneConsumer + 0.49))
}
