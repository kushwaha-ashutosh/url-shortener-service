package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kushwaha-ashutosh/url-shortener/internal/cache"
	"github.com/kushwaha-ashutosh/url-shortener/internal/logging"
	"github.com/kushwaha-ashutosh/url-shortener/internal/ratelimit"
	"github.com/kushwaha-ashutosh/url-shortener/internal/shortener"
	"github.com/kushwaha-ashutosh/url-shortener/internal/store"
)

const (
	codeLength       = 7
	maxCreateRetries = 5
)

var customCodePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{3,32}$`)

// reservedCodes can't be used as custom codes because they'd either
// never be reachable (an explicit route like /healthz always wins over
// the /{code} wildcard) or would be confusing to see as a short link.
var reservedCodes = map[string]bool{
	"api":     true,
	"healthz": true,
}

func validateCustomCode(code string) error {
	if !customCodePattern.MatchString(code) {
		return errors.New("custom_code must be 3-32 characters (letters, numbers, hyphens, underscores)")
	}
	if reservedCodes[strings.ToLower(code)] {
		return errors.New("custom_code is reserved")
	}
	return nil
}

type Handler struct {
	store   *store.Store
	cache   *cache.Cache
	clicks  *ClickBatcher
	limiter *ratelimit.Limiter
	baseURL string
}

func NewHandler(s *store.Store, c *cache.Cache, cb *ClickBatcher, rl *ratelimit.Limiter, baseURL string) *Handler {
	return &Handler{store: s, cache: c, clicks: cb, limiter: rl, baseURL: baseURL}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(RequestID)
	r.Get("/healthz", h.HealthCheck)
	r.With(h.RateLimit).Post("/api/links", h.CreateLink)
	r.Get("/api/links/{code}/stats", h.GetStats)
	r.Get("/{code}", h.Redirect)
	return r
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

type createLinkRequest struct {
	URL        string `json:"url"`
	CustomCode string `json:"custom_code,omitempty"`
}

type createLinkResponse struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
	LongURL  string `json:"long_url"`
}

func (h *Handler) CreateLink(w http.ResponseWriter, r *http.Request) {
	var req createLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	longURL, err := normalizeURL(req.URL)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var link *store.Link
	if req.CustomCode != "" {
		if err := validateCustomCode(req.CustomCode); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		link, err = h.store.CreateLink(r.Context(), req.CustomCode, longURL)
		if err != nil {
			if errors.Is(err, store.ErrCodeTaken) {
				writeError(w, http.StatusConflict, "custom_code is already in use")
				return
			}
			logging.From(r.Context()).Error("failed to create link with custom code", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to create link")
			return
		}
	} else {
		for attempt := 0; attempt < maxCreateRetries; attempt++ {
			code, err := shortener.Generate(codeLength)
			if err != nil {
				logging.From(r.Context()).Error("failed to generate short code", "error", err)
				writeError(w, http.StatusInternalServerError, "failed to generate code")
				return
			}
			link, err = h.store.CreateLink(r.Context(), code, longURL)
			if err == nil {
				break
			}
			if errors.Is(err, store.ErrCodeTaken) {
				continue // collision on a 7-char random code is rare; just retry
			}
			logging.From(r.Context()).Error("failed to create link", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to create link")
			return
		}
		if link == nil {
			writeError(w, http.StatusInternalServerError, "could not allocate a unique code, try again")
			return
		}
	}

	writeJSON(w, http.StatusCreated, createLinkResponse{
		Code:     link.Code,
		ShortURL: h.baseURL + "/" + link.Code,
		LongURL:  link.LongURL,
	})
}

func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	ctx := r.Context()

	longURL, err := h.cache.GetURL(ctx, code)
	if err != nil {
		// Covers both a cache miss and a degraded Redis: Postgres is the
		// source of truth either way, so fall through rather than fail.
		link, err := h.store.GetLink(ctx, code)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "short link not found")
				return
			}
			logging.From(ctx).Error("failed to look up link", "code", code, "error", err)
			writeError(w, http.StatusInternalServerError, "lookup failed")
			return
		}
		longURL = link.LongURL
		_ = h.cache.SetURL(ctx, code, longURL) // best-effort fill
	}

	h.clicks.Enqueue(store.ClickEvent{
		Code:      code,
		Timestamp: time.Now().UTC(),
		Referrer:  r.Referer(),
		UserAgent: r.UserAgent(),
	})

	http.Redirect(w, r, longURL, http.StatusFound)
}

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	stats, err := h.store.GetStats(r.Context(), code)
	if err != nil {
		logging.From(r.Context()).Error("failed to load stats", "code", code, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load stats")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// normalizeURL rejects empty input and non-http(s) schemes so the
// service can't be used as an open redirector to javascript:/data: URIs
// or to local/file URLs.
func normalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", errors.New("url is not valid")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("url must use http or https")
	}
	if u.Host == "" {
		return "", errors.New("url must include a host")
	}
	return u.String(), nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Nothing actionable to do with an encode error here: headers and
	// status are already written, so the response is already committed.
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
