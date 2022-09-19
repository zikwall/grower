package kafkalog

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/segmentio/kafka-go"
	clickhousebuffer "github.com/zikwall/clickhouse-buffer/v4"
	"github.com/zikwall/clickhouse-buffer/v4/src/buffer/cxmem"
	"github.com/zikwall/clickhouse-buffer/v4/src/cx"
	"github.com/zikwall/clickhouse-buffer/v4/src/db/cxnative"

	"github.com/zikwall/grower/pkg/drop"
	"github.com/zikwall/grower/pkg/handler"
	"github.com/zikwall/grower/pkg/log"
	"github.com/zikwall/grower/pkg/nginx"
	"github.com/zikwall/grower/pkg/wrap"
)

type Server struct {
	*drop.Impl
	worker        *ServerWorker
	bufferWrapper *wrap.BufferWrapper
	clientWrapper *wrap.ClientWrapper
}

func NewServer(ctx context.Context, opt *Opt) (*Server, error) {
	var err error
	ch, _, err := cxnative.NewClickhouse(ctx, opt.Clickhouse, &cx.RuntimeOptions{
		WriteTimeout: opt.WriteTimeout,
	})
	if err != nil {
		return nil, err
	}
	client := clickhousebuffer.NewClientWithOptions(ctx, ch, clickhousebuffer.NewOptions(
		clickhousebuffer.WithFlushInterval(opt.BufFlushInterval),
		clickhousebuffer.WithBatchSize(opt.BufSize),
		clickhousebuffer.WithDebugMode(opt.Debug),
		clickhousebuffer.WithRetry(true),
	))
	s := &Server{
		Impl:          drop.NewContext(ctx),
		bufferWrapper: wrap.NewBufferWrapper(ch),
		clientWrapper: wrap.NewClientWrapper(client),
	}
	columns, scheme := opt.Config.Scheme.MapKeys()
	writerAPI := s.clientWrapper.Client().Writer(
		clickhouse.Context(context.Background(), clickhouse.WithSettings(clickhouse.Settings{
			"max_execution_time": opt.ServerOpt.WriteTimeout.Seconds(),
		})),
		cx.NewView(opt.Config.Scheme.LogsTable, columns),
		cxmem.NewBuffer(s.clientWrapper.Client().Options().BatchSize()),
	)
	server, err := NewServerWorker(
		ctx,
		handler.NewRowHandler(
			columns, scheme,
			nginx.NewTemplate(opt.Config.Nginx.LogFormat),
			nginx.NewTypeCaster(&nginx.CasterCfg{
				CustomCasts:       opt.Config.Nginx.LogCustomCasts,
				LocalTimeFormat:   opt.Config.Nginx.LogTimeFormat,
				CustomCastsEnable: opt.Config.Nginx.LogCustomCastsEnable,
				RemoveHyphen:      opt.Config.Nginx.LogRemoveHyphen,
			}),
		),
		writerAPI,
		opt,
	)
	if err != nil {
		return nil, err
	}
	s.worker = server
	s.AddDroppers(
		s.worker,
		s.clientWrapper,
		s.bufferWrapper,
	)
	return s, nil
}

func (w *Server) Run(ctx context.Context) {
	w.worker.preparePool(ctx)
}

type ServerWorker struct {
	opt      *Opt
	wg       *sync.WaitGroup
	handler  handler.Handler
	writer   clickhousebuffer.Writer
	isClosed uint32
}

func (s *ServerWorker) Drop() error {
	atomic.StoreUint32(&s.isClosed, 1)
	s.wg.Wait()
	log.Info("stop all reader")
	return nil
}

func (s *ServerWorker) DropMsg() string {
	return "kill kafka server"
}

// create worker pool for handling parsed rows
func (s *ServerWorker) preparePool(ctx context.Context) {
	var i uint
	for i = 1; i <= s.opt.AsyncFactor; i++ {
		s.wg.Add(1)
		go s.makeReaderListener(ctx, i)
	}
}

func (s *ServerWorker) makeReaderListener(ctx context.Context, worker uint) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: s.opt.KafkaBrokers,
		Topic:   s.opt.KafkaTopic,
		GroupID: s.opt.KafkaGroupID,
	})
	defer func() {
		if err := r.Close(); err != nil {
			log.Warningf("close reader err: %v", err)
		}
		s.wg.Done()
		if s.opt.Debug {
			log.Infof("stop kafka reader %d", worker)
		}
	}()
	if s.opt.Debug {
		log.Infof("run kafka reader %d", worker)
	}
	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			break
		}
		if s.opt.Debug {
			fmt.Printf("kafka [reader %d] [partiton %d] at offset %d: %s\n",
				worker, m.Partition, m.Offset, string(m.Key),
			)
		}
		vector, err := s.handler.Handle(string(m.Value))
		if err != nil {
			log.Warning(err)
			continue
		}
		s.writer.WriteVector(vector)
	}
}

func NewServerWorker(
	ctx context.Context,
	rowHandler handler.Handler,
	writer clickhousebuffer.Writer,
	opt *Opt,
) (*ServerWorker, error) {
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
	s := &ServerWorker{
		handler: rowHandler,
		writer:  writer,
		opt:     opt,
		wg:      &sync.WaitGroup{},
	}
	return s, nil
}
