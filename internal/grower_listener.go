package internal

import (
	_const "github.com/zikwall/grower/pkg/const"
	"github.com/zikwall/grower/pkg/storage"
	"sync/atomic"
)

type Listener struct {
	done        chan struct{}
	storage     storage.Storage
	writePolicy WritePolicy
}

type WritePolicy interface {
	Partition() int
}

type RoundRobinWritePolicy struct {
	partitions []int
	next       uint32
}

func NewRoundRobinWritePolicy(partitions int) *RoundRobinWritePolicy {
	rr := &RoundRobinWritePolicy{}
	for partition := 1; partition <= partitions; partition++ {
		rr.partitions = append(rr.partitions, partition)
	}
	return rr
}

func (rr *RoundRobinWritePolicy) Partition() int {
	return rr.partitions[(int(atomic.AddUint32(&rr.next, 1))-1)%len(rr.partitions)]
}

func NewListener(
	s storage.Storage, channel chan _const.Message, topic _const.Topic, partitions _const.Partition,
) *Listener {
	ln := &Listener{storage: s, done: make(chan struct{}), writePolicy: NewRoundRobinWritePolicy(partitions)}

	go ln.listen(topic, channel)

	return ln
}

func (ln *Listener) stop() {
	close(ln.done)
}

func (ln *Listener) listen(topic _const.Topic, messages chan _const.Message) {
	for {
		select {
		case <-ln.done:
			return
		case message := <-messages:
			partition := ln.writePolicy.Partition()
			ln.storage.Write(topic, partition, message)
		}
	}
}
