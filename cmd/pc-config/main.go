package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/byuoitav/pc-config/couch"
	"github.com/byuoitav/pc-config/handlers"
	"github.com/labstack/echo"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	var (
		port     int
		logLevel int8

		dbAddress  string
		dbUsername string
		dbPassword string
	)

	pflag.IntVarP(&port, "port", "P", 8080, "port to run the server on")
	pflag.Int8VarP(&logLevel, "log-level", "L", 0, "level to log at. refer to https://godoc.org/go.uber.org/zap/zapcore#Level for options")

	pflag.StringVar(&dbAddress, "db-address", "", "address of the database")
	pflag.StringVar(&dbUsername, "db-username", "", "username for the database")
	pflag.StringVar(&dbPassword, "db-password", "", "password for the database")
	pflag.Parse()

	// build the logger
	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(zapcore.Level(logLevel)),
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding: "json", EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "@",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	plain, err := config.Build()
	if err != nil {
		fmt.Printf("unable to build logger: %s", err)
		os.Exit(1)
	}

	sugared := plain.Sugar()

	if len(dbAddress) == 0 || len(dbUsername) == 0 || len(dbPassword) == 0 {
		sugared.Fatalf("db address, username, and password *must* be set")
	}

	configService := &couch.ConfigService{
		Address:  dbAddress,
		Username: dbUsername,
		Password: dbPassword,
	}

	handlers := handlers.Handlers{
		ConfigService: configService,
	}

	e := echo.New()

	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "healthy")
	})

	e.GET("/:hostname/config", handlers.ConfigForPC)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		sugared.Fatalf("unable to bind listener: %s", err)
	}

	sugared.Infof("Starting server on %s", lis.Addr().String())
	err = e.Server.Serve(lis)
	switch {
	case errors.Is(err, http.ErrServerClosed):
	case err != nil:
		sugared.Fatalf("failed to serve: %s", err)
	}
}
