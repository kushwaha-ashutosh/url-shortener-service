# url-shortener

A URL shortener built like it might actually need to survive traffic:
redirects are served from a Redis cache in front of Postgres, and click
analytics are logged asynchronously so writing an analytics row never
adds latency to a redirect.

## Why this isn't just a toy CRUD app

- **Cache-aside redirects.** `GET /:code` checks Redis first. On a miss
  it reads Postgres, then fills the cache — no cache stampede handling
  needed at this scale, but the pattern is the same one that scales.
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

## Running locally

```bash
cp .env.example .env
make up          # starts Postgres + Redis via docker compose
make run         # starts the API on :8080
```

Create a short link:

```bash
curl -X POST localhost:8080/api/links \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://example.com/some/very/long/path"}'
```

Follow it (and generate a click event):

```bash
curl -iL localhost:8080/<code>
```

Check stats:

```bash
curl localhost:8080/api/links/<code>/stats
```

## Testing

```bash
make test
```

Covers: short-code generation (length, character set, collision rate
over repeated draws) and URL validation (rejects non-http(s) schemes,
empty input, malformed URLs).

## Roadmap

- [ ] Custom short codes (currently random-only)
- [ ] Aggregated stats by day/referrer/user-agent, not just a total count
- [ ] Small React dashboard for link + click stats
- [ ] k6 load test script for the redirect hot path
- [ ] Rate limiting on `POST /api/links` to prevent abuse
