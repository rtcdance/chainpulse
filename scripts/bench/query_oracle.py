"""Static PostgreSQL snapshot oracle for exact query parity, not chain truth.

Input: {"schema":"chainpulse.query-oracle.v1", "events":[...]}.
Each event carries eventId, chainId, blockNumber, logIndex, contractAddress,
transactionHash and eventName. Export canonical rows only; freeze writes while
measuring. The independent RPC digest is a separate E14 requirement.
"""
import bisect
from collections import Counter
import json


class QueryOracle:
    def __init__(self, document, chains):
        if document.get('schema') != 'chainpulse.query-oracle.v1':
            raise ValueError('unsupported query oracle schema')
        events = document.get('events')
        if not isinstance(events, list) or not events:
            raise ValueError('query oracle must contain events')
        self.chains = {chain: [] for chain in chains}
        self.contracts = {}
        self.positions = {}
        seen = set()
        for event in events:
            chain, identity = event['chainId'], event['eventId']
            if not isinstance(chain, str) or chain not in self.chains or not isinstance(identity, str) or not identity or identity in seen:
                raise ValueError('duplicate identity or undeclared oracle chain')
            if (type(event['blockNumber']) is not int or type(event['logIndex']) is not int
                    or event['blockNumber'] < 0 or event['logIndex'] < 0):
                raise ValueError('oracle block/log must be nonnegative integers')
            if not all(isinstance(event.get(key), str) for key in ('contractAddress', 'transactionHash', 'eventName')):
                raise ValueError('oracle event fields missing')
            seen.add(identity)
            self.chains[chain].append(event)
        for chain, rows in self.chains.items():
            if not rows:
                raise ValueError('each declared query chain needs snapshot events')
            rows.sort(key=lambda row: (row['blockNumber'], row['logIndex'], row['eventId']), reverse=True)
            self.positions[chain] = {row['eventId']: index for index, row in enumerate(rows)}
            for row in rows:
                self.contracts.setdefault((chain, row['contractAddress'].lower()), []).append(row)
        self.negative_blocks = {chain: [-row['blockNumber'] for row in rows] for chain, rows in self.chains.items()}
        self.total = len(seen)
        self.by_chain = {chain: len(rows) for chain, rows in self.chains.items()}
        self.by_name = dict(Counter(row['eventName'] for rows in self.chains.values() for row in rows))
        self.reorged = document.get('reorged', 0)
        if type(self.reorged) is not int or self.reorged < 0:
            raise ValueError('oracle reorged count must be nonnegative')

    def bounds(self):
        return {chain: (max(1, rows[-1]['blockNumber']), rows[0]['blockNumber'])
                for chain, rows in self.chains.items()}

    def validate(self, spec, body, cursor_anchor=None):
        doc = json.loads(body)
        if spec['kind'] == 'stats':
            if type(doc.get('total')) is not int or doc['total'] != self.total:
                return 'oracle_stats_total'
            for field, expected in [('byChain', self.by_chain), ('byEventName', self.by_name)]:
                actual = doc.get(field)
                if not isinstance(actual, dict) or actual != expected or any(type(value) is not int for value in actual.values()):
                    return 'oracle_stats_' + field
            if type(doc.get('reorged')) is not int or doc['reorged'] != self.reorged:
                return 'oracle_stats_reorged'
            return None
        chain = spec['chain']
        rows = self.chains[chain]
        start, end = 0, len(rows)
        if spec['kind'] == 'contract':
            rows = self.contracts.get((chain, spec['contract'].lower()), [])
            end = len(rows)
        elif spec['kind'] == 'range':
            blocks = self.negative_blocks[chain]
            start = bisect.bisect_left(blocks, -spec['to_block'])
            end = bisect.bisect_right(blocks, -spec['from_block'])
        total = end - start
        if cursor_anchor is not None:
            position = self.positions[chain].get(cursor_anchor)
            if position is None:
                return 'oracle_cursor_anchor'
            start = position + 1
        else:
            start += spec.get('offset', 0)
        expected = rows[start:min(end, start + spec['limit'])]
        page_total = (doc.get('pagination') or {}).get('total')
        if type(page_total) is not int or page_total != total:
            return 'oracle_page_total'
        actual = doc.get('data')
        if not isinstance(actual, list) or len(actual) != len(expected):
            return 'oracle_page_length'
        if (doc.get('meta') or {}).get('source') != 'postgres-canonical':
            return 'oracle_page_source'
        for got, want in zip(actual, expected):
            for field in ('eventId', 'chainId', 'blockNumber', 'logIndex', 'contractAddress', 'transactionHash', 'eventName'):
                value = got.get(field) if isinstance(got, dict) else None
                target = want[field]
                if field in ('contractAddress', 'transactionHash'):
                    value = value.lower() if isinstance(value, str) else value
                    target = target.lower()
                if type(value) is not type(target) or value != target:
                    return 'oracle_event_' + field
        return None
