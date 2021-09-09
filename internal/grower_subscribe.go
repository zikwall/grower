package internal

import (
	_const "github.com/zikwall/grower/pkg/const"
	"time"
)

const reclaimInterval = time.Second * 1

func (g *Grower) Subscribe(topic _const.Topic, group _const.Group, onMessages func(messages ..._const.Message)) {
	ch := make(chan []_const.Message, 10)

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

	ticker := time.NewTicker(reclaimInterval)

	go func() {
		for {
			select {
			case <-g.ctx.Done():
				return
			case <-ticker.C:
				// Читаем топик и партиции для определенной группы подписчика
				// ```go
				//	for partition := range partitions {
				//     	offset := getOffset(topic, partition, consumer)
				//		current := getCurrentLen(topic, partition)
				//
				//		генерируем новое смещение newOffset := offset + batchSize
				//
				//		messages := storage.Read(topic, partition, offset, newOffset)
				//
				//		сохраняем за пользователем новый офсет
				//		commitOffset(topic, partition, consumer, newOffset)
				//
				//		пишем в канал для подписчика
				//		ch <- messages
				//	}
				// ```
			}
		}
	}()
}
