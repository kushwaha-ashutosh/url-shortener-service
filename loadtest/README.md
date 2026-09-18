# Load testing the redirect path

Two runs, same 30-ish second window, same machine, different client
placement — this matters more than it sounds like it should.

## 1. Through Docker (`redirect.js`, via k6)

```bash
make loadtest
```

k6 runs *inside* a Docker container and reaches the Go server on the
Windows host through `host.docker.internal`. That hop goes through
Docker Desktop's NAT/virtualized networking layer.

Ramping to 200 VUs over 80s:

| Metric | Value |
|---|---|
| Requests | 202,739 |
| Throughput | 2,395 req/s |
| p90 | 70.1ms |
| p95 | 86.7ms |
| p99 | 132.4ms |
| Errors | 0.00% |

## 2. Native, same host (`hey`)

```bash
hey -z 30s -c 100 -disable-redirects http://localhost:8081/<code>
```

No Docker hop — `hey` and the Go server are both native Windows
processes talking over loopback.

| Metric | Value |
|---|---|
| Requests | 141,163 |
| Throughput | 4,703 req/s |
| p90 | 36.3ms |
| p95 | 44.2ms |
| p99 | 65.5ms |
| Errors | 0.00% |

## Why both numbers are worth keeping

The Docker-routed run isn't "wrong" — it's testing something real:
what a client outside the app's network actually experiences, NAT
included. But citing only that number as "the redirect service's
latency" would be misleading, since roughly half of that p95 is
Docker Desktop's networking layer, not the application. Keeping both
numbers, and being explicit about why they differ by ~2x, is the more
defensible thing to say in an interview than picking whichever number
looks better.

In both runs: 0% errors, and the cache-aside path holds up under load
without degrading — the Redis hit path is doing its job.
