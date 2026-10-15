// k6 query-latency benchmark for the ChainPulse read API.
//
// Usage:
//   BASE_URL=http://127.0.0.1:8080 DURATION=60s k6 run scripts/bench/query_events.js
//
// Measures GET /events and GET /events/stats latency under a constant arrival
// rate. Results are printed by k6 (http_req_duration p50/p99) and summarised in
// the README's benchmark table.
import http from 'k6/http';
import { check } from 'k6';

const BASE = __ENV.BASE_URL || 'http://127.0.0.1:8080';
const DURATION = __ENV.DURATION || '60s';
const RATE = Number(__ENV.RATE || 100);

export const options = {
  scenarios: {
    query_events: {
      executor: 'constant-arrival-rate',
      exec: 'queryEvents',
      rate: RATE,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: 20,
      maxVUs: 200,
    },
    query_stats: {
      executor: 'constant-arrival-rate',
      exec: 'queryStats',
      rate: Math.max(1, Math.floor(RATE / 10)),
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: 5,
      maxVUs: 50,
    },
  },
  thresholds: {
    // Recorded for the README table; not a pass/fail gate here.
    http_req_failed: ['rate<0.01'],
  },
  summaryTrendStats: ['avg', 'min', 'med', 'p(50)', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

export function queryEvents() {
  const res = http.get(`${BASE}/events`);
  check(res, { 'events 200': (r) => r.status === 200 });
}

export function queryStats() {
  const res = http.get(`${BASE}/events/stats`);
  check(res, { 'stats 200': (r) => r.status === 200 });
}
