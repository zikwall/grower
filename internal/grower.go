package internal

import (
	"context"
	"github.com/zikwall/grower/pkg/const"
	"github.com/zikwall/grower/pkg/storage"
	"sync"
	"time"
)

const shutdownWaitDuration = time.Second * 5

type Grower struct {
	ctx         context.Context
	shutdown    chan struct{}
	wg          sync.WaitGroup
	storage     storage.Storage
	listeners   []*Listener
	messagePool map[_const.Topic]chan _const.Message
}

func NewGrower(ctx context.Context, _storage storage.Storage) *Grower {
	grower := &Grower{
		ctx: ctx, wg: sync.WaitGroup{}, storage: _storage, shutdown: make(chan struct{}),
		messagePool: map[_const.Topic]chan _const.Message{},
	}
	return grower
}

func (g *Grower) await() error {
	select {
	case <-g.shutdown:
		return nil
	case <-time.After(shutdownWaitDuration):
		return _const.ErrorShutdownWithoutGracefulCompletion
	}
}

func (g *Grower) down() error {
	// Ждем завершения всех слушателей топиков
	for _, listener := range g.listeners {
		listener.stop()
	}

	go func() {
		// wait all async processes
		g.wg.Wait()
		// to inform about the successful completion of the task
		g.shutdown <- struct{}{}
	}()

	return g.await()
}

func (g *Grower) Drop() error {
	return g.down()
}
