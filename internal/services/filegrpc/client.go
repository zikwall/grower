package filegrpc

import (
	"bufio"
	"context"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/zikwall/grower/pkg/drop"
	"github.com/zikwall/grower/pkg/fileio"
	"github.com/zikwall/grower/pkg/log"
	"github.com/zikwall/grower/protobuf/filebuf"
)

type Client struct {
	*drop.Impl
	worker *ClientWorker
}

func NewClient(ctx context.Context, opt *ClientOpt) (*Client, error) {
	worker, err := NewClientWorker(opt)
	if err != nil {
		return nil, err
	}
	w := &Client{
		Impl:   drop.NewContext(ctx),
		worker: worker,
	}
	w.AddDropper(worker)
	return w, nil
}

// Run service
func (c *Client) Run(ctx context.Context) {
	c.worker.runContext(ctx)
}

type ClientWorker struct {
	wg       *sync.WaitGroup
	str      chan string
	opt      *ClientOpt
	client   filebuf.FileBufferServiceClient
	conn     *grpc.ClientConn
	rotate   fileio.Rotator
	isClosed uint32
	senders  uint32
}

func NewClientWorker(opt *ClientOpt) (*ClientWorker, error) {
	conn, err := grpc.Dial(opt.ConnectAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	w := &ClientWorker{
		wg:     &sync.WaitGroup{},
		opt:    opt,
		str:    make(chan string),
		client: filebuf.NewFileBufferServiceClient(conn),
		conn:   conn,
	}
	w.rotate = fileio.New(
		opt.SourceLogFile,
		opt.LogsDir,
		opt.BackupFiles,
		opt.BackupFileMaxAge,
		opt.AutoCreateTargetFromScratch,
		opt.EnableRotating,
		opt.SkipNginxReopen,
		w.handleFile,
	)
	return w, nil
}

func (w *ClientWorker) Drop() error {
	// mark client is closed
	atomic.StoreUint32(&w.isClosed, 1)
	// wait all workers
	w.wg.Wait()
	// close gRPC client connection
	return w.conn.Close()
}

func (w *ClientWorker) DropMsg() string {
	return "kill file gRPC client"
}

// prepareClientWorkerPool create worker pool for handling parsed rows
func (w *ClientWorker) preparePool(ctx context.Context) {
	for i := 1; i <= w.opt.Parallelism; i++ {
		w.wg.Add(1)
		go w.makeSender(ctx, i)
	}
	// wait for ready, refactor in the future, temporary way
	<-time.After(500 * time.Millisecond)
}

func (w *ClientWorker) makeSender(ctx context.Context, worker int) {
	defer func() {
		w.wg.Done()
		if w.opt.Debug {
			log.Infof("stop client gRPC  worker %d", worker)
		}
	}()
	if w.opt.Debug {
		log.Infof("run client gRPC worker %d", worker)
	}
	stream, err := w.client.CreateDataStreamer(ctx)
	if err != nil {
		log.Warningf("stream create: %v", err)
		return
	}
	defer func() {
		if resp, err := stream.CloseAndRecv(); err != nil {
			log.Warningf("stream close and receive: response %s with error: %v", resp.String(), err)
		}
	}()
	atomic.AddUint32(&w.senders, 1)
	defer func() {
		atomic.AddUint32(&w.senders, ^uint32(0))
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case str := <-w.str:
			if err := stream.Send(&filebuf.Request{Data: str}); err != nil {
				log.Warningf("send stream: %s", err.Error())
			}
		}
	}
}

// runContext main loop for read and rotating logs
func (w *ClientWorker) runContext(ctx context.Context) {
	w.preparePool(ctx)
	// run it only if and only if there is at least one writer on the server,
	// otherwise a deadlock may occur
	if atomic.LoadUint32(&w.senders) > 0 {
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
				if err := w.rotate.Rotate(); err != nil {
					log.Warning(err)
				}
			}
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := w.rotate.Rotate(); err != nil {
						log.Warning(err)
					}
				}
			}
		}()
	} else {
		log.Warning("main worker not running, service is useless, please restart")
	}
}

// handleFile rotate target file and handle all rows
func (w *ClientWorker) handleFile(file *os.File) error {
	// Optionally, resize scanner's capacity for lines over 64K.
	// Problem is Scanner.Scan() is limited in a 4096 []byte buffer size per line.
	// We will get bufio.ErrTooLong error, which is bufio.Scanner: token too long if the line is too long.
	// In which case, you'll have to use bufio.ReaderLine() or ReadString()
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
