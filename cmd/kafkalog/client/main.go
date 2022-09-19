package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/zikwall/grower/internal/services/kafkalog"
	stdout "github.com/zikwall/grower/pkg/log"
	"github.com/zikwall/grower/pkg/signal"
)

// nolint:funlen // it's OK
func main() {
	application := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "kafka-topic",
				Required: true,
				Usage:    "Kafka topic name",
				EnvVars:  []string{"KAFKA_TOPIC"},
			},
			&cli.BoolFlag{
				Name:    "kafka-create-topic",
				EnvVars: []string{"KAFKA_CREATE_TOPIC"},
				Value:   false,
			},
			&cli.StringSliceFlag{
				Name:     "kafka-brokers",
				Required: true,
				Usage:    "Connect to brokers",
				EnvVars:  []string{"KAFKA_BROKERS"},
			},
			&cli.BoolFlag{
				Name:    "kafka-async",
				EnvVars: []string{"KAFKA_ASYNC"},
				Value:   false,
			},
			&cli.StringFlag{
				Name:    "kafka-balancer",
				Value:   "least_bytes",
				Usage:   "Balancer for write to kafka: round_robin, hash, reference_hash, least_bytes",
				EnvVars: []string{"KAFKA_BALANCER"},
			},
			&cli.DurationFlag{
				Name:    "kafka-write-timeout",
				Value:   5 * time.Second,
				Usage:   "Kafka write timeout",
				EnvVars: []string{"KAFKA_WRITE_TIMEOUT"},
			},
			&cli.StringFlag{
				Name:    "logs-dir",
				Value:   "/var/log/nginx",
				Usage:   "Logs directory",
				EnvVars: []string{"LOGS_DIR"},
			},
			&cli.StringFlag{
				Name:    "source-log-file",
				Value:   "access.log",
				Usage:   "Source log file name",
				EnvVars: []string{"TARGET_LOG_FILE"},
			},
			&cli.UintFlag{
				Name:    "async-factor",
				Value:   10,
				Usage:   "Number of run parallel workers",
				EnvVars: []string{"ASYNC_FACTOR"},
			},
			&cli.DurationFlag{
				Name:    "scrape-interval",
				Value:   time.Duration(60000) * time.Millisecond,
				Usage:   "Scrape interval",
				EnvVars: []string{"SCRAPE_INTERVAL"},
			},
			&cli.UintFlag{
				Name:     "backup-files",
				Usage:    "Count of backup files",
				Required: false,
				Value:    5,
				EnvVars:  []string{"BACKUP_FILES"},
			},
			&cli.DurationFlag{
				Name:    "backup-file-max-age",
				Value:   time.Duration(60000*5) * time.Millisecond,
				Usage:   "Backup file max age",
				EnvVars: []string{"BACKUP_FILE_MAX_AGE"},
			},
			&cli.BoolFlag{
				Name:    "auto-create-target-from-scratch",
				EnvVars: []string{"AUTO_CREATE_TARGET_FROM_SCRATCH"},
				Value:   false,
			},
			&cli.BoolFlag{
				Name:    "enable-rotating",
				EnvVars: []string{"ENABLE_ROTATING"},
				Value:   false,
			},
			&cli.BoolFlag{
				Name:    "run-at-startup",
				EnvVars: []string{"RUN_AT_STARTUP"},
				Value:   false,
			},
			&cli.BoolFlag{
				Name:    "run-http-server",
				EnvVars: []string{"RUN_HTTP_SERVER"},
				Value:   false,
			},
			&cli.BoolFlag{
				Name:    "skip-nginx-reopen",
				EnvVars: []string{"SKIP_NGINX_REOPEN"},
				Value:   false,
			},
			&cli.BoolFlag{
				Name:    "rewrite-nginx-local-time",
				EnvVars: []string{"REWRITE_NGINX_LOCAL_TIME"},
				Value:   false,
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
	instance, err := kafkalog.NewClient(appContext, &kafkalog.Opt{
		ClientOpt: kafkalog.ClientOpt{
			KafkaAsync:                  ctx.Bool("kafka-async"),
			KafkaCreateTopic:            ctx.Bool("kafka-create-topic"),
			KafkaBalancer:               ctx.String("kafka-balancer"),
			KafkaWriteTimeout:           ctx.Duration("kafka-write-timeout"),
			LogsDir:                     ctx.String("logs-dir"),
			SourceLogFile:               ctx.String("source-log-file"),
			ScrapeInterval:              ctx.Duration("scrape-interval"),
			BackupFiles:                 ctx.Uint("backup-files"),
			BackupFileMaxAge:            ctx.Duration("backup-file-max-age"),
			EnableRotating:              ctx.Bool("enable-rotating"),
			AutoCreateTargetFromScratch: ctx.Bool("auto-create-target-from-scratch"),
			RunAtStartup:                ctx.Bool("run-at-startup"),
			SkipNginxReopen:             ctx.Bool("skip-nginx-reopen"),
			RewriteNginxLocalTime:       ctx.Bool("rewrite-nginx-local-time"),
		},
		KafkaBrokers: ctx.StringSlice("kafka-brokers"),
		KafkaTopic:   ctx.String("kafka-topic"),
		AsyncFactor:  ctx.Uint("async-factor"),
		Debug:        ctx.Bool("debug"),
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
		stdout.Info("received a system signal to shut down Kafka writer, start the shutdown process..")
	})
	stdout.Info("congratulations, kafka writer service has been successfully launched")
	instance.Run(instance.Context())
	return await()
}
