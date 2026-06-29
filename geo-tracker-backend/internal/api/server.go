package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/adoreme/geo-tracker/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

func StartServer(ctx context.Context, cfg config.ServeConfig, database *sqlx.DB, logger *zap.Logger) error {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.Info("api access",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote", r.RemoteAddr),
				zap.Duration("duration", time.Since(start)),
			)
		})
	})
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	h := NewHandlers(database)

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", h.GetHealth)
		r.Get("/prompts", h.GetPrompts)
		r.Get("/prompts/{id}/results", h.GetPromptResults)
		r.Get("/runs", h.GetRuns)
		r.Get("/runs/{id}/results", h.GetRunResults)
		r.Get("/brands", h.GetBrands)
		r.Get("/brands/{brand}/summary", h.GetBrandSummary)
		r.Get("/brands/{brand}/trend", h.GetBrandTrend)
		r.Get("/brands/{brand}/stability", h.GetBrandStability)
		r.Get("/brands/{brand}/citation-gap", h.GetCitationGap)
		r.Get("/explain/{run_id}", h.GetExplain)
		r.Get("/compare", h.GetCompare)
		r.Get("/competitors", h.GetCompetitors)
		r.Get("/recommendations", h.GetRecommendations)
		r.Post("/recommendations/{id}/implement", h.PostImplementRecommendation)
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler: r,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	logger.Info("starting api server", zap.String("addr", srv.Addr))
	return srv.ListenAndServe()
}
