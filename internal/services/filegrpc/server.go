package filegrpc

import (
	"context"
	"io"
	"sync"

	"github.com/ClickHouse/clickhouse-go/v2"
	clickhousebuffer "github.com/zikwall/clickhouse-buffer/v4"
	"github.com/zikwall/clickhouse-buffer/v4/src/buffer/cxmem"
	"github.com/zikwall/clickhouse-buffer/v4/src/cx"
	"github.com/zikwall/clickhouse-buffer/v4/src/db/cxnative"

	"github.com/zikwall/grower/pkg/drop"
	"github.com/zikwall/grower/pkg/handler"
	"github.com/zikwall/grower/pkg/log"
	"github.com/zikwall/grower/pkg/nginx"
	"github.com/zikwall/grower/pkg/wrap"
	"github.com/zikwall/grower/protobuf/filebuf"
)

type Server struct {
	filebuf.UnimplementedFileBufferServiceServer
	*drop.Impl
	bufferWrapper *wrap.BufferWrapper
	clientWrapper *wrap.ClientWrapper
	worker        *FileServerWorker
}

func NewServer(ctx context.Context, opt *ServerOpt) (*Server, error) {
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
	writerAPI := s.Buffer().Writer(
		clickhouse.Context(context.Background(), clickhouse.WithSettings(clickhouse.Settings{
			"max_execution_time": opt.WriteTimeout.Seconds(),
		})),
		cx.NewView(opt.Config.Scheme.LogsTable, columns),
		cxmem.NewBuffer(s.Buffer().Options().BatchSize()),
	)
	s.worker = NewWorker(
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
	s.AddDroppers(
		s.worker,
		s.clientWrapper,
		s.bufferWrapper,
	)
	return s, nil
}

// CreateDataStreamer creates a constant stream receiving data from the client
func (s *Server) CreateDataStreamer(server filebuf.FileBufferService_CreateDataStreamerServer) error {
	for {
		req, err := server.Recv()
		if err == io.EOF {
			return server.SendAndClose(&filebuf.Response{})
		}
		if err != nil {
			return err
		}
		s.worker.str <- req.Data
	}
}

// Context get root service level context
func (s *Server) Context() context.Context {
	return s.Impl.Context()
}

// Buffer get clickhouse buffer client
func (s *Server) Buffer() clickhousebuffer.Client {
	return s.clientWrapper.Client()
}

// Run service
func (s *Server) Run(ctx context.Context) {
	s.worker.preparePool(ctx)
}

type FileServerWorker struct {
	handler handler.Handler
	writer  clickhousebuffer.Writer
	wg      *sync.WaitGroup
	opt     *ServerOpt
	str     chan string
}

func NewWorker(hand handler.Handler, writer clickhousebuffer.Writer, opt *ServerOpt) *FileServerWorker {
	w := &FileServerWorker{
		handler: hand,
		writer:  writer,
		wg:      &sync.WaitGroup{},
		opt:     opt,
		str:     make(chan string),
	}
	return w
}

func (w *FileServerWorker) Drop() error {
	w.wg.Wait()
	return nil
}

func (w *FileServerWorker) DropMsg() string {
	return "kill file gRPC server"
}

func (w *FileServerWorker) preparePool(ctx context.Context) {
	for i := 1; i <= w.opt.Parallelism; i++ {
		w.wg.Add(1)
		go w.makeReceiver(ctx, i)
	}
}

func (w *FileServerWorker) makeReceiver(ctx context.Context, worker int) {
	if w.opt.Debug {
		log.Infof("run server gRPC worker %d", worker)
	}
	defer func() {
		w.wg.Done()
		if w.opt.Debug {
			log.Infof("stop server gRPC  worker %d", worker)
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case str := <-w.str:
			vector, err := w.handler.Handle(str)
			if err != nil {
				log.Warning(err)
				continue
			}
			w.writer.WriteVector(vector)
		}
	}
}
