#!/usr/bin/env python3
"""Export a frozen PG query oracle. Does not prove independent RPC correctness.

CHAINPULSE_BENCH_DATABASE_URL supplies credentials; never written to artifacts.
Requires psql. Stop/freeze event writers before export and query measurement.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys

from query_oracle import QueryOracle


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--out', type=Path, required=True)
    parser.add_argument('--chains', default='31337,31338')
    parser.add_argument('--min-events', type=int, default=100000)
    args = parser.parse_args()
    chains = args.chains.split(',')
    if not 1 <= len(chains) <= 50 or len(set(chains)) != len(chains) or any(not re.fullmatch('[1-9][0-9]{0,18}', chain) for chain in chains):
        parser.error('declare 1..50 unique positive numeric chain IDs')
    if not 1 <= args.min_events <= 500000:
        parser.error('min-events must be 1..500000')
    endpoint = os.environ.get('CHAINPULSE_BENCH_DATABASE_URL')
    if not endpoint:
        parser.error('CHAINPULSE_BENCH_DATABASE_URL is required')
    args.out.parent.mkdir(parents=True, exist_ok=True)
    with args.out.open('x') as output:
        sql = '''BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
WITH scope AS (SELECT jsonb_array_elements_text(:'chains'::jsonb) AS chain_id),
events AS (SELECT e.* FROM blockchain_events e JOIN scope s USING(chain_id)
 WHERE status<>'reorged' AND event_state->>'canonicality'='canonical' LIMIT 500001)
SELECT jsonb_build_object('schema','chainpulse.query-oracle.v1',
 'events',COALESCE((SELECT jsonb_agg(jsonb_build_object('eventId',id,'chainId',chain_id,
 'blockNumber',block_number,'logIndex',log_index,'contractAddress',contract_address,
 'transactionHash',transaction_hash,'eventName',event_name)) FROM events),'[]'::jsonb),
 'reorged',(SELECT count(*) FROM blockchain_events e JOIN scope s USING(chain_id) WHERE status='reorged'),
 'provenance',jsonb_build_object('database',current_database(),'snapshot',pg_current_snapshot()::text,
 'generatedAt',clock_timestamp(),'scope',:'chains'::jsonb,'purpose','query parity only; not RPC truth'));
COMMIT;
'''
        try:
            result = subprocess.run(['psql', '-X', '-qAt', '--set', 'ON_ERROR_STOP=1', '--set', 'chains=' + json.dumps(chains),
                                     '--dbname', endpoint], input=sql, capture_output=True, text=True, timeout=120)
        except subprocess.TimeoutExpired:
            raise RuntimeError('PostgreSQL oracle export timed out') from None
        if result.returncode:
            # CalledProcessError includes command arguments (and credentials).
            raise RuntimeError(f'PostgreSQL oracle export failed: exit {result.returncode}')
        document = json.loads(result.stdout)
        oracle = QueryOracle(document, chains)
        if not args.min_events <= oracle.total <= 500000:
            raise ValueError(f'need {args.min_events}..500000 canonical events; found {oracle.total}')
        output.write(json.dumps(document, separators=(',', ':')) + '\n')
    print(json.dumps(dict(path=str(args.out), events=oracle.total, chains=chains,
                          sha256=hashlib.sha256(args.out.read_bytes()).hexdigest(), independent_rpc_oracle=False)))


if __name__ == '__main__':
    try:
        main()
    except (OSError, RuntimeError, ValueError, KeyError) as error:
        print(f'query oracle export failed: {error}', file=sys.stderr)
        sys.exit(2)
