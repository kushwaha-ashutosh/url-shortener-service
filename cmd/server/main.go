package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ashutoshk/url-shortener/internal/api"
	"github.com/ashutoshk/url-shortener/internal/cache"
	"github.com/ashutoshk/url-shortener/internal/ratelimit"
	"github.com/ashutoshk/url-shortener/internal/store"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := loadConfig()

	s, err := store.New(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer s.Close()

	c := cache.New(cfg.RedisAddr)
	if err := c.Ping(ctx); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	defer c.Close()

	batcher := api.NewClickBatcher(s, 10_000, 200, 2*time.Second)
	batcherCtx, cancelBatcher := context.WithCancel(context.Background())
	go batcher.Run(batcherCtx)

	limiter := ratelimit.New(c.Client(), cfg.RateLimitRequests, cfg.RateLimitWindow)

	handler := api.NewHandler(s, c, batcher, limiter, cfg.BaseURL)
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler.Routes(),
	}

	go func() {
		log.Printf("listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}

	// Stop the batcher after HTTP stops accepting new redirects, so it
	// can drain and flush whatever is left in the buffer.
	cancelBatcher()
	time.Sleep(200 * time.Millisecond)
}

type config struct {
	Port              string
	BaseURL           string
	PostgresDSN       string
	RedisAddr         string
	RateLimitRequests int
	RateLimitWindow   time.Duration
}

func loadConfig() config {
	return config{
		Port:              getEnv("PORT", "8081"),
		BaseURL:           getEnv("BASE_URL", "http://localhost:8081"),
		PostgresDSN:       getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/urlshortener?sslmode=disable"),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		RateLimitRequests: getEnvInt("RATE_LIMIT_REQUESTS", 20),
		RateLimitWindow:   time.Duration(getEnvInt("RATE_LIMIT_WINDOW_SECONDS", 60)) * time.Second,
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
		log.Printf("invalid int for %s=%q, using default %d", key, v, fallback)
		return fallback
	}
	return n
}
