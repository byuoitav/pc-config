package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/byuoitav/pc-config/couch"
	"github.com/byuoitav/pc-config/handlers"
	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	var (
		port     int
		logLevel string

		dbAddr     string
		dbUsername string
		dbPassword string
		dbInsecure bool
	)

	pflag.IntVarP(&port, "port", "P", 8080, "port to run the server on")
	pflag.StringVarP(&logLevel, "log-level", "L", "", "level to log at. refer to https://godoc.org/go.uber.org/zap/zapcore#Level for options")
	pflag.StringVar(&dbAddr, "db-address", "", "database address")
	pflag.StringVar(&dbUsername, "db-username", "", "database username")
	pflag.StringVar(&dbPassword, "db-password", "", "database password")
	pflag.BoolVar(&dbInsecure, "db-insecure", false, "don't use SSL in database connection")
	pflag.Parse()

	var level zapcore.Level
	if err := level.Set(logLevel); err != nil {
		fmt.Printf("invalid log level: %s\n", err.Error())
		os.Exit(1)
	}

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
			StacktraceKey:  "trace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	log, err := config.Build()
	if err != nil {
		fmt.Printf("unable to build logger: %s", err)
		os.Exit(1)
	}
	defer func() {
		_ = log.Sync()
	}()

	// context for setup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// build the config service
	if dbInsecure {
		dbAddr = "http://" + dbAddr
	} else {
		dbAddr = "https://" + dbAddr
	}

	var csOpts []couch.Option
	if dbUsername != "" {
		csOpts = append(csOpts, couch.WithBasicAuth(dbUsername, dbPassword))
	}

	cs, err := couch.New(ctx, dbAddr, csOpts...)
	if err != nil {
		log.Fatal("unable to create config service", zap.Error(err))
	}

	handlers := handlers.Handlers{
		ConfigService: cs,
	}

	r := gin.New()
	r.Use(gin.Recovery())

	debug := r.Group("/debug")
	debug.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "healthy")
	})

	api := r.Group("/api")
	api.GET("/:hostname/config", handlers.ConfigForPC)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal("unable to bind listener", zap.Error(err))
	}

	log.Info("Starting server", zap.String("on", lis.Addr().String()))
	err = r.RunListener(lis)
	switch {
	case errors.Is(err, http.ErrServerClosed):
	case err != nil:
		log.Fatal("failed to serve", zap.Error(err))
	}
}
