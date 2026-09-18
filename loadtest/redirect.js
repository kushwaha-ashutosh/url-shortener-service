// Load test for the GET /:code redirect hot path.
//
// setup() creates a pool of real short links first (unmeasured), then
// every VU picks a random code from that pool for each request — this
// exercises the Redis cache-aside path the way real traffic would,
// hitting a fixed set of "hot" links repeatedly rather than a single
// code (unrealistically cache-friendly) or a fresh code every time
// (unrealistically cache-hostile).
import http from 'k6/http';
import { check } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8081';
const LINK_COUNT = parseInt(__ENV.LINK_COUNT || '50', 10);

export const options = {
  scenarios: {
    redirect_load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '20s', target: 50 },
        { duration: '40s', target: 200 },
        { duration: '20s', target: 0 },
      ],
    },
  },
  thresholds: {
    // Measured from a k6 container hitting the host app via Docker
    // Desktop's host.docker.internal bridge on Windows, so these
    // budgets include that NAT hop's overhead, not just the app's own
    // redirect latency. See loadtest/README.md for a same-host baseline.
    'http_req_duration{expected_response:true}': ['p(95)<100', 'p(99)<150'],
    http_req_failed: ['rate<0.01'],
  },
};

export function setup() {
  const codes = [];
  for (let i = 0; i < LINK_COUNT; i++) {
    const res = http.post(
      `${BASE_URL}/api/links`,
      JSON.stringify({ url: `https://example.com/load-test/${i}` }),
      { headers: { 'Content-Type': 'application/json' } }
    );
    if (res.status !== 201) {
      throw new Error(`setup: failed to create link ${i}: ${res.status} ${res.body}`);
    }
    codes.push(JSON.parse(res.body).code);
  }
  return { codes };
}

export default function (data) {
  const code = data.codes[Math.floor(Math.random() * data.codes.length)];
  const res = http.get(`${BASE_URL}/${code}`, { redirects: 0 });
  check(res, { 'status is 302': (r) => r.status === 302 });
}
