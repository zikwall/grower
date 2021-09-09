package storage

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	_const "github.com/zikwall/grower/pkg/const"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	flushInterval = time.Second * 1
)

type IsomorphicMemoryStorage struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	mu                   sync.RWMutex
	memory               map[_const.Topic]map[_const.Partition][]_const.Message
	createWriterCallback func(topic _const.Topic) (io.WriteCloser, error)
	wg                   sync.WaitGroup
}

func NewIsomorphicMemoryStorage(ctx context.Context, wCb ...func(topic _const.Topic) (io.WriteCloser, error)) *IsomorphicMemoryStorage {
	ctx, cancel := context.WithCancel(ctx)

	is := &IsomorphicMemoryStorage{
		ctx: ctx, cancel: cancel, wg: sync.WaitGroup{},
		memory: map[_const.Topic]map[_const.Partition][]_const.Message{},
	}

	if len(wCb) <= 0 {
		is.createWriterCallback = func(topic _const.Topic) (io.WriteCloser, error) {
			dir, err := os.Getwd()

			if err != nil {
				return nil, err
			}

			return os.Create(fmt.Sprintf("%s/tmp/%s.log", dir, topic))
		}
	} else {
		is.createWriterCallback = wCb[0]
	}

	return is
}

func (is *IsomorphicMemoryStorage) Write(topic _const.Topic, partition _const.Partition, message _const.Message) {
	is.mu.Lock()
	is.memory[topic][partition] = append(is.memory[topic][partition], message)
	is.mu.Unlock()
}

func (is *IsomorphicMemoryStorage) NewTopic(topic _const.Topic, partitions ...int) error {
	exist := false

	is.mu.RLock()
	if _, ok := is.memory[topic]; ok {
		exist = true
	}
	is.mu.RUnlock()

	if exist {
		return errors.New("topic already exists")
	}

	f, err := is.createWriterCallback(topic)

	if err != nil {
		return err
	}

	is.mu.Lock()
	is.memory[topic] = map[_const.Partition][]_const.Message{}

	if len(partitions) > 0 {
		for i := 1; i <= partitions[0]; i++ {
			is.memory[topic][i] = []_const.Message{}
		}
	}

	is.mu.Unlock()

	go is.gc(is.ctx, topic, f)

	return nil
}

func (is *IsomorphicMemoryStorage) Close() error {
	is.cancel()
	is.wg.Wait()
	return nil
}

func (is *IsomorphicMemoryStorage) cl(topic _const.Topic) {
	is.mu.Lock()
	delete(is.memory, topic)
	is.mu.Unlock()
}

func (is *IsomorphicMemoryStorage) flush(topic _const.Topic, w *bufio.Writer) {
	is.mu.Lock()
	if len(is.memory[topic]) > 0 {
		for partition := range is.memory[topic] {
			data := strings.Join(is.memory[topic][partition][:], "\n")
			is.memory[topic][partition] = is.memory[topic][partition][:0]

			_, _ = w.WriteString(data)
			_, _ = w.WriteString("\n")
		}

		_ = w.Flush()
	}
	is.mu.Unlock()
}

func (is *IsomorphicMemoryStorage) gc(ctx context.Context, topic _const.Topic, writer io.WriteCloser) {
	is.wg.Add(1)
	w := bufio.NewWriter(writer)

	ticker := time.NewTicker(flushInterval)

	defer func() {
		is.flush(topic, w)
		is.cl(topic)

		_ = writer.Close()

		is.wg.Done()

		fmt.Printf("stop isomorphic GC for topic %s\n", topic)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			is.flush(topic, w)
		}
	}
}
