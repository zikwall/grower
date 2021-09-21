package storage

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	_const "github.com/zikwall/grower/pkg/const"
	"github.com/zikwall/grower/pkg/storage/file"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	flushInterval = time.Millisecond * 300
)

type IsomorphicMemoryStorage struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	mu                   sync.RWMutex
	memory               map[_const.Topic]map[_const.Partition][]_const.Message
	reader               map[_const.Topic]map[_const.Partition]*os.File
	createWriterCallback func(topic _const.Topic, partition _const.Partition) (*os.File, error)
	wg                   sync.WaitGroup
}

func NewIsomorphicMemoryStorage(ctx context.Context,
	wCb ...func(topic _const.Topic, partition _const.Partition) (*os.File, error),
) *IsomorphicMemoryStorage {
	ctx, cancel := context.WithCancel(ctx)

	is := &IsomorphicMemoryStorage{
		ctx: ctx, cancel: cancel, wg: sync.WaitGroup{}, mu: sync.RWMutex{},
		memory: map[_const.Topic]map[_const.Partition][]_const.Message{},
		reader: map[_const.Topic]map[_const.Partition]*os.File{},
	}

	if len(wCb) == 0 {
		is.createWriterCallback = func(topic _const.Topic, partition _const.Partition) (*os.File, error) {
			dir, err := os.Getwd()

			if err != nil {
				return nil, err
			}

			readWrite, err := os.Create(fmt.Sprintf("%s/tmp/%s-%d.log", dir, topic, partition))

			if err != nil {
				return nil, err
			}

			return readWrite, nil
		}
	} else {
		is.createWriterCallback = wCb[0]
	}

	return is
}

func (is *IsomorphicMemoryStorage) Read(
	topic _const.Topic, partition _const.Partition, from, to int64,
) ([]_const.Message, error) {
	is.mu.RLock()
	reader := is.reader[topic][partition]
	is.mu.RUnlock()

	messages, err := file.Read(reader, from, to)

	if err != nil {
		return nil, err
	}

	return messages, nil
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

	parts := 1
	if len(partitions) > 0 {
		parts = partitions[0]
	}

	is.mu.Lock()
	is.memory[topic] = map[_const.Partition][]_const.Message{}
	is.reader[topic] = map[_const.Partition]*os.File{}
	is.mu.Unlock()

	// Инициализуруем ресурсы: хранилище в памяти, а также объекты данных (файлы), разделенные по партициям
	for partition := 1; partition <= parts; partition++ {
		readWrite, err := is.createWriterCallback(topic, partition)

		if err != nil {
			return err
		}

		is.wg.Add(1)

		is.mu.Lock()
		is.memory[topic][partition] = []_const.Message{}
		is.reader[topic][partition] = readWrite
		is.mu.Unlock()

		go is.gc(is.ctx, topic, partition)
	}

	return nil
}

func (is *IsomorphicMemoryStorage) HasTopic(topic _const.Topic) bool {
	is.mu.RLock()
	_, has := is.memory[topic]
	is.mu.RUnlock()
	return has
}

func (is *IsomorphicMemoryStorage) Close() error {
	is.cancel()
	is.wg.Wait()

	is.mu.Lock()
	for k := range is.memory {
		delete(is.memory, k)
	}

	for k := range is.reader {
		delete(is.reader, k)
	}
	is.mu.Unlock()

	return nil
}

func (is *IsomorphicMemoryStorage) clean(topic _const.Topic, partition _const.Partition) {
	is.mu.Lock()
	delete(is.memory[topic], partition)
	is.mu.Unlock()
}

func (is *IsomorphicMemoryStorage) flush(topic _const.Topic, partition _const.Partition, w *bufio.Writer) {
	is.mu.Lock()

	var data string
	if messages := is.memory[topic][partition]; len(messages) > 0 {
		data = strings.Join(messages, "\n")
		is.memory[topic][partition] = is.memory[topic][partition][:0]
	}

	is.mu.Unlock()

	if data != "" {
		_, _ = w.WriteString(data)
		_, _ = w.WriteString("\n")
		_ = w.Flush()
	}
}

func (is *IsomorphicMemoryStorage) gc(ctx context.Context, topic _const.Topic, partition _const.Partition) {
	is.mu.RLock()
	writer := is.reader[topic][partition]
	is.mu.RUnlock()

	w := bufio.NewWriter(writer)

	ticker := time.NewTicker(flushInterval)

	defer func() {
		_ = writer.Close()
		ticker.Stop()
		is.wg.Done()

		fmt.Printf("stop isomorphic GC for topic %s\n", topic)
	}()

	for {
		select {
		case <-ctx.Done():
			is.flush(topic, partition, w)
			is.clean(topic, partition)

			return
		case <-ticker.C:
			is.flush(topic, partition, w)
		}
	}
}
