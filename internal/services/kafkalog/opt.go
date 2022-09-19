package kafkalog

import (
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/segmentio/kafka-go"

	"github.com/zikwall/grower/config"
)

type Opt struct {
	ClientOpt
	ServerOpt
	Config       *config.Config
	AsyncFactor  uint
	Debug        bool
	KafkaBrokers []string
	KafkaTopic   string
}

type ClientOpt struct {
	KafkaBalancer               string
	KafkaAsync                  bool
	KafkaCreateTopic            bool
	KafkaWriteTimeout           time.Duration
	LogsDir                     string
	SourceLogFile               string
	ScrapeInterval              time.Duration
	BackupFiles                 uint
	BackupFileMaxAge            time.Duration
	EnableRotating              bool
	AutoCreateTargetFromScratch bool
	RunAtStartup                bool
	SkipNginxReopen             bool
	RewriteNginxLocalTime       bool
}

type ServerOpt struct {
	KafkaGroupID     string
	Clickhouse       *clickhouse.Options
	BufSize          uint
	BufFlushInterval uint
	WriteTimeout     time.Duration
}

type Balancer string

func (b Balancer) Match() kafka.Balancer {
	switch b {
	case "round_robin":
		return &kafka.RoundRobin{}
	case "least_bytes":
		return &kafka.LeastBytes{}
	case "hash":
		return &kafka.Hash{}
	case "reference_hash":
		return &kafka.ReferenceHash{}
	}
	return &kafka.LeastBytes{}
}
