package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"imagepp/internal/config"
	"imagepp/internal/db"
	"imagepp/internal/handler"
	"imagepp/internal/scheduler"
	"imagepp/pkg/logger"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()
	logg := logger.New()
	ctx := context.Background()

	// Initialize database pool
	if err := db.Open(cfg); err != nil {
		logg.Fatal().Err(err).Msg("failed to initialize database pool")
	}
	defer db.Close()

	dbpool := db.Get()
	logg.Info().Msg("database pool initialized")
	opt, _ := redis.ParseURL(cfg.RedisUrl)
	rdb := redis.NewClient(opt)

	if err := rdb.Ping(ctx).Err(); err != nil {
		logg.Fatal().Err(err).Msg("redis connection failed")
	}
	logg.Info().Msg("redis connected")

	// Initialize scheduler client for task enqueueing
	schedulerClient := scheduler.NewClient(asynq.RedisClientOpt{
		Addr:      opt.Addr,
		Username:  opt.Username,
		Password:  opt.Password,
		DB:        opt.DB,
		TLSConfig: opt.TLSConfig, // Important for rediss:// URLs!
	})
	logg.Info().Msg("scheduler client initialized")

	router := handler.NewRouter(logg, cfg, dbpool, rdb, schedulerClient)

	server := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logg.Info().
			Str("port", cfg.AppPort).
			Msg("server starting")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logg.Fatal().Err(err).Msg("server failed to start")
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logg.Info().Msg("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// graceful HTTP shutdown
	if err := server.Shutdown(shutdownCtx); err != nil {
		logg.Error().Err(err).Msg("server shutdown failed")
	}

	// close resources
	if err := rdb.Close(); err != nil {
		logg.Error().Err(err).Msg("redis close failed")
	}
	if err := schedulerClient.Close(); err != nil {
		logg.Error().Err(err).Msg("scheduler client close failed")
	}

	logg.Info().Msg("server stopped cleanly")
}
