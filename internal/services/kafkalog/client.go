package kafkalog

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/zikwall/grower/pkg/drop"
	"github.com/zikwall/grower/pkg/fileio"
	"github.com/zikwall/grower/pkg/log"
)

type Client struct {
	*drop.Impl
	worker *ClientWorker
}

func NewClient(ctx context.Context, opt *Opt) (*Client, error) {
	worker, err := NewClientWorker(ctx, opt)
	if err != nil {
		return nil, err
	}
	client := &Client{
		Impl:   drop.NewContext(ctx),
		worker: worker,
	}
	client.AddDropper(client.worker)
	return client, nil
}

func (c *Client) Run(ctx context.Context) {
	c.worker.runContext(ctx)
}

type ClientWorker struct {
	writer   *kafka.Writer
	rotator  fileio.Rotator
	opt      *Opt
	str      chan string
	wg       *sync.WaitGroup
	isClosed uint32
}

func NewClientWorker(ctx context.Context, opt *Opt) (*ClientWorker, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	// ping leader node
	conn, err := kafka.DialLeader(ctx, "tcp", opt.KafkaBrokers[0], opt.KafkaTopic, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to dial leader: %v", err)
	}
	if err := conn.Close(); err != nil {
		log.Warningf("failed to close connection: %v", err)
	}
	log.Info("all nodes are successfully connected")
	w := &kafka.Writer{
		Addr:         kafka.TCP(opt.KafkaBrokers...),
		Topic:        opt.KafkaTopic,
		Balancer:     Balancer(opt.KafkaBalancer).Match(),
		Async:        opt.KafkaAsync,
		WriteTimeout: opt.KafkaWriteTimeout,
	}
	c := &ClientWorker{
		writer: w,
		opt:    opt,
		str:    make(chan string),
		wg:     &sync.WaitGroup{},
	}
	c.rotator = fileio.New(
		opt.SourceLogFile,
		opt.LogsDir,
		opt.BackupFiles,
		opt.BackupFileMaxAge,
		opt.AutoCreateTargetFromScratch,
		opt.EnableRotating,
		opt.SkipNginxReopen,
		c.handleFile,
	)
	// nolint:staticcheck // it's ok
	if opt.KafkaCreateTopic {
		// todo
	}
	return c, nil
}

func (w *ClientWorker) Write(ctx context.Context, message string) error {
	ctx, cancel := context.WithTimeout(ctx, w.opt.KafkaWriteTimeout)
	defer cancel()
	return w.writer.WriteMessages(ctx, kafka.Message{
		Value: []byte(message),
	})
}

func (w *ClientWorker) Drop() error {
	atomic.StoreUint32(&w.isClosed, 1)
	w.wg.Wait()
	log.Info("stop all workers")
	err := w.writer.Close()
	log.Info("close kafka writer")
	return err
}

func (w *ClientWorker) DropMsg() string {
	return "kill kafka client"
}

// create worker pool for handling parsed rows
func (w *ClientWorker) preparePool(ctx context.Context) {
	var i uint
	for i = 1; i <= w.opt.AsyncFactor; i++ {
		w.wg.Add(1)
		go w.makeWriteListener(ctx, i)
	}
}

func (w *ClientWorker) makeWriteListener(ctx context.Context, worker uint) {
	defer func() {
		w.wg.Done()
		if w.opt.Debug {
			log.Infof("stop kafka writer %d", worker)
		}
	}()
	if w.opt.Debug {
		log.Infof("run kafka writer %d", worker)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case str := <-w.str:
			if err := w.Write(ctx, str); err != nil {
				log.Warningf("write to kafka: %s", err.Error())
			}
		}
	}
}

// runContext main loop for read and rotating logs
func (w *ClientWorker) runContext(ctx context.Context) {
	w.preparePool(ctx)
	w.wg.Add(1)
	go func() {
		ticker := time.NewTicker(w.opt.ScrapeInterval)
		defer func() {
			ticker.Stop()
			close(w.str)
			w.wg.Done()
			log.Info("stop rotate worker")
		}()
		if w.opt.RunAtStartup {
			if err := w.rotator.Rotate(); err != nil {
				log.Warning(err)
			}
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := w.rotator.Rotate(); err != nil {
					log.Warning(err)
				}
			}
		}
	}()
}

// handleFile rotate target file and handle all rows
func (w *ClientWorker) handleFile(file *os.File) error {
	scanner := bufio.NewScanner(bufio.NewReader(file))
	for scanner.Scan() {
		if atomic.LoadUint32(&w.isClosed) == 1 {
			break
		}
		w.str <- scanner.Text()
	}
	if scanner.Err() != nil {
		return scanner.Err()
	}
	return nil
}
