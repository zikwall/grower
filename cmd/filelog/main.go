package main

import (
	"context"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/urfave/cli/v2"

	"github.com/zikwall/ck-nginx/config"
	"github.com/zikwall/ck-nginx/internal/services/filelog"
	stdout "github.com/zikwall/ck-nginx/pkg/log"
	"github.com/zikwall/ck-nginx/pkg/signal"
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
				Name:    "bind-address",
				Value:   "0.0.0.0:3000",
				Usage:   "Run HTTP server in host",
				EnvVars: []string{"BIND_ADDRESS"},
			},
			&cli.StringFlag{
				Name:    "logs-dir",
				Value:   "/var/log/nginx",
				Usage:   "Nginx logs directory",
				EnvVars: []string{"LOGS_DIR"},
			},
			&cli.StringFlag{
				Name:    "source-log-file",
				Value:   "access.log",
				Usage:   "Source log file name",
				EnvVars: []string{"TARGET_LOG_FILE"},
			},
			&cli.IntFlag{
				Name:     "parallelism",
				Usage:    "Number of threads processing logs, default num CPU",
				Required: false,
				Value:    runtime.NumCPU(),
				EnvVars:  []string{"PARALLELISM"},
			},
			&cli.DurationFlag{
				Name:    "scrape-interval",
				Value:   time.Duration(60000) * time.Millisecond,
				Usage:   "Scrape interval",
				EnvVars: []string{"CRAPE_INTERVAL"},
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
			&cli.UintFlag{
				Name:     "buffer-size",
				Usage:    "Clickhouse buffer size",
				Required: false,
				Value:    5000,
				EnvVars:  []string{"BUFFER_SIZE"},
			},
			&cli.UintFlag{
				Name:     "buffer-flush-interval",
				Usage:    "Clickhouse buffer flush interval",
				Required: false,
				Value:    2000,
				EnvVars:  []string{"BUFFER_FLUSH_INTERVAL"},
			},
			&cli.StringSliceFlag{
				Name:     "clickhouse-host",
				Usage:    "Clickhouse connect servers",
				Required: true,
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
				Name:    "enable-rotating",
				EnvVars: []string{"ENABLE_ROTATING"},
				Value:   false,
			},
			&cli.BoolFlag{
				Name:    "run-rotating-at-startup",
				EnvVars: []string{"RUN_ROTATING_AT_STARTUP"},
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
				Name:    "auto-create-target-from-scratch",
				EnvVars: []string{"AUTO_CREATE_TARGET_FROM_SCRATCH"},
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
	yamlConfig, err := config.New(ctx.String("config-file"))
	if err != nil {
		return err
	}
	instance, err := filelog.New(appContext, &filelog.Opt{
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
		FileLogConfig: &filelog.Cfg{
			LogsDir:                     ctx.String("logs-dir"),
			SourceLogFile:               ctx.String("source-log-file"),
			CounterFile:                 ctx.String("counter-file"),
			ScrapeInterval:              ctx.Duration("scrape-interval"),
			BackupFiles:                 ctx.Uint("backup-files"),
			BackupFileMaxAge:            ctx.Duration("backup-file-max-age"),
			EnableRotating:              ctx.Bool("enable-rotating"),
			AutoCreateTargetFromScratch: ctx.Bool("auto-create-target-from-scratch"),
			RunAtStartup:                ctx.Bool("run-rotating-at-startup"),
			SkipNginxReopen:             ctx.Bool("skip-nginx-reopen"),
			RewriteNginxLocalTime:       ctx.Bool("rewrite-nginx-local-time"),
			Runtime: config.Runtime{
				Parallelism: ctx.Int("parallelism"),
				Debug:       ctx.Bool("debug"),
			},
			Buffer: config.Buffer{
				BufSize:          ctx.Uint("buffer-size"),
				BufFlushInterval: ctx.Uint("buffer-flush-interval"),
			},
		},
		Config: yamlConfig,
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
	await, stop := signal.Notifier(func() {
		stdout.Info("received a system signal to shut down FILELOG server, start the shutdown process..")
	})
	if ctx.Bool("run-http-server") {
		// run HTTP server
		go func() {
			app := fiber.New(fiber.Config{
				ServerHeader: "Lime Filelog Server",
			})
			app.Get("/live", func(ctx *fiber.Ctx) error {
				return ctx.Status(200).SendString("Alive")
			})
			ln, err := signal.ResolveListener(
				instance.Context(), signal.ListenerTCP, "", ctx.String("bind-address"),
			)
			if err != nil {
				stop(err)
				return
			}
			if err := app.Listener(ln); err != nil {
				stop(err)
			}
		}()
	}
	stdout.Info("Congratulations, the Filelog service has been successfully launched")
	instance.BootContext(instance.Context())
	return await()
}
