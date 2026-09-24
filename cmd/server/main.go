package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/kushwaha-ashutosh/url-shortener/internal/api"
	"github.com/kushwaha-ashutosh/url-shortener/internal/cache"
	"github.com/kushwaha-ashutosh/url-shortener/internal/logging"
	"github.com/kushwaha-ashutosh/url-shortener/internal/ratelimit"
	"github.com/kushwaha-ashutosh/url-shortener/internal/store"
)

var log = logging.Base()

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := loadConfig()

	s, err := store.New(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer s.Close()

	c, err := cache.New(cfg.RedisAddr)
	if err != nil {
		log.Error("invalid redis address/URL", "error", err)
		os.Exit(1)
	}
	if err := pingWithRetry(ctx, c, 5, 2*time.Second); err != nil {
		log.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := c.Close(); err != nil {
			log.Warn("failed to close redis client cleanly", "error", err)
		}
	}()

	batcher := api.NewClickBatcher(s, 10_000, 200, 2*time.Second)
	batcherCtx, cancelBatcher := context.WithCancel(context.Background())
	go batcher.Run(batcherCtx)

	limiter := ratelimit.New(c.Client(), cfg.RateLimitRequests, cfg.RateLimitWindow)

	handler := api.NewHandler(s, c, batcher, limiter, cfg.BaseURL, cfg.AllowedOrigin)
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler.Routes(),
	}

	go func() {
		log.Info("listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
	}

	// Stop the batcher after HTTP stops accepting new redirects, so it
	// can drain and flush whatever is left in the buffer.
	cancelBatcher()
	time.Sleep(200 * time.Millisecond)
}

// pingWithRetry retries the startup Redis health check rather than
// failing on the first attempt. This is a different situation from
// the deliberately no-retry, fail-fast policy on the redirect hot
// path (internal/cache, internal/api/handlers.go): retrying a few
// times before the process has even started serving traffic is the
// standard "wait for a dependency to become reachable" pattern, not a
// retry loop hidden inside a live request.
func pingWithRetry(ctx context.Context, c *cache.Cache, attempts int, delay time.Duration) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = c.Ping(ctx); err == nil {
			return nil
		}
		log.Warn("redis ping failed, retrying", "attempt", i+1, "of", attempts, "error", err)
		if i < attempts-1 {
			time.Sleep(delay)
		}
	}
	return err
}

type config struct {
	Port              string
	BaseURL           string
	PostgresDSN       string
	RedisAddr         string
	RateLimitRequests int
	RateLimitWindow   time.Duration
	AllowedOrigin     string
}

func loadConfig() config {
	return config{
		Port:              getEnv("PORT", "8081"),
		BaseURL:           getEnv("BASE_URL", "http://localhost:8081"),
		PostgresDSN:       getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/urlshortener?sslmode=disable"),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		RateLimitRequests: getEnvInt("RATE_LIMIT_REQUESTS", 20),
		RateLimitWindow:   time.Duration(getEnvInt("RATE_LIMIT_WINDOW_SECONDS", 60)) * time.Second,
		AllowedOrigin:     getEnv("ALLOWED_ORIGIN", "http://localhost:5173"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Warn("invalid int env var, using default", "key", key, "value", v, "default", fallback)
		return fallback
	}
	return n
}
