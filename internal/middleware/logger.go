package middleware

import (
	"net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"

	"imagepp/pkg/logger"
)

func RequestLogger(base zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			start := time.Now()

			reqID := chimiddleware.GetReqID(r.Context())

			reqLogger := base.With().
				Str("request_id", reqID).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Logger()

			ctx := logger.WithContext(r.Context(), reqLogger)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)

			reqLogger.Info().
				Dur("duration", time.Since(start)).
				Msg("request completed")
		})
	}
}
