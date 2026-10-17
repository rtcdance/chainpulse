import copy
import importlib.util
import json
from pathlib import Path
import unittest

SPEC = importlib.util.spec_from_file_location('query_oracle', Path(__file__).with_name('query_oracle.py'))
ORACLE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(ORACLE)


class QueryOracleTests(unittest.TestCase):
    def setUp(self):
        self.events = [dict(eventId=f'{chain}-{number}', chainId=chain, blockNumber=number,
                            logIndex=0, contractAddress='0xabc', transactionHash=f'0x{number}', eventName='Transfer')
                       for chain in ('1', '2') for number in range(1, 5)]
        self.document = dict(schema='chainpulse.query-oracle.v1', events=self.events)
        self.oracle = ORACLE.QueryOracle(self.document, ['1', '2'])

    def page(self, ids, total=4):
        return dict(data=[copy.deepcopy(row) for identity in ids for row in self.events if row['eventId'] == identity],
                    pagination=dict(total=total), meta=dict(source='postgres-canonical'))

    def test_offset_and_exact_order(self):
        spec = dict(kind='offset', chain='1', offset=1, limit=2)
        self.assertIsNone(self.oracle.validate(spec, json.dumps(self.page(['1-3', '1-2']))))
        self.assertEqual('oracle_event_eventId', self.oracle.validate(spec, json.dumps(self.page(['1-2', '1-3']))))

    def test_range_and_contract_filters(self):
        spec = dict(kind='range', chain='1', from_block=2, to_block=3, limit=50)
        self.assertIsNone(self.oracle.validate(spec, json.dumps(self.page(['1-3', '1-2'], 2))))
        spec = dict(kind='contract', chain='1', contract='0xDEF', limit=50)
        self.assertIsNone(self.oracle.validate(spec, json.dumps(self.page([], 0))))

    def test_cursor_follows_actual_anchor(self):
        spec = dict(kind='cursor', chain='1', limit=2)
        self.assertIsNone(self.oracle.validate(spec, json.dumps(self.page(['1-2', '1-1'])), '1-3'))
        self.assertEqual('oracle_cursor_anchor', self.oracle.validate(spec, json.dumps(self.page([])), 'missing'))

    def test_cross_chain_corruption_and_total_rejected(self):
        spec = dict(kind='offset', chain='1', offset=0, limit=1)
        self.assertEqual('oracle_event_eventId', self.oracle.validate(spec, json.dumps(self.page(['2-4']))))
        self.assertEqual('oracle_page_total', self.oracle.validate(spec, json.dumps(self.page(['1-4'], 3))))
        self.assertEqual('oracle_stats_total', self.oracle.validate(dict(kind='stats'), b'{"total":7}'))
        stats = dict(total=8, byChain={'1': 4, '2': 4}, byEventName={'Transfer': 8}, reorged=0)
        self.assertIsNone(self.oracle.validate(dict(kind='stats'), json.dumps(stats)))
        stats['byChain'] = {'1': 8}
        self.assertEqual('oracle_stats_byChain', self.oracle.validate(dict(kind='stats'), json.dumps(stats)))

    def test_duplicate_or_incomplete_input_rejected(self):
        self.document['events'].append(self.events[0])
        with self.assertRaises(ValueError):
            ORACLE.QueryOracle(self.document, ['1', '2'])
        with self.assertRaises(ValueError):
            ORACLE.QueryOracle(dict(schema='chainpulse.query-oracle.v1', events=[]), ['1'])


if __name__ == '__main__':
    unittest.main()
