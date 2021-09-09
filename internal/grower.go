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
	ctx      context.Context
	shutdown chan struct{}
	wg       sync.WaitGroup
	storage  storage.Storage
}

func NewGrower(ctx context.Context, _storage storage.Storage) *Grower {
	grower := &Grower{ctx: ctx, wg: sync.WaitGroup{}, storage: _storage}
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
