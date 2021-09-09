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

		consumers := map[_const.Group]map[int64][]int{}
		freePartitions := map[_const.Group]map[_const.Partition]struct{}{}

		for {
			select {
			case <-g.ctx.Done():
				return
			case change := <-g.subscriberChanges:
				switch change.Direction {
				case GetOut:
					delete(consumers[change.Group], change.UUID)
				case GetIn:
					// Добавляем нового слушателя
					consumers[change.Group][change.UUID] = []int{}
					freePartitions[change.Group] = map[_const.Partition]struct{}{}

					// Освобождаем все занятые партиции
					for i := 1; i <= partitions; i++ {
						freePartitions[change.Group][i] = struct{}{}
					}
				}

				// Считаем, сколько партиций приходится на одного слушателя.
				// Далее равномерно распределяем слушателей по партициям
				partsForOne := distributionPartitions(partitions, len(consumers[change.Group]))
				for consumer := range consumers[change.Group] {
					for part := range freePartitions[change.Group] {
						// если слушатель уже "полный", переходим к заполнению следующего
						if len(consumers[change.Group][consumer]) >= partsForOne {
							break
						}

						// линкуем слушателя и партицию
						consumers[change.Group][consumer] = append(consumers[change.Group][consumer], part)
						// удаляем свободную партицию
						delete(freePartitions[change.Group], part)
					}
				}
			}
		}
	}()
}

func distributionPartitions(partitions, consumers int) int {
	partOneConsumer := float64(partitions) / float64(consumers)
	return int(math.Round(partOneConsumer + 0.49))
}
