package syslog

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	clickhousebuffer "github.com/zikwall/clickhouse-buffer/v4"
	"github.com/zikwall/clickhouse-buffer/v4/src/buffer/cxmem"
	"github.com/zikwall/clickhouse-buffer/v4/src/cx"
	"github.com/zikwall/clickhouse-buffer/v4/src/db/cxnative"
	"gopkg.in/mcuadros/go-syslog.v2/format"

	"github.com/zikwall/grower/config"
	"github.com/zikwall/grower/pkg/drop"
	"github.com/zikwall/grower/pkg/handler"
	"github.com/zikwall/grower/pkg/log"
	"github.com/zikwall/grower/pkg/nginx"
	"github.com/zikwall/grower/pkg/wrap"
)

type Syslog struct {
	*drop.Impl
	syslog        *server
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
	config.Runtime
	config.Buffer
	Listeners []string
	Unix      string
	UPD       string
	TCP       string
}

func New(ctx context.Context, opt *Opt) (*Syslog, error) {
	ch, _, err := cxnative.NewClickhouse(ctx, opt.Clickhouse, &cx.RuntimeOptions{
		WriteTimeout: opt.SyslogConfig.WriteTimeout,
	})
	if err != nil {
		return nil, err
	}
	client := clickhousebuffer.NewClientWithOptions(ctx, ch, clickhousebuffer.NewOptions(
		clickhousebuffer.WithFlushInterval(opt.SyslogConfig.BufFlushInterval),
		clickhousebuffer.WithBatchSize(opt.SyslogConfig.BufSize),
		clickhousebuffer.WithDebugMode(opt.SyslogConfig.Debug),
		clickhousebuffer.WithRetry(true),
	))
	columns, scheme := opt.Config.Scheme.MapKeys()
	s := &Syslog{
		Impl:          drop.NewContext(ctx),
		bufferWrapper: wrap.NewBufferWrapper(ch),
		clientWrapper: wrap.NewClientWrapper(client),
		rowHandler: handler.NewRowHandler(
			columns, scheme,
			nginx.NewTemplate(opt.Config.Nginx.LogFormat),
			nginx.NewTypeCaster(&nginx.CasterCfg{
				CustomCasts:       opt.Config.Nginx.LogCustomCasts,
				LocalTimeFormat:   opt.Config.Nginx.LogTimeFormat,
				CustomCastsEnable: opt.Config.Nginx.LogCustomCastsEnable,
				RemoveHyphen:      opt.Config.Nginx.LogRemoveHyphen,
			}),
		),
		syslog: newServer(opt.SyslogConfig),
	}
	s.AddDroppers(
		s.syslog,
		s.clientWrapper,
		s.bufferWrapper,
	)
	writerAPI := s.Buffer().Writer(
		clickhouse.Context(context.Background(), clickhouse.WithSettings(clickhouse.Settings{
			"max_execution_time": opt.SyslogConfig.WriteTimeout.Seconds(),
		})),
		cx.NewView(opt.Config.Scheme.LogsTable, columns),
		cxmem.NewBuffer(
			s.Buffer().Options().BatchSize(),
		),
	)
	s.syslog.setHandler(func(parts format.LogParts) {
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

// Context get root service level context
func (s *Syslog) Context() context.Context {
	return s.Impl.Context()
}

// Buffer get clickhouse buffer client
func (s *Syslog) Buffer() clickhousebuffer.Client {
	return s.clientWrapper.Client()
}

// Run service
func (s *Syslog) Run(ctx context.Context) error {
	return s.syslog.runContext(ctx)
}
