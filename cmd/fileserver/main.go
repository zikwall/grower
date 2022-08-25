package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"

	"github.com/zikwall/grower/config"
	"github.com/zikwall/grower/internal/services/filegrpc"
	stdout "github.com/zikwall/grower/pkg/log"
	"github.com/zikwall/grower/pkg/signal"
	"github.com/zikwall/grower/protobuf/filebuf"
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
				Name:    "grpc-bind-address",
				Value:   "0.0.0.0:3003",
				Usage:   "Bin address on host",
				EnvVars: []string{"GRPC_BIND_ADDRESS"},
			},
			&cli.IntFlag{
				Name:     "parallelism",
				Usage:    "Number of threads processing logs, default num CPU",
				Required: false,
				Value:    runtime.NumCPU(),
				EnvVars:  []string{"PARALLELISM"},
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
			&cli.DurationFlag{
				Name:    "write-timeout",
				Value:   time.Duration(30) * time.Second,
				Usage:   "Clickhouse Write timout",
				EnvVars: []string{"WRITE_TIMEOUT"},
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
	instance, err := filegrpc.NewServer(appContext, &filegrpc.ServerOpt{
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
		Runtime: config.Runtime{
			Parallelism:  ctx.Int("parallelism"),
			WriteTimeout: ctx.Duration("write-timeout"),
			Debug:        ctx.Bool("debug"),
		},
		Buffer: config.Buffer{
			BufSize:          ctx.Uint("buffer-size"),
			BufFlushInterval: ctx.Uint("buffer-flush-interval"),
		},
		Config:      yamlConfig,
		BindAddress: ctx.String("grpc-bind-address"),
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
		stdout.Info("received a system signal to shut down File Log gRPC Server, start the shutdown process..")
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
	// register and launch gRPC server
	server := grpc.NewServer([]grpc.ServerOption{}...)
	filebuf.RegisterFileBufferServiceServer(server, instance)
	defer func() {
		server.Stop()
		stdout.Info("file log gRPC server is stopped")
	}()
	go func() {
		listener, err := net.Listen("tcp", ctx.String("grpc-bind-address"))
		if err != nil {
			stop(fmt.Errorf("failed to listen: %v", err))
			return
		}
		if err := server.Serve(listener); err != nil {
			stop(fmt.Errorf("failed run gRPC server: %v", err))
			return
		}
	}()
	stdout.Info("congratulations, File Log gRPC Server has been successfully launched")
	instance.Run(instance.Context())
	return await()
}
