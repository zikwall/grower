package main

import (
	"context"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/urfave/cli/v2"

	"github.com/zikwall/grower/config"
	"github.com/zikwall/grower/internal/services/filegrpc"
	stdout "github.com/zikwall/grower/pkg/log"
	"github.com/zikwall/grower/pkg/signal"
)

// nolint:funlen // it's OK
func main() {
	application := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "bind-address",
				Value:   "0.0.0.0:3000",
				Usage:   "Run HTTP server in host",
				EnvVars: []string{"BIND_ADDRESS"},
			},
			&cli.StringFlag{
				Name:    "grpc-conn-address",
				Value:   "0.0.0.0:3003",
				Usage:   "Connect to host",
				EnvVars: []string{"GRPC_CONN_ADDRESS"},
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
	instance, err := filegrpc.NewClient(appContext, &filegrpc.ClientOpt{
		ConnectAddress:              ctx.String("grpc-conn-address"),
		LogsDir:                     ctx.String("logs-dir"),
		SourceLogFile:               ctx.String("source-log-file"),
		ScrapeInterval:              ctx.Duration("scrape-interval"),
		BackupFiles:                 ctx.Uint("backup-files"),
		BackupFileMaxAge:            ctx.Duration("backup-file-max-age"),
		EnableRotating:              ctx.Bool("enable-rotating"),
		AutoCreateTargetFromScratch: ctx.Bool("auto-create-target-from-scratch"),
		RunAtStartup:                ctx.Bool("run-rotating-at-startup"),
		SkipNginxReopen:             ctx.Bool("skip-nginx-reopen"),
		Runtime: config.Runtime{
			Parallelism: ctx.Int("parallelism"),
			Debug:       ctx.Bool("debug"),
		},
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
		stdout.Info("received a system signal to shut down File Log gRPC Client, start the shutdown process..")
	})
	// HTTP server is needed mainly to track viability of the service and for metrics such as prometheus
	if ctx.Bool("run-http-server") {
		// run HTTP server
		go func() {
			app := fiber.New(fiber.Config{
				ServerHeader: "Grower FileLog Server",
			})
			app.Get("/live", func(ctx *fiber.Ctx) error {
				return ctx.Status(200).SendString("Alive")
			})
			ln, err := signal.Listener(
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
	stdout.Info("congratulations, File Log gRPC Client has been successfully launched")
	instance.Run(instance.Context())
	return await()
}
