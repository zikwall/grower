package storage

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	_const "github.com/zikwall/grower/pkg/const"
	"github.com/zikwall/grower/pkg/storage/file"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	flushInterval = time.Millisecond * 300
)

type TopicContext struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func newTopicContext(ctx context.Context) *TopicContext {
	topicContext := &TopicContext{}
	topicContext.ctx, topicContext.cancel = context.WithCancel(ctx)
	return topicContext
}

type WriterCallback = func(topic _const.Topic, partition _const.Partition) (*os.File, error)

func buildDefaultWriterCallback() WriterCallback {
	return func(topic _const.Topic, partition _const.Partition) (*os.File, error) {
		dir, err := os.Getwd()
		if err != nil {
			return nil, err
		}

		return os.Create(fmt.Sprintf("%s/tmp/%s-%d.log", dir, topic, partition))
	}
}

type IsomorphicMemoryStorage struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	mu                   sync.RWMutex
	memory               map[_const.Topic]map[_const.Partition][]_const.Message
	descriptor           map[_const.Topic]map[_const.Partition]*os.File
	topicsContext        map[_const.Topic]*TopicContext
	createWriterCallback func(topic _const.Topic, partition _const.Partition) (*os.File, error)
	wg                   sync.WaitGroup
}

func NewIsomorphicMemoryStorage(ctx context.Context, wCb ...WriterCallback) *IsomorphicMemoryStorage {
	ctx, cancel := context.WithCancel(ctx)

	is := &IsomorphicMemoryStorage{
		ctx: ctx, cancel: cancel, wg: sync.WaitGroup{}, mu: sync.RWMutex{},
		memory:        map[_const.Topic]map[_const.Partition][]_const.Message{},
		descriptor:    map[_const.Topic]map[_const.Partition]*os.File{},
		topicsContext: map[_const.Topic]*TopicContext{},
	}

	if len(wCb) == 0 {
		is.createWriterCallback = buildDefaultWriterCallback()
	} else {
		is.createWriterCallback = wCb[0]
	}

	return is
}

func (is *IsomorphicMemoryStorage) periodicallyCheckResources() {
	defer log.Println("isomorphic memory resources cleaner is stopped")
	ticker := time.NewTicker(10_000 * time.Millisecond)

	for {
		select {
		case <-is.ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			is.mu.RLock()
			topicContextSnapshot := is.topicsContext
			is.mu.RUnlock()

			for topic := range topicContextSnapshot {
				if is.HasTopic(topic) {
					is.mu.Lock()
					delete(is.topicsContext, topic)
					is.mu.Unlock()
				}
			}
		}
	}
}

func (is *IsomorphicMemoryStorage) Read(
	topic _const.Topic, partition _const.Partition, from, to int64,
) ([]_const.Message, error) {
	is.mu.RLock()
	descriptor := is.descriptor[topic][partition]
	is.mu.RUnlock()

	messages, err := file.Read(descriptor, from, to)

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
	is.descriptor[topic] = map[_const.Partition]*os.File{}
	is.topicsContext[topic] = newTopicContext(is.ctx)
	is.mu.Unlock()

	// Инициализуруем ресурсы: хранилище в памяти, а также объекты данных (файлы), разделенные по партициям
	for partition := 1; partition <= parts; partition++ {
		readWrite, err := is.createWriterCallback(topic, partition)

		if err != nil {
			return err
		}

		is.mu.Lock()
		is.memory[topic][partition] = []_const.Message{}
		is.descriptor[topic][partition] = readWrite
		is.mu.Unlock()

		is.wg.Add(1)
		go is.gc(topic, partition)
	}

	return nil
}

func (is *IsomorphicMemoryStorage) HasTopic(topic _const.Topic) bool {
	is.mu.RLock()
	_, has := is.memory[topic]
	is.mu.RUnlock()
	return has
}

func (is *IsomorphicMemoryStorage) DeleteTopic(topic _const.Topic) error {
	is.mu.Lock()
	is.topicsContext[topic].cancel()

	delete(is.memory, topic)
	delete(is.descriptor, topic)

	is.mu.Unlock()

	return nil
}

func (is *IsomorphicMemoryStorage) Close() error {
	is.cancel()
	is.wg.Wait()

	is.mu.Lock()
	for k := range is.memory {
		delete(is.memory, k)
	}

	for k := range is.descriptor {
		delete(is.descriptor, k)
	}

	for k := range is.topicsContext {
		delete(is.topicsContext, k)
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

func (is *IsomorphicMemoryStorage) gc(topic _const.Topic, partition _const.Partition) {
	is.mu.RLock()
	descriptor := is.descriptor[topic][partition]
	topicContext := is.topicsContext[topic]
	is.mu.RUnlock()

	writer := bufio.NewWriter(descriptor)
	ticker := time.NewTicker(flushInterval)

	defer fmt.Printf("stop isomorphic GC for topic %s and partition %d\n", topic, partition)
	for {
		select {
		case <-topicContext.ctx.Done():
			is.flush(topic, partition, writer)
			is.clean(topic, partition)
			_ = descriptor.Close()
			ticker.Stop()
			is.wg.Done()

			return
		case <-ticker.C:
			is.flush(topic, partition, writer)
		}
	}
}
