package syslog

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	clickhousebuffer "github.com/zikwall/clickhouse-buffer/v3"
	"github.com/zikwall/clickhouse-buffer/v3/src/buffer/cxmem"
	"github.com/zikwall/clickhouse-buffer/v3/src/cx"
	"github.com/zikwall/clickhouse-buffer/v3/src/db/cxnative"
	"gopkg.in/mcuadros/go-syslog.v2/format"

	"github.com/zikwall/ck-nginx/config"
	"github.com/zikwall/ck-nginx/pkg/drop"
	"github.com/zikwall/ck-nginx/pkg/handler"
	"github.com/zikwall/ck-nginx/pkg/log"
	"github.com/zikwall/ck-nginx/pkg/wrap"
)

type Syslog struct {
	*drop.Impl
	syslog        *Server
	bufferWrapper *wrap.BufferWrapper
	clientWrapper *wrap.ClientWrapper
	rowHandler    handler.Handler
}

type Opt struct {
	Config       *config.Config
	Clickhouse   *clickhouse.Options
	SyslogConfig *Cfg
}

type Cfg struct {
	Listeners        []string
	Unix             string
	UPD              string
	TCP              string
	BufSize          uint
	BufFlushInterval uint
	Debug            bool
}

func New(ctx context.Context, opt *Opt) (*Syslog, error) {
	ch, _, err := cxnative.NewClickhouse(ctx, opt.Clickhouse)
	if err != nil {
		return nil, err
	}
	client := clickhousebuffer.NewClientWithOptions(ctx, ch,
		clickhousebuffer.DefaultOptions().
			SetFlushInterval(opt.SyslogConfig.BufFlushInterval).
			SetBatchSize(opt.SyslogConfig.BufSize+1).
			SetDebugMode(opt.SyslogConfig.Debug).
			SetRetryIsEnabled(true),
	)
	s := &Syslog{
		bufferWrapper: wrap.NewBufferWrapper(ch),
		clientWrapper: wrap.NewClientWrapper(client),
		rowHandler:    handler.NewRowHandler(false),
		syslog:        NewServer(opt.SyslogConfig),
	}
	s.Impl = drop.NewContext(ctx)
	s.AddDroppers(s.clientWrapper, s.bufferWrapper)
	writerAPI := s.Buffer().Writer(
		cx.NewView(opt.Config.Scheme.LogsTable, opt.Config.Scheme.MapKeys()),
		cxmem.NewBuffer(
			s.Buffer().Options().BatchSize(),
		),
	)
	s.syslog.SetHandler(func(parts format.LogParts) {
		if value, ok := parts["content"]; ok && value != "" {
			vector, err := s.rowHandler.Handle(fmt.Sprintf("%v", parts["content"]))
			if err != nil {
				log.Warning(err)
				return
			}
			writerAPI.WriteVector(vector)
		}
	})
	return s, nil
}

func (s *Syslog) Context() context.Context {
	return s.Impl.Context()
}

func (s *Syslog) Await() error {
	return s.syslog.Await()
}

func (s *Syslog) Buffer() clickhousebuffer.Client {
	return s.clientWrapper.Client()
}
