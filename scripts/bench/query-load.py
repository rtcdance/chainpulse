#!/usr/bin/env python3
"""Bounded, constant-arrival query measurement; not the complete E14 gate.

Use a NEW --out directory for every run. Output contains every planned request,
including generator overload and failed HTTP calls. No third-party packages.
"""
import argparse
from collections import Counter
from concurrent.futures import ThreadPoolExecutor
import hashlib
import datetime
import json
import math
from pathlib import Path
import platform
import queue
import random
import subprocess
import sys
import threading
import time
import urllib.error
import urllib.parse
import urllib.request


def percentile(values, percent):
    if not values:
        return None
    values = sorted(values)
    return values[max(0, math.ceil(len(values) * percent / 100) - 1)]


def workload(base, chains, contract, count, seed, bounds=None):
    rng = random.Random(seed)
    plan = []
    for index in range(count):
        chain = rng.choice(chains)
        bucket = rng.random()
        kind = ('offset' if bucket < .30 else 'cursor' if bucket < .55
                else 'stats' if bucket < .70 else 'contract' if bucket < .90 else 'range')
        params = {'chain_id': chain, 'limit': 25 if kind == 'cursor' else 50}
        if kind == 'offset':
            params['offset'] = rng.randrange(500) * 50
        if kind == 'contract':
            params['contract'] = contract
        if kind == 'range':
            low, high = (bounds or {}).get(chain, (1, 1000))
            lower = rng.randint(low, high)
            params.update(from_block=lower, to_block=min(high, lower + 100))
        path = '/events/stats' if kind == 'stats' else '/events?' + urllib.parse.urlencode(params)
        plan.append(dict(params, index=index, kind=kind, chain=chain, url=base.rstrip('/') + path))
    return plan


def shape_error(kind, body):
    try:
        doc = json.loads(body)
    except (ValueError, UnicodeError):
        return 'invalid_json'
    if not isinstance(doc, dict):
        return 'expected_object'
    if kind == 'stats':
        return None if type(doc.get('total')) is int and doc['total'] >= 0 else 'invalid_stats_total'
    if not isinstance(doc.get('data'), list) or not isinstance(doc.get('pagination'), dict):
        return 'invalid_event_page'
    return None


def request(url, timeout):
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    try:
        with opener.open(url, timeout=timeout) as response:
            return response.status, response.read(4 * 1024 * 1024 + 1)
    except urllib.error.HTTPError as error:
        # HTTPError is also URLError: preserve its status before generic errors.
        with error:
            return error.code, error.read(4096)


def measure(plan, rate, workers, capacity, timeout, emit, fetch=request, validator=None):
    inbox = queue.Queue(maxsize=capacity)
    lock = threading.Lock()
    cursors = {}
    records = []
    start = time.monotonic()
    finished = threading.Event()
    aborted = threading.Event()

    def record(row):
        with lock:
            records.append(row)
            emit(row)

    def worker():
        while True:
            if aborted.is_set():
                return
            try:
                item = inbox.get(timeout=.1)
            except queue.Empty:
                if finished.is_set():
                    return
                continue
            try:
                spec, target = item
                picked = time.monotonic()
                url = spec['url']
                cursor_used = False
                cursor_anchor = None
                if spec['kind'] == 'cursor':
                    with lock:
                        cursor, cursor_anchor = cursors.get(spec['chain'], (None, None))
                    if cursor:
                        url += '&cursor=' + urllib.parse.quote(cursor, safe='')
                        cursor_used = True
                row = dict(spec, url=url, cursor_anchor=cursor_anchor, scheduled_s=target - start, started_s=picked - start,
                           queue_wait_ms=max(0, picked - target) * 1000, cursor_used=cursor_used,
                           status=None, error=None)
                try:
                    status, body = fetch(url, timeout)
                    completed = time.monotonic()
                    row['status'] = status
                    row['error'] = ('http_error' if not 200 <= status < 300
                                    else shape_error(spec['kind'], body))
                    if not row['error'] and validator is not None:
                        row['error'] = validator(spec, body, cursor_anchor)
                    if not row['error'] and spec['kind'] in ('offset', 'cursor'):
                        cursor = json.loads(body)['pagination'].get('nextCursor')
                        if isinstance(cursor, str) and len(cursor) <= 8192:
                            with lock:
                                data = json.loads(body)['data']
                                if data and isinstance(data[-1].get('eventId'), str):
                                    cursors[spec['chain']] = (cursor, data[-1]['eventId'])
                except Exception as error:
                    # A worker must emit a failure and keep draining even for
                    # an unexpected transport/parser error; otherwise shutdown
                    # could deadlock behind a full queue.
                    row['error'] = type(error).__name__
                    reason = getattr(error, 'reason', error)
                    row['error_reason_type'] = type(reason).__name__
                    errno = getattr(reason, 'errno', None)
                    row['error_errno'] = errno if type(errno) is int else None
                    completed = time.monotonic()
                row['latency_ms'] = max(0, completed - target) * 1000
                row['validation_ms'] = max(0, time.monotonic() - completed) * 1000
                record(row)
            finally:
                inbox.task_done()

    with ThreadPoolExecutor(max_workers=workers) as pool:
        futures = [pool.submit(worker) for _ in range(workers)]
        try:
            for spec in plan:
                for future in futures:
                    if future.done():
                        future.result()  # Fail promptly if an evidence writer failed.
                target = start + spec['index'] / rate
                time.sleep(max(0, target - time.monotonic()))
                try:
                    inbox.put_nowait((spec, target))
                except queue.Full:
                    record(dict(spec, scheduled_s=target - start, started_s=None, status=None,
                                error='generator_overload', cursor_used=False, queue_wait_ms=None,
                                latency_ms=max(0, time.monotonic() - target) * 1000))
            finished.set()
            for future in futures:
                future.result()
        except BaseException:
            # Do not block trying to enqueue shutdown sentinels into a full
            # inbox when the writer failed or the operator interrupted.
            aborted.set()
            finished.set()
            raise
    return records, time.monotonic() - start


def summarize(records, planned, duration, elapsed):
    latencies = [row['latency_ms'] for row in records]
    errors = sum(row['error'] is not None for row in records)
    overload = sum(row['error'] == 'generator_overload' for row in records)
    correctness = sum(row['error'] is not None and 200 <= (row.get('status') or 0) < 300 for row in records)
    complete = len(records) == planned and {row['index'] for row in records} == set(range(planned))
    p95, p99 = percentile(latencies, 95), percentile(latencies, 99)
    verdicts = {'complete_samples': complete, 'generator_capacity': overload == 0,
                'error_rate': planned > 0 and errors / planned <= .001,
                'p95': p95 is not None and p95 <= 100,
                'p99': p99 is not None and p99 <= 250, 'response_correctness': correctness == 0}
    return {'planned': planned, 'recorded': len(records), 'errors': errors, 'overload': overload, 'correctness_errors': correctness,
            'status_counts': dict(Counter(str(row['status']) for row in records)),
            'p95_ms': p95, 'p99_ms': p99, 'arrival_window_s': duration, 'elapsed_s': elapsed,
            'wire_started': sum(row.get('started_s') is not None for row in records),
            'wire_started_within_window': sum(row.get('started_s') is not None and row['started_s'] < duration for row in records),
            'cursor_used': sum(row['cursor_used'] for row in records), 'verdicts': verdicts,
            'performance_passed': all(verdicts.values()), 'e14_verified': False,
            'validation_scope': 'HTTP and response shape only; separate canonical oracle required'}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--base-url', required=True)
    parser.add_argument('--chains', default='31337,31338')
    parser.add_argument('--contract', required=True)
    parser.add_argument('--out', type=Path, required=True)
    parser.add_argument('--duration', type=float, default=600)
    parser.add_argument('--rate', type=float, default=100)
    parser.add_argument('--workers', type=int, default=32)
    parser.add_argument('--queue-capacity', type=int, default=100)
    parser.add_argument('--timeout', type=float, default=10)
    parser.add_argument('--seed', type=int, default=20261016)
    parser.add_argument('--oracle', type=Path, help='frozen canonical PG snapshot; separate RPC truth still required')
    parser.add_argument('--cache-state', choices=['cold', 'hot', 'unspecified'], default='unspecified')
    args = parser.parse_args()
    try:
        endpoint = urllib.parse.urlsplit(args.base_url)
        valid_url = (endpoint.scheme in ('http', 'https') and endpoint.hostname and endpoint.port != 0
                     and endpoint.username is None and endpoint.password is None and not endpoint.query and not endpoint.fragment)
    except ValueError:
        valid_url = False
    if not valid_url:
        parser.error('base-url must be HTTP(S), without credentials, query or fragment')
    if not all(math.isfinite(n) and n > 0 for n in (args.duration, args.rate, args.timeout)):
        parser.error('duration, rate and timeout must be finite and positive')
    count = int(args.duration * args.rate)
    if not 1 <= count <= 1_000_000 or not 1 <= args.workers <= 256 or not 1 <= args.queue_capacity <= 10000:
        parser.error('require 1..1M requests, 1..256 workers and 1..10000 queue capacity')
    chains = args.chains.split(',')
    if any(not chain.strip() for chain in chains):
        parser.error('chains must be nonempty')
    oracle = None
    oracle_bytes = None
    if args.oracle:
        from query_oracle import QueryOracle
        oracle_bytes = args.oracle.read_bytes()
        oracle = QueryOracle(json.loads(oracle_bytes), chains)
    # mkdir is exclusive: retries must never overwrite or append prior evidence.
    args.out.mkdir(parents=True, exist_ok=False)
    root = Path(__file__).resolve().parents[2]
    manifest = dict(vars(args), out=str(args.out), oracle=str(args.oracle) if args.oracle else None, platform=platform.platform(),
                    source_sha=subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=root, text=True).strip(),
                    source_dirty=bool(subprocess.check_output(['git', 'status', '--porcelain'], cwd=root)),
                    runner_sha256=hashlib.sha256(Path(__file__).read_bytes()).hexdigest())
    manifest['started_at_utc'] = datetime.datetime.now(datetime.timezone.utc).isoformat()
    # Bind dirty source contents as well as HEAD; build/ and node_modules are
    # excluded by Git's normal ignore rules. Hash data, never print contents.
    paths = subprocess.check_output(['git', 'ls-files', '--cached', '--others', '--exclude-standard', '-z'], cwd=root).split(b'\0')
    source_digest = hashlib.sha256()
    for name in sorted(set(paths) - {b''}):
        path = root / name.decode()
        source_digest.update(name + b'\0')
        source_digest.update(hashlib.sha256(path.read_bytes()).digest() if path.is_file() else b'missing-or-gitlink')
    manifest['source_contents_sha256'] = source_digest.hexdigest()
    if args.oracle:
        manifest.update(oracle_sha256=hashlib.sha256(oracle_bytes).hexdigest(), oracle_events=oracle.total,
                        oracle_validator_sha256=hashlib.sha256(Path(__file__).with_name('query_oracle.py').read_bytes()).hexdigest())
        (args.out / 'oracle.json').write_bytes(oracle_bytes)
    (args.out / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
    status, _ = request(args.base_url.rstrip('/') + '/health', args.timeout)
    if not 200 <= status < 300:
        raise RuntimeError(f'query target health preflight failed: HTTP {status}')
    plan = workload(args.base_url, chains, args.contract, count, args.seed, oracle.bounds() if oracle else None)
    with (args.out / 'requests.jsonl').open('x') as stream:
        def emit(row):
            stream.write(json.dumps(row) + '\n')
            stream.flush()
        records, elapsed = measure(plan, args.rate, args.workers, args.queue_capacity, args.timeout, emit,
                                   validator=oracle.validate if oracle else None)
    result = summarize(records, count, args.duration, elapsed)
    result.update(cache_state=args.cache_state, oracle_validated=oracle is not None,
                  per_kind_counts=dict(Counter(row['kind'] for row in records)),
                  validation_p95_ms=percentile([row.get('validation_ms', 0) for row in records], 95))
    if oracle is not None:
        result['validation_scope'] = 'Exact PG snapshot ordered IDs, totals and fields; independent RPC truth still required'
    (args.out / 'result.json').write_text(json.dumps(result, indent=2) + '\n')
    print(json.dumps(result, indent=2))
    return 0 if result['performance_passed'] else 1


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (OSError, RuntimeError, ValueError, KeyError, subprocess.SubprocessError) as error:
        print(f'query measurement setup failed: {error}', file=sys.stderr)
        sys.exit(2)
