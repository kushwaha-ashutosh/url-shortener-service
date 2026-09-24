package main

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
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
	logRedisDNS(cfg.RedisAddr)
	logRedisTCPDial(cfg.RedisAddr)
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
// retry loop hidden inside a live request. Added after a first
// production deploy to Upstash failed on process startup with a bare
// "EOF" — the process' own supervisor restarting the whole container
// on crash is a much cruder recovery path than retrying in-process.
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

// redisHost extracts just the hostname from REDIS_ADDR, which is
// either a plain "host:port" or a full "redis://"/"rediss://" URL —
// mirrors the scheme-detection in internal/cache.New so the log line
// below is diagnosing the same host that cache.New will actually dial.
func redisHost(addrOrURL string) string {
	if strings.Contains(addrOrURL, "://") {
		if u, err := url.Parse(addrOrURL); err == nil {
			return u.Hostname()
		}
		return ""
	}
	if h, _, err := net.SplitHostPort(addrOrURL); err == nil {
		return h
	}
	return addrOrURL
}

// logRedisDNS resolves the Redis host and logs the result before the
// connection is attempted. Added while diagnosing a production deploy
// where the app could connect to Postgres but got a bare "EOF" trying
// to reach Redis: this narrows whether that's a DNS problem specific
// to the deploy environment (which would show up here as a lookup
// failure) or something failing later, at the TCP/TLS layer instead.
func logRedisDNS(addrOrURL string) {
	host := redisHost(addrOrURL)
	if host == "" {
		log.Warn("could not extract a host from REDIS_ADDR for DNS diagnostics")
		return
	}
	ips, err := net.LookupHost(host)
	if err != nil {
		log.Warn("redis host DNS lookup failed", "host", host, "error", err)
		return
	}
	log.Info("resolved redis host", "host", host, "ips", ips)
}

// redisAddr returns a dialable "host:port" for REDIS_ADDR in either
// its plain or URL form, defaulting to Redis's conventional 6379 when
// a URL doesn't specify one explicitly.
func redisAddr(addrOrURL string) string {
	if strings.Contains(addrOrURL, "://") {
		u, err := url.Parse(addrOrURL)
		if err != nil {
			return ""
		}
		port := u.Port()
		if port == "" {
			port = "6379"
		}
		return net.JoinHostPort(u.Hostname(), port)
	}
	return addrOrURL
}

// logRedisTCPDial attempts a plain TCP connection to the Redis host —
// no TLS, no Redis protocol, just "can a socket be opened at all."
// Added alongside logRedisDNS to isolate where a connection actually
// fails: DNS resolved fine in production, but the TLS-wrapped
// connection still failed with a bare "EOF" in under 100ms — too fast
// to be a timeout, and consistent across two different databases, in
// a way that pointed away from anything Redis- or TLS-specific and
// toward the network path itself. This checks that theory directly:
// if even a bare TCP handshake to port 6379 fails or is refused, the
// problem is the deploy environment's outbound network on that port,
// not this application or its TLS setup.
func logRedisTCPDial(addrOrURL string) {
	addr := redisAddr(addrOrURL)
	if addr == "" {
		log.Warn("could not extract host:port from REDIS_ADDR for TCP dial diagnostics")
		return
	}
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	elapsed := time.Since(start)
	if err != nil {
		log.Warn("raw TCP dial to redis host failed", "addr", addr, "elapsed_ms", elapsed.Milliseconds(), "error", err)
		return
	}
	_ = conn.Close()
	log.Info("raw TCP dial to redis host succeeded", "addr", addr, "elapsed_ms", elapsed.Milliseconds())
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
