package main

import (
	"context"

	"log"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"

	"github.com/urfave/cli/v2"

	"github.com/zikwall/grower/config"
	"github.com/zikwall/grower/internal/services/kafkalog"
	stdout "github.com/zikwall/grower/pkg/log"
	"github.com/zikwall/grower/pkg/signal"
)

// nolint:funlen // it's OK
func main() {
	application := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "config-file",
				Required: true,
				Usage:    "YAML config filepath",
				EnvVars:  []string{"CONFIG_FILE"},
				FilePath: "/srv/vp_secret/config_file",
			},
			&cli.StringFlag{
				Name:     "kafka-topic",
				Required: true,
				Usage:    "Kafka topic name",
				EnvVars:  []string{"KAFKA_TOPIC"},
			},
			&cli.StringSliceFlag{
				Name:     "kafka-brokers",
				Required: true,
				Usage:    "Connect to brokers",
				EnvVars:  []string{"KAFKA_BROKERS"},
			},
			&cli.StringFlag{
				Name:     "kafka-group",
				Required: true,
				Usage:    "Kafka group",
				EnvVars:  []string{"KAFKA_GROUP"},
			},
			&cli.UintFlag{
				Name:    "async-factor",
				Value:   10,
				Usage:   "Number of run parallel workers",
				EnvVars: []string{"ASYNC_FACTOR"},
			},
			&cli.UintFlag{
				Name:     "buffer-size",
				Usage:    "Размер буфера syslog",
				Required: false,
				Value:    10000,
				EnvVars:  []string{"BUFFER_SIZE"},
			},
			&cli.UintFlag{
				Name:     "buffer-flush-interval",
				Usage:    "Интервал сброса буфера syslog в миллисекундах",
				Required: false,
				Value:    5000,
				EnvVars:  []string{"BUFFER_FLUSH_INTERVAL"},
			},
			&cli.DurationFlag{
				Name:    "write-timeout",
				Value:   time.Duration(30) * time.Second,
				Usage:   "Clickhouse Write timout",
				EnvVars: []string{"WRITE_TIMEOUT"},
			},
			&cli.StringSliceFlag{
				Name:     "clickhouse-host",
				Usage:    "Hosts",
				EnvVars:  []string{"CLICKHOUSE_HOST"},
				FilePath: "/srv/vp_secret/clickhouse_host",
			},
			&cli.StringFlag{
				Name:     "clickhouse-user",
				Usage:    "Clickhouse server user",
				EnvVars:  []string{"CLICKHOUSE_USER"},
				FilePath: "/srv/vp_secret/clickhouse_user",
			},
			&cli.StringFlag{
				Name:     "clickhouse-password",
				Usage:    "Clickhouse server user password",
				EnvVars:  []string{"CLICKHOUSE_PASSWORD"},
				FilePath: "/srv/vp_secret/clickhouse_password",
			},
			&cli.StringFlag{
				Name:     "clickhouse-database",
				Usage:    "Clickhouse server database name",
				EnvVars:  []string{"CLICKHOUSE_DATABASE"},
				FilePath: "/srv/vp_secret/clickhouse_database",
			},
			&cli.BoolFlag{
				Name:    "debug",
				EnvVars: []string{"DEBUG"},
				Value:   false,
			},
		},
		Action: Main,
	}
	if err := application.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func Main(ctx *cli.Context) error {
	appContext, cancel := context.WithCancel(ctx.Context)
	defer func() {
		cancel()
		<-time.After(time.Second)
		stdout.Info("app context is canceled, service is down!")
	}()
	yamlConfig, err := config.New(ctx.String("config-file"))
	if err != nil {
		return err
	}
	instance, err := kafkalog.NewServer(appContext, &kafkalog.Opt{
		ServerOpt: kafkalog.ServerOpt{
			KafkaGroupID: ctx.String("kafka-group"),
			Clickhouse: &clickhouse.Options{
				Addr: ctx.StringSlice("clickhouse-host"),
				Auth: clickhouse.Auth{
					Database: ctx.String("clickhouse-database"),
					Username: ctx.String("clickhouse-username"),
					Password: ctx.String("clickhouse-password"),
				},
				Settings: clickhouse.Settings{
					"max_execution_time": 60,
				},
				DialTimeout: 5 * time.Second,
				Compression: &clickhouse.Compression{
					Method: clickhouse.CompressionLZ4,
				},
				Debug: ctx.Bool("debug"),
			},
			BufSize:          ctx.Uint("buffer-size"),
			BufFlushInterval: ctx.Uint("buffer-flush-interval"),
			WriteTimeout:     ctx.Duration("write-timeout"),
		},
		KafkaBrokers: ctx.StringSlice("kafka-brokers"),
		KafkaTopic:   ctx.String("kafka-topic"),
		Debug:        ctx.Bool("debug"),
		AsyncFactor:  ctx.Uint("async-factor"),
		Config:       yamlConfig,
	})
	if err != nil {
		return err
	}
	defer func() {
		instance.Shutdown(func(err error) {
			stdout.Warning(err)
		})
		instance.Stacktrace()
	}()
	await, _ := signal.Notifier(func() {
		stdout.Info("received a system signal to shut down kafka reader server, start the shutdown process..")
	})
	stdout.Info("congratulations, kafka reader server has been successfully launched")
	instance.Run(instance.Context())
	return await()
}
