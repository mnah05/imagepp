package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"imagepp/internal/config"
	"imagepp/internal/db"
	jobs "imagepp/internal/jobs"
	workers "imagepp/internal/workers"
	"imagepp/pkg/logger"

	"github.com/hibiken/asynq"
	redis "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

const TypeHealthCheck = "system:health_check"

func main() {
	cfg := config.Load()
	logg := logger.New()

	// Initialize database pool
	if err := db.Open(cfg); err != nil {
		logg.Fatal().Err(err).Msg("failed to initialize database pool")
	}
	defer db.Close()

	dbpool := db.Get()
	logg.Info().Msg("database pool initialized")

	workers.Queries = db.New(dbpool)

	opt, err := redis.ParseURL(cfg.RedisUrl)
	if err != nil {
		logg.Fatal().Err(err).Msg("failed to parse redis url")
	}

	rdb := redis.NewClient(opt)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		logg.Fatal().Err(err).Msg("redis is not reachable")
	}
	rdb.Close()

	// Convert redis.Options to asynq.RedisClientOpt
	redisOpt := asynq.RedisClientOpt{
		Addr:     opt.Addr,
		Username: opt.Username,
		Password: opt.Password,
		DB:       opt.DB,
		// TLSConfig is needed for rediss:// URLs (Redis Cloud, Upstash, etc.)
		TLSConfig: opt.TLSConfig,
	}
	srv := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
				return time.Duration(1<<uint(n)) * time.Second
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				logEvent := logg.Error().
					Err(err).
					Str("task_type", task.Type())

				if task.ResultWriter() != nil {
					logEvent = logEvent.Str("task_id", task.ResultWriter().TaskID())
				}

				logEvent.Msg("task processing failed")
			}),
			ShutdownTimeout: 30 * time.Second,
		},
	)

	mux := asynq.NewServeMux()
	mux.Use(loggingMiddleware(logg))

	mux.HandleFunc(TypeHealthCheck, func(ctx context.Context, t *asynq.Task) error {
		logg.Info().
			Str("task_type", t.Type()).
			Msg("worker is working")
		return nil
	})
	mux.HandleFunc(jobs.TypeImageProcess, workers.HandleImagePP)

	sigctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logg.Info().Msg("worker starting")
		if err := srv.Run(mux); err != nil {
			if !errors.Is(err, asynq.ErrServerClosed) {
				logg.Fatal().Err(err).Msg("worker failed to start")
			}
		}
	}()

	// Wait for interrupt signal
	<-sigctx.Done()
	logg.Info().Msg("shutting down worker...")

	// Create a channel to signal shutdown completion
	shutdownDone := make(chan struct{})
	go func() {
		srv.Shutdown()
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
		logg.Info().Msg("worker stopped cleanly")
	case <-time.After(35 * time.Second):
		logg.Error().Msg("shutdown timed out, forcing exit")
		os.Exit(1)
	}
}

func loggingMiddleware(logg zerolog.Logger) asynq.MiddlewareFunc {
	return func(next asynq.Handler) asynq.Handler {
		return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
			start := time.Now()

			logEvent := logg.Info().
				Str("task_type", task.Type())

			if task.ResultWriter() != nil {
				logEvent = logEvent.Str("task_id", task.ResultWriter().TaskID())
			}

			logEvent.Msg("task started")

			err := next.ProcessTask(ctx, task)
			duration := time.Since(start)

			if err != nil {
				logEvent := logg.Error().
					Err(err).
					Str("task_type", task.Type()).
					Dur("duration", duration)

				if task.ResultWriter() != nil {
					logEvent = logEvent.Str("task_id", task.ResultWriter().TaskID())
				}

				logEvent.Msg("task failed")
			} else {
				logEvent := logg.Info().
					Str("task_type", task.Type()).
					Dur("duration", duration)

				if task.ResultWriter() != nil {
					logEvent = logEvent.Str("task_id", task.ResultWriter().TaskID())
				}

				logEvent.Msg("task completed")
			}
			return err
		})
	}
}
