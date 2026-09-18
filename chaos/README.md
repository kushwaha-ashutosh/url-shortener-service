# Chaos test: kill Redis mid-load

The README had claimed since the first commit that a Redis outage
falls through to Postgres cleanly. That claim had never actually been
tested — this is what happened when it was.

## Procedure

```bash
# 1. Create a link and warm its cache entry
CODE=$(curl -s -X POST localhost:8081/api/links -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com/chaos-test"}' | grep -o '"code":"[^"]*"' | cut -d'"' -f4)
curl -s -o /dev/null "localhost:8081/$CODE"

# 2. Baseline load (Redis healthy)
hey -z 15s -c 50 -disable-redirects "http://localhost:8081/$CODE"

# 3. Kill Redis mid-load
docker compose stop redis
hey -z 15s -c 50 -disable-redirects "http://localhost:8081/$CODE"

# 4. Recovery
docker compose start redis
hey -z 15s -c 50 -disable-redirects "http://localhost:8081/$CODE"
```

## Round 1: the claim didn't hold

A single request during the outage hung for **30+ seconds** before
returning — `go-redis`'s default timeouts (5s dial, 3s read, 3
retries with backoff) let one request eat 20-30s trying to reach a
dead Redis before the cache-aside fallback ever got a chance to run.
Under concurrent load it was worse: 50 concurrent clients produced
**2.4 req/s**, p50 of **15.3s**, and 21 outright client-side timeouts.
The code's fallback logic was correct; the client configuration made
it unreachable in practice.

## Fix

Two changes, both in this repo's history:

1. **[internal/cache/redis.go](../internal/cache/redis.go)** — the
   Redis client now sets `DialTimeout`/`ReadTimeout`/`WriteTimeout` to
   250ms and disables retries (`MaxRetries: -1`), instead of relying
   on go-redis's defaults.
2. **[internal/api/handlers.go](../internal/api/handlers.go)** — each
   Redis call on the redirect path is additionally wrapped in an
   explicit `context.WithTimeout(ctx, 300ms)`. Client-level timeouts
   alone didn't cap worst-case latency under concurrency — most likely
   pool-level connection acquisition under contention isn't fully
   bounded by `DialTimeout` alone. A context deadline enforced at the
   call site is bounded by Go's own cancellation regardless of what
   the client does internally to acquire a connection.

## Round 2: results after the fix

| Phase | Redis state | Throughput | p50 | p99 | Success rate |
|---|---|---|---|---|---|
| Baseline | up, cache warm | 5,017 req/s | 8.8ms | 29.1ms | 100% |
| **Outage** | **down** | **3,510 req/s** | **11.9ms** | **37.9ms** | **100% (52,682/52,682)** |
| Recovery | back up | 4,590 req/s | 9.6ms | 32.5ms | 100% |

Worst single request during the outage: 930ms (bounded, as designed —
never the 20-30s seen before the fix). Every request across all three
phases returned a correct `302`; recovery required no restart or
manual intervention, since `go-redis` reconnects on its own once the
server is reachable again.

## Why this is worth keeping in the portfolio story

Not because the fix is clever — it's two timeout values — but because
finding it required actually running the failure, not reasoning about
it. The original design (cache-aside with a Postgres fallback) was
correct on paper and still failed in practice because of a default
buried three layers down in a client library. That's a more honest
and more common version of "resilient system" than a README claim
that was never load-tested against.
