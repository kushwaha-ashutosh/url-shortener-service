# url-shortener

[![CI](https://github.com/kushwaha-ashutosh/url-shortener-service/actions/workflows/ci.yml/badge.svg)](https://github.com/kushwaha-ashutosh/url-shortener-service/actions/workflows/ci.yml)

**Live:** [url-shortener-lq0d.onrender.com](https://url-shortener-lq0d.onrender.com) —
try it: `POST /api/links` with `{"url": "https://example.com"}`, then
visit the short link it returns. (Free-tier hosting: the first request
after a period of inactivity may take a few seconds to wake up.)

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

## Deploying

Free-tier deployment across three providers rather than one all-in-one
platform, since none of the always-free single-platform options are
reliably free anymore: [Render](https://render.com) for the app,
[Neon](https://neon.tech) for Postgres, [Upstash](https://upstash.com)
for Redis. Account creation on each is something only you can do —
here's the exact path.

**1. Postgres (Neon)**
1. Create a free account at neon.tech, create a project.
2. Copy the connection string it gives you (looks like
   `postgres://user:pass@ep-xxx.neon.tech/dbname?sslmode=require`) —
   this is your `DATABASE_URL`.
3. Open Neon's SQL Editor (in their dashboard) and paste in the
   contents of [migrations/0001_init.sql](migrations/0001_init.sql),
   then run it. This is the one manual migration step — there's no
   `docker-entrypoint-initdb.d` equivalent on a managed Postgres.

**2. Redis (Upstash)**
1. Create a free account at upstash.com, create a Redis database.
2. Copy the `rediss://` connection string from its "Connect" tab
   (note the double-s — Upstash requires TLS). This is your
   `REDIS_ADDR`; the app accepts either a plain `host:port` (local
   dev) or a full `redis://`/`rediss://` URL (see
   [internal/cache/redis.go](internal/cache/redis.go)).
   **Copy it carefully** — see the postmortem below for what happens
   if you don't.

**3. The app (Render)**
1. Create a free account at render.com and connect your GitHub account.
2. New → Blueprint → select this repo. Render reads
   [render.yaml](render.yaml) and creates a free Docker web service
   for the API and a free static site for the dashboard automatically.
3. On the `url-shortener` web service's Environment tab, set the
   secrets `render.yaml` left blank: `DATABASE_URL` (from Neon),
   `REDIS_ADDR` (from Upstash — the `rediss://` URL).
4. Deploy. Once it's live, Render assigns a URL like
   `https://url-shortener-xxxx.onrender.com` — set that as `BASE_URL`
   on the same service and redeploy, so short links the app generates
   point at itself, not `localhost`.
5. On the `url-shortener-dashboard` static site's Environment tab, set
   `VITE_API_BASE_URL` to that same API URL, then redeploy the
   dashboard too.

**Note on the free tier:** the Render free plan sleeps the *web
service* (the API) after inactivity; the first request after a while
wakes it back up with a several-second delay. The static site (the
dashboard) doesn't sleep. Fine for a portfolio demo, not for anything
latency-sensitive — see [loadtest/README.md](loadtest/README.md) for
what this service's actual latency looks like when it's warm.

### Postmortem: the first deploy wasn't actually broken

The first real deploy failed on startup with a bare `"EOF"` connecting
to Redis — repeatedly, identically, across three separately-created
Upstash databases. Debugged it live, layer by layer, adding one
targeted diagnostic at a time straight into the running deploy rather
than guessing: DNS resolution (fine), a raw TCP dial (fine, 1ms), a
raw TLS handshake bypassing the Redis client entirely (fine — TLS 1.3,
a real certificate, a genuine cross-cloud handshake), and finally a
hand-encoded `AUTH` + `PING` exchange over that TLS connection using
the actual RESP protocol.

That last one returned a completely valid, correctly-formed Redis
response: `-WRONGPASS invalid username-password pair`. The entire
investigation — timeout tuning, RESP2/RESP3 protocol forcing, retry
logic — chased what looked like a network or protocol bug, when the
real issue was a wrong password in an environment variable the whole
time. go-redis just never surfaced the actual `WRONGPASS` reason,
reporting the connection closing after the failed auth as a bare
`EOF` instead.

Two of those detours turned out to be worth keeping anyway (see their
code comments for why): the retry-on-startup logic in
[cmd/server/main.go](cmd/server/main.go), and the client timeout
tuning in [internal/cache/redis.go](internal/cache/redis.go), which
were both genuine gaps independent of what actually caused this
particular failure. The diagnostic code itself — the DNS/TCP/TLS/
command-exchange probes — was removed once it had done its job; running
a four-layer network probe on every boot isn't something a working
service should carry forward.

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
- [x] Deploy publicly (Render/Neon/Upstash) — [live](https://url-shortener-lq0d.onrender.com), see "Deploying" above and its postmortem
