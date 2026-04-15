package httpadapter

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"

	"github.com/webvalera96/ai-speech-recognition/internal/config"
	httphandler "github.com/webvalera96/ai-speech-recognition/internal/handlers/http"
)

// ServerParams bundles HTTP server construction for Fx.
type ServerParams struct {
	fx.In
	LC   fx.Lifecycle
	Cfg  *config.Config
	Pool *pgxpool.Pool
}

// NewRouter builds chi router with health endpoints.
func NewRouter(pool *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Get("/health", httphandler.Health)
	r.Get("/ready", httphandler.Ready(pool))
	return r
}

// RegisterHTTPServer starts and stops the HTTP server.
func RegisterHTTPServer(p ServerParams, h http.Handler) {
	srv := &http.Server{
		Addr:         p.Cfg.HTTPAddr,
		Handler:      h,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	p.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					panic(err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			return srv.Shutdown(shutdownCtx)
		},
	})
}
