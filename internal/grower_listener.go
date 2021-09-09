package internal

import (
	_const "github.com/zikwall/grower/pkg/const"
	"github.com/zikwall/grower/pkg/storage"
	"sync"
)

type Listener struct {
	done    chan struct{}
	storage storage.Storage
	wg      sync.WaitGroup
}

func NewListener(
	s storage.Storage, channel <-chan _const.Message, topic _const.Topic, partitions _const.Partition,
) *Listener {
	ln := &Listener{storage: s, done: make(chan struct{}), wg: sync.WaitGroup{}}

	for i := partitions; i <= partitions; i++ {
		ln.wg.Add(1)
		go ln.listen(topic, i, channel)
	}

	return ln
}

func (ln *Listener) stop() {
	ln.done <- struct{}{}
	ln.wg.Wait()
}

func (ln *Listener) listen(topic _const.Topic, partition _const.Partition, messages <-chan _const.Message) {
	defer ln.wg.Done()

	for {
		select {
		case <-ln.done:
			return
		case message := <-messages:
			ln.storage.Write(topic, partition, message)
		}
	}
}
