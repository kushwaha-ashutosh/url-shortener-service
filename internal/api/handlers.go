package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ashutoshk/url-shortener/internal/cache"
	"github.com/ashutoshk/url-shortener/internal/shortener"
	"github.com/ashutoshk/url-shortener/internal/store"
)

const (
	codeLength      = 7
	maxCreateRetries = 5
)

type Handler struct {
	store   *store.Store
	cache   *cache.Cache
	clicks  *ClickBatcher
	baseURL string
}

func NewHandler(s *store.Store, c *cache.Cache, cb *ClickBatcher, baseURL string) *Handler {
	return &Handler{store: s, cache: c, clicks: cb, baseURL: baseURL}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/healthz", h.HealthCheck)
	r.Post("/api/links", h.CreateLink)
	r.Get("/api/links/{code}/stats", h.GetStats)
	r.Get("/{code}", h.Redirect)
	return r
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

type createLinkRequest struct {
	URL string `json:"url"`
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
	for attempt := 0; attempt < maxCreateRetries; attempt++ {
		code, err := shortener.Generate(codeLength)
		if err != nil {
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
		writeError(w, http.StatusInternalServerError, "failed to create link")
		return
	}
	if link == nil {
		writeError(w, http.StatusInternalServerError, "could not allocate a unique code, try again")
		return
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
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
