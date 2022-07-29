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
	"github.com/zikwall/ck-nginx/internal/services/syslog"
	stdout "github.com/zikwall/ck-nginx/pkg/log"
	"github.com/zikwall/ck-nginx/pkg/signal"
)

// nolint:funlen // it's not important here
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
			&cli.StringSliceFlag{
				Name:     "listeners",
				Usage:    "Run syslog listeners in interfaces",
				Required: false,
				Value:    cli.NewStringSlice(syslog.ListenerUDS),
				EnvVars:  []string{"LISTENERS"},
			},
			&cli.StringFlag{
				Name:     "syslog-unix-socket",
				Usage:    "Path to UNIX socket file",
				Required: false,
				Value:    "/tmp/syslog.sock",
				EnvVars:  []string{"SYSLOG_UNIX_SOCKET"},
			},
			&cli.StringFlag{
				Name:     "syslog-udp-address",
				Value:    "0.0.0.0:3011",
				Required: false,
				Usage:    "Syslog server UDP address",
				EnvVars:  []string{"SYSLOG_UDP_ADDRESS"},
			},
			&cli.StringFlag{
				Name:     "syslog-tcp-address",
				Value:    "0.0.0.0:3012",
				Required: false,
				Usage:    "Syslog server TCP address",
				EnvVars:  []string{"SYSLOG_TCP_ADDRESS"},
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
			&cli.IntFlag{
				Name:     "parallelism",
				Usage:    "Number of threads processing logs, default num CPU",
				Required: false,
				Value:    runtime.NumCPU(),
				EnvVars:  []string{"PARALLELISM"},
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
				Name:    "run-http-server",
				EnvVars: []string{"RUN_HTTP_SERVER"},
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
	instance, err := syslog.New(appContext, &syslog.Opt{
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
		SyslogConfig: &syslog.Cfg{
			Listeners: ctx.StringSlice("listeners"),
			Unix:      ctx.String("syslog-unix-socket"),
			UPD:       ctx.String("syslog-udp-address"),
			TCP:       ctx.String("syslog-tcp-address"),
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
		stdout.Info("received a system signal to shut down SYSLOG server, start the shutdown process..")
	})
	if ctx.Bool("run-http-server") {
		// add metrics
		go func() {
			app := fiber.New(fiber.Config{
				ServerHeader: "CK-NGINX: Syslog Server",
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
	go func() {
		if err := instance.Await(instance.Context()); err != nil {
			stop(err)
		}
	}()
	stdout.Info("congratulations, the Syslog service has been successfully launched")
	return await()
}
