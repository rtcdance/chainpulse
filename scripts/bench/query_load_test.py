"""Deterministic regression controls for the query measurement driver."""
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

SPEC = importlib.util.spec_from_file_location('query_load', Path(__file__).with_name('query-load.py'))
LOAD = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(LOAD)


class QueryLoadTests(unittest.TestCase):
    def test_workload_seed_and_real_contract_parameter(self):
        plan = LOAD.workload('http://localhost', ['1', '2'], '0xabc', 100, 7)
        self.assertEqual(plan, LOAD.workload('http://localhost', ['1', '2'], '0xabc', 100, 7))
        self.assertEqual({'offset', 'cursor', 'stats', 'contract'}, {p['kind'] for p in plan})
        for row in plan:
            if row['kind'] == 'contract':
                self.assertIn('contract=0xabc', row['url'])

    def test_event_page_requires_data_and_pagination(self):
        self.assertIsNone(LOAD.shape_error('offset', b'{"data":[],"pagination":{}}'))
        for body in (b'[]', b'bad', b'{"pagination":{}}', b'{"events":[]}'):
            self.assertIsNotNone(LOAD.shape_error('offset', body))

    def test_stats_requires_nonnegative_integer(self):
        self.assertIsNone(LOAD.shape_error('stats', b'{"total":0}'))
        for body in (b'{}', b'{"total":-1}', b'{"total":"1"}', b'{"total":true}'):
            self.assertIsNotNone(LOAD.shape_error('stats', body))

    def test_failed_latency_is_in_percentiles_and_status_counts(self):
        rows = [dict(index=0, error='http_error', status=503, latency_ms=1000,
                     started_s=0, cursor_used=False)]
        result = LOAD.summarize(rows, 1, 1, 1)
        self.assertEqual(1000, result['p95_ms'])
        self.assertEqual({'503': 1}, result['status_counts'])
        self.assertFalse(result['performance_passed'])

    def test_duplicate_or_missing_sample_rejected(self):
        row = dict(index=0, error=None, status=200, latency_ms=1, started_s=0, cursor_used=False)
        self.assertFalse(LOAD.summarize([row, row], 2, 1, 1)['verdicts']['complete_samples'])
        self.assertFalse(LOAD.summarize([], 1, 1, 1)['performance_passed'])

    def test_overload_is_bounded_counted_and_not_retried(self):
        release = threading.Event()
        def fetch(url, timeout):
            self.assertTrue(release.wait(5), 'test control must release worker')
            return 200, b'{"total":0}'
        def emit(row):
            if row['error'] == 'generator_overload':
                release.set()
        plan = [dict(index=i, kind='stats', chain='1', url='unused') for i in range(20)]
        rows, elapsed = LOAD.measure(plan, 1e9, 1, 1, 1, emit, fetch)
        self.assertEqual(20, len(rows))
        result = LOAD.summarize(rows, 20, 1, elapsed)
        self.assertGreater(result['overload'], 0)
        self.assertFalse(result['performance_passed'])

    def test_unexpected_worker_error_still_records_terminal_sample(self):
        def broken(url, timeout):
            raise TypeError('transport defect')
        plan = [dict(index=i, kind='stats', chain='1', url='unused') for i in range(3)]
        rows, _ = LOAD.measure(plan, 1000, 1, 3, 1, lambda row: None, broken)
        self.assertEqual(['TypeError'] * 3, [row['error'] for row in rows])

    def test_http_503_preserves_status(self):
        class Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                self.send_response(503)
                self.end_headers()
                self.wfile.write(b'busy')
            def log_message(self, *args):
                pass
        server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
        thread = threading.Thread(target=server.serve_forever)
        thread.start()
        try:
            self.assertEqual((503, b'busy'), LOAD.request(f'http://127.0.0.1:{server.server_port}', 1))
        finally:
            server.shutdown()
            thread.join()
            server.server_close()

    def test_existing_output_is_rejected_without_modification(self):
        with tempfile.TemporaryDirectory() as directory:
            marker = Path(directory) / 'requests.jsonl'
            marker.write_text('prior evidence\n')
            result = subprocess.run([sys.executable, str(Path(__file__).with_name('query-load.py')),
                                     '--base-url', 'http://localhost', '--contract', '0xabc',
                                     '--out', directory], capture_output=True)
            self.assertEqual(2, result.returncode)
            self.assertEqual('prior evidence\n', marker.read_text())


if __name__ == '__main__':
    unittest.main()
