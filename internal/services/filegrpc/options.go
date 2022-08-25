package filegrpc

import (
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"

	"github.com/zikwall/grower/config"
)

type ClientOpt struct {
	config.Runtime
	ConnectAddress              string
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

type ServerOpt struct {
	config.Runtime
	config.Buffer
	Config      *config.Config
	BindAddress string
	Clickhouse  *clickhouse.Options
}
