# url-shortener

[![CI](https://github.com/kushwaha-ashutosh/url-shortener-service/actions/workflows/ci.yml/badge.svg)](https://github.com/kushwaha-ashutosh/url-shortener-service/actions/workflows/ci.yml)

A URL shortener built like it might actually need to survive traffic:
redirects are served from a Redis cache in front of Postgres, and click
analytics are logged asynchronously so writing an analytics row never
adds latency to a redirect.

## Why this isn't just a toy CRUD app

- **Cache-aside redirects that actually degrade gracefully.** `GET
  /:code` checks Redis first, falling back to Postgres on a miss. This
  claim used to be untested — [chaos-testing it](chaos/README.md) by
  actually killing Redis under load found the fallback took 20-30s per
  request in practice (client library defaults, not the fallback logic
  itself), fixed to a 300ms-bounded worst case, verified with
  concurrent load before/during/after the outage.
- **Redirect latency is decoupled from analytics writes.** Every click
  goes into a buffered channel ([internal/api/batcher.go](internal/api/batcher.go))
  and is flushed to Postgres in batches via `COPY`, not one `INSERT` per
  click. The HTTP response never waits on that write.
- **Backpressure has an explicit policy.** If the click buffer fills up
  (Postgres falling behind), new click events are dropped rather than
  blocking redirects. Losing a click event is fine; slowing down every
  redirect in the system is not.
- **Open-redirect protection.** Submitted URLs are parsed and restricted
  to `http`/`https` with a host, so the service can't be used to shorten
  `javascript:`/`data:`/`file:` URIs.
- **Distributed rate limiting on link creation.** `POST /api/links` is
  gated by a sliding-window-log limiter ([internal/ratelimit](internal/ratelimit))
  backed by Redis, not an in-process counter — the limit holds even
  with multiple API instances behind a load balancer, since they all
  check the same Redis key. The count-then-add sequence runs as a
  single Lua script so concurrent requests from the same client can't
  race past the limit. On a Redis error the limiter fails open (logs
  and lets the request through) — a degraded rate limiter shouldn't be
  able to take down link creation entirely.
- **Structured logs correlated by request ID.** Every request is
  tagged with an ID (reused from an inbound `X-Request-Id` header if
  present, so it survives a hop behind a gateway), echoed back in the
  response, and attached to every log line produced while handling
  that request ([internal/logging](internal/logging),
  [internal/api/middleware.go](internal/api/middleware.go)) — so a
  redirect's log entry and any error it hits can be found by grepping
  for one ID, not reconstructed by guessing timestamps.
- **Aggregated stats, not just a counter.** `GET /api/links/:code/stats`
  returns a 30-day daily series, top referrers (empty referrer grouped
  as `direct`), and a browser-family breakdown — all computed as
  `GROUP BY` aggregates in Postgres ([internal/store/postgres.go](internal/store/postgres.go))
  rather than pulling every click row back to Go to count in memory.
  Also returns `404` for a code that was never created, instead of a
  misleading `200` with all-zero counts (the previous behavior).
- **The app itself is containerized**, not just its dependencies. A
  multi-stage [Dockerfile](Dockerfile) builds a static binary and ships
  it in a `alpine` runtime image (~37MB, no Go toolchain along for the
  ride) running as a non-root user — `docker compose up -d --build` is
  a genuine one-command demo, not "clone this and configure five things
  first."
- **QR codes generated server-side.** `GET /:code/qr` returns a PNG QR
  code encoding the short URL, generated on the backend rather than in
  the browser — the result is a stable, cacheable image URL that works
  the same whether it's loaded in the dashboard or embedded directly
  in a flyer, not something that only exists while a client-side
  script is running.

## Architecture

```
                 ┌──────────┐   miss   ┌────────────┐
  GET /:code ───▶│  Redis   │─────────▶│  Postgres  │
                 │ (cache)  │◀─────────│  (source   │
                 └────┬─────┘   fill   │  of truth) │
                      │ hit            └─────┬──────┘
                      ▼                      ▲ batched COPY
                 302 redirect                │ every 2s / 200 events
                      │                      │
                      ▼                      │
              ┌───────────────┐              │
              │ click buffer  │──────────────┘
              │ (chan, async) │
              └───────────────┘
```

## Running it

One command, nothing installed locally except Docker — builds the
app's own image (multi-stage, ~37MB final) alongside Postgres and
Redis, and starts all three:

```bash
make up          # docker compose up -d --build
```

```bash
curl localhost:8081/healthz   # ok
```

On Windows PowerShell without `make` installed: `docker compose up -d --build`.

Postgres is mapped to host port **5433** (not 5432) and the API listens
on **8081** (not 8080) to avoid colliding with other services that may
already be running locally.

### Local dev loop

Rebuilding the app's Docker image on every code change is slow for
active development. `make up-deps` starts only Postgres and Redis,
and `make run` runs the app natively against them with a normal
`go run` edit/rebuild cycle:

```bash
cp .env.example .env
make up-deps     # Postgres (:5433) + Redis (:6379) only
make run         # go run ./cmd/server, on :8081
```

Create a short link:

```bash
curl -X POST localhost:8081/api/links \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://example.com/some/very/long/path"}'
```

Or with a custom code (3-32 chars, letters/numbers/hyphens/underscores,
returns `409` if taken):

```bash
curl -X POST localhost:8081/api/links \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://example.com/some/path", "custom_code": "my-link"}'
```

Follow it (and generate a click event):

```bash
curl -iL localhost:8081/<code>
```

Get its QR code:

```bash
curl localhost:8081/<code>/qr -o qr.png
```

Check stats (total, a 30-day daily series, top referrers, and a
browser-family breakdown):

```bash
curl localhost:8081/api/links/<code>/stats
```

```json
{
  "code": "faNiQGa",
  "total_clicks": 5,
  "clicks_by_day": [{"day": "2026-09-18", "clicks": 5}],
  "top_referrers": [
    {"name": "direct", "clicks": 2},
    {"name": "https://twitter.com/foo", "clicks": 2}
  ],
  "top_browsers": [{"name": "Chrome", "clicks": 2}, {"name": "Firefox", "clicks": 1}]
}
```

## Testing

```bash
make test
```

Covers: short-code generation (length, character set, collision rate
over repeated draws), URL and custom-code validation, and the rate
limiter (admits up to the limit, denies over it, keys are independent
per client, and — via 100 concurrent goroutines racing a limit of
20 — that the atomic Lua script never admits more than the limit
under concurrent load).

## Dashboard

A small React + TypeScript dashboard lives in [dashboard/](dashboard) —
create links, watch their stats (30-day chart, referrers, browsers)
update live via polling. See [dashboard/README.md](dashboard/README.md)
for setup; the backend's `ALLOWED_ORIGIN` env var needs to match
wherever it's served from (defaults to Vite's `:5173`).

## Load testing

See [loadtest/README.md](loadtest/README.md) — the redirect path holds
0% errors at ~2,400 req/s through a Docker network hop (p99 132ms) and
~4,700 req/s native same-host (p99 65ms), with the gap between the two
attributed and explained rather than glossed over.

## Roadmap

- [x] k6 load test script for the redirect hot path
- [x] Rate limiting on `POST /api/links` to prevent abuse
- [x] Custom short codes
- [x] Structured logging + request ID middleware
- [x] Aggregated stats by day/referrer/user-agent, not just a total count
- [x] Small React dashboard for link + click stats
- [x] Dockerize the app itself
- [x] CI pipeline (build/vet/test/lint on every push)
- [ ] Integration-test the store layer against a real Postgres (e.g.
      testcontainers-go) — the stats aggregation SQL is currently only
      verified manually, since `internal/store` has no automated tests yet
- [x] Chaos test: kill Redis mid-load (found and fixed a real 20-30s hang — see [chaos/README.md](chaos/README.md))
- [ ] Deploy publicly (Fly.io/Railway) so the demo link is real
