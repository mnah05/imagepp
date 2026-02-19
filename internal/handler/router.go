package handler

import (
	"imagepp/internal/config"
	"imagepp/internal/db"
	custommiddleware "imagepp/internal/middleware"
	"imagepp/internal/scheduler"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

func NewRouter(log zerolog.Logger, cfg *config.Config, dbpool *pgxpool.Pool, redis *redis.Client, scheduler *scheduler.Client) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Recoverer)
	r.Use(custommiddleware.RequestLogger(log))

	// Initialize queries (sqlc)
	queries := db.New(dbpool)

	// Health check
	health := NewHealthHandler(dbpool, redis, scheduler)
	r.Get("/health", health.Check)

	// Image processing routes
	imageHandler := NewImageHandler(log, queries, scheduler)
	r.Route("/api", func(r chi.Router) {
		r.Post("/image", imageHandler.ProcessImage)
		//r.Get("/image-status", imageHandler.GetStatus)
		//r.Get("/user/{email}/images", imageHandler.GetUserImages)
	})

	return r
}
