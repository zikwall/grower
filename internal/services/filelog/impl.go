package filelog

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	clickhousebuffer "github.com/zikwall/clickhouse-buffer/v4"
	"github.com/zikwall/clickhouse-buffer/v4/src/buffer/cxmem"
	"github.com/zikwall/clickhouse-buffer/v4/src/cx"
	"github.com/zikwall/clickhouse-buffer/v4/src/db/cxnative"

	"github.com/zikwall/grower/config"
	"github.com/zikwall/grower/pkg/drop"
	"github.com/zikwall/grower/pkg/fileio"
	"github.com/zikwall/grower/pkg/handler"
	"github.com/zikwall/grower/pkg/log"
	"github.com/zikwall/grower/pkg/nginx"
	"github.com/zikwall/grower/pkg/wrap"
)

type FileLog struct {
	*drop.Impl
	bufferWrapper *wrap.BufferWrapper
	clientWrapper *wrap.ClientWrapper
	worker        *Worker
}

type Opt struct {
	Config        *config.Config
	Clickhouse    *clickhouse.Options
	FileLogConfig *Cfg
}

type Cfg struct {
	config.Runtime
	config.Buffer
	LogsDir                     string
	SourceLogFile               string
	ScrapeInterval              time.Duration
	BackupFiles                 uint
	BackupFileMaxAge            time.Duration
	EnableRotating              bool
	AutoCreateTargetFromScratch bool
	RunAtStartup                bool
	SkipNginxReopen             bool
}

func New(ctx context.Context, opt *Opt) (*FileLog, error) {
	var err error
	ch, _, err := cxnative.NewClickhouse(ctx, opt.Clickhouse, &cx.RuntimeOptions{
		WriteTimeout: opt.FileLogConfig.WriteTimeout,
	})
	if err != nil {
		return nil, err
	}
	client := clickhousebuffer.NewClientWithOptions(ctx, ch, clickhousebuffer.DefaultOptions().
		SetFlushInterval(opt.FileLogConfig.BufFlushInterval).
		SetBatchSize(opt.FileLogConfig.BufSize).
		SetDebugMode(opt.FileLogConfig.Debug).
		SetRetryIsEnabled(true),
	)
	columns, scheme := opt.Config.Scheme.MapKeys()
	f := &FileLog{
		Impl:          drop.NewContext(ctx),
		bufferWrapper: wrap.NewBufferWrapper(ch),
		clientWrapper: wrap.NewClientWrapper(client),
	}
	writerAPI := f.Buffer().Writer(
		clickhouse.Context(context.Background(), clickhouse.WithSettings(clickhouse.Settings{
			"max_execution_time": opt.FileLogConfig.WriteTimeout.Seconds(),
		})),
		cx.NewView(opt.Config.Scheme.LogsTable, columns),
		cxmem.NewBuffer(f.Buffer().Options().BatchSize()),
	)
	f.worker = NewWorker(
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
		opt.FileLogConfig,
	)
	f.AddDroppers(
		f.worker,
		f.clientWrapper,
		f.bufferWrapper,
	)
	return f, nil
}

// Context get root service level context
func (f *FileLog) Context() context.Context {
	return f.Impl.Context()
}

// Buffer get clickhouse buffer client
func (f *FileLog) Buffer() clickhousebuffer.Client {
	return f.clientWrapper.Client()
}

// Run service
func (f *FileLog) Run(ctx context.Context) {
	f.worker.runContext(ctx)
}

type Worker struct {
	wg         *sync.WaitGroup
	cfg        *Cfg
	rowHandler handler.Handler
	writer     clickhousebuffer.Writer
	raw        chan string
}

func (w *Worker) Drop() error {
	w.wg.Wait()
	return nil
}

func (w *Worker) DropMsg() string {
	return "kill file log server"
}

func NewWorker(rowHandler handler.Handler, writer clickhousebuffer.Writer, cfg *Cfg) *Worker {
	w := &Worker{
		wg:         &sync.WaitGroup{},
		cfg:        cfg,
		rowHandler: rowHandler,
		writer:     writer,
		raw:        make(chan string),
	}
	return w
}

// create worker pool for handling parsed rows
func (w *Worker) preparePoolContext(ctx context.Context) {
	var i int
	for i = 1; i <= w.cfg.Parallelism; i++ {
		w.wg.Add(1)
		go w.worker(ctx, i)
	}
}

func (w *Worker) worker(ctx context.Context, worker int) {
	if w.cfg.Debug {
		log.Infof("run worker %d", worker)
	}
	defer func() {
		w.wg.Done()
		if w.cfg.Debug {
			log.Infof("stop worker %d", worker)
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case raw := <-w.raw:
			if vector, err := w.rowHandler.Handle(raw); err != nil {
				log.Warning(err)
			} else {
				w.writer.WriteVector(vector)
			}
		}
	}
}

// runContext main loop for read and rotating logs
func (w *Worker) runContext(ctx context.Context) {
	w.preparePoolContext(ctx)
	w.wg.Add(1)
	go func() {
		ticker := time.NewTicker(w.cfg.ScrapeInterval)
		defer func() {
			close(w.raw)
			ticker.Stop()
			w.wg.Done()
			log.Info("stop scrapper worker")
		}()
		if w.cfg.RunAtStartup {
			w.timeHasComeRotate()
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.timeHasComeRotate()
			}
		}
	}()
}

// timeHasComeRotate function starts processing log file, and then deletes outdated logs
func (w *Worker) timeHasComeRotate() {
	if w.cfg.AutoCreateTargetFromScratch {
		// only for local development mode, each cyclo create new access.log from scratch
		// will be removed in future
		fileio.FromScratch(w.cfg.SourceLogFile, w.cfg.LogsDir)
		log.Infof(
			"%s will be generated from scratch and mounted in the directory: %s",
			w.cfg.SourceLogFile,
			w.cfg.LogsDir,
		)
	}
	if err := w.handleFile(w.cfg.LogsDir, w.cfg.SourceLogFile); err != nil {
		log.Warning(err)
	}
	// if rotation option is enabled, we delete outdated log files
	if w.cfg.EnableRotating {
		err := fileio.DeleteOutdatedBackupFiles(
			w.cfg.SourceLogFile,
			w.cfg.LogsDir,
			w.cfg.BackupFiles,
			w.cfg.BackupFileMaxAge,
		)
		if err != nil {
			log.Warning(err)
		}
	}
}

// handleFile rotate target file and handle all rows
func (w *Worker) handleFile(dir, file string) error {
	oldFilepath := path.Join(dir, file)
	if err := fileio.CheckFile(oldFilepath); err != nil {
		return err
	}
	newFilepath := path.Join(dir, fileio.BuildGrowerLogName(file))
	if err := os.Rename(oldFilepath, newFilepath); err != nil {
		return fmt.Errorf("failed to rotate file: %w", err)
	}
	if !w.cfg.SkipNginxReopen {
		// send command to nginx for reopen log file
		if err := exec.Command("nginx", "-s", "reopen").Run(); err != nil {
			return fmt.Errorf("failed to reopen nginx: %w", err)
		}
	}
	f, err := os.OpenFile(newFilepath, os.O_RDONLY, 0o777)
	if err != nil {
		return fmt.Errorf("failed open log file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Warning(err)
		}
		// if rotation is not enabled, just delete the current index file
		if !w.cfg.EnableRotating {
			if err := os.Remove(newFilepath); err != nil {
				log.Warning(err)
			}
		}
	}()
	// Optionally, resize scanner's capacity for lines over 64K.
	// Problem is Scanner.Scan() is limited in a 4096 []byte buffer size per line.
	// We will get bufio.ErrTooLong error, which is bufio.Scanner: token too long if the line is too long.
	// In which case, you'll have to use bufio.ReaderLine() or ReadString()
	scanner := bufio.NewScanner(bufio.NewReader(f))
	for scanner.Scan() {
		w.raw <- scanner.Text()
	}
	if scanner.Err() != nil {
		// todo if receive error save file to temporary directory
		return scanner.Err()
	}
	return nil
}
