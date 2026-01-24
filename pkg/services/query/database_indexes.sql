-- Database Index Migration Script for Web3 Indexer
-- This script creates optimal indexes for blockchain event queries
-- Requirements: 3.7, 3.8

-- ============================================================================
-- Single Column Indexes
-- ============================================================================

-- Index on block_number for efficient block range queries
-- Used by: EventFilter with FromBlock/ToBlock
-- Selectivity: High (events distributed across many blocks)
CREATE INDEX IF NOT EXISTS idx_block_number 
ON events(block_number DESC);

-- Index on contract_address for contract-specific queries
-- Used by: EventFilter with ContractAddress
-- Selectivity: Very High (many events per contract)
CREATE INDEX IF NOT EXISTS idx_contract_address 
ON events(contract_address);

-- Index on event_signature for event type queries
-- Used by: EventFilter with EventSignature
-- Selectivity: High (many events per signature)
CREATE INDEX IF NOT EXISTS idx_event_signature 
ON events(event_signature);

-- Index on block_timestamp for time-range queries
-- Used by: EventFilter with FromTimestamp/ToTimestamp
-- Selectivity: Medium (events distributed across time)
CREATE INDEX IF NOT EXISTS idx_block_timestamp 
ON events(block_timestamp DESC);

-- Index on status for status-filtered queries
-- Used by: EventFilter with Status
-- Selectivity: Medium (events in different states)
CREATE INDEX IF NOT EXISTS idx_status 
ON events(status);

-- Index on transaction_hash for transaction lookups
-- Used by: Transaction-specific queries
-- Selectivity: Very High (unique per transaction)
CREATE INDEX IF NOT EXISTS idx_transaction_hash 
ON events(transaction_hash);

-- Index on log_index for event ordering within blocks
-- Used by: Ordering and pagination
-- Selectivity: High (unique per block)
CREATE INDEX IF NOT EXISTS idx_log_index 
ON events(log_index);

-- Index on network for multi-chain queries
-- Used by: EventFilter with Network
-- Selectivity: Low (few networks)
CREATE INDEX IF NOT EXISTS idx_network 
ON events(network);

-- ============================================================================
-- Composite Indexes (Most Frequently Used Combinations)
-- ============================================================================

-- Composite index for contract + block range queries
-- Used by: EventFilter with ContractAddress + FromBlock/ToBlock
-- Query pattern: WHERE contract_address = ? AND block_number BETWEEN ? AND ?
-- Selectivity: Very High (most selective combination)
CREATE INDEX IF NOT EXISTS idx_contract_block 
ON events(contract_address, block_number DESC);

-- Composite index for contract + event signature queries
-- Used by: EventFilter with ContractAddress + EventSignature
-- Query pattern: WHERE contract_address = ? AND event_signature = ?
-- Selectivity: Very High (highly selective)
CREATE INDEX IF NOT EXISTS idx_contract_signature 
ON events(contract_address, event_signature);

-- Composite index for network + block range queries
-- Used by: Multi-chain queries with block filtering
-- Query pattern: WHERE network = ? AND block_number BETWEEN ? AND ?
-- Selectivity: High (network + block range)
CREATE INDEX IF NOT EXISTS idx_network_block 
ON events(network, block_number DESC);

-- Composite index for block range + status queries
-- Used by: EventFilter with FromBlock/ToBlock + Status
-- Query pattern: WHERE block_number BETWEEN ? AND ? AND status = ?
-- Selectivity: High (block range + status)
CREATE INDEX IF NOT EXISTS idx_block_status 
ON events(block_number DESC, status);

-- Composite index for timestamp + status queries
-- Used by: EventFilter with FromTimestamp/ToTimestamp + Status
-- Query pattern: WHERE block_timestamp BETWEEN ? AND ? AND status = ?
-- Selectivity: Medium (timestamp + status)
CREATE INDEX IF NOT EXISTS idx_timestamp_status 
ON events(block_timestamp DESC, status);

-- Composite index for contract + timestamp queries
-- Used by: Contract-specific time-range queries
-- Query pattern: WHERE contract_address = ? AND block_timestamp BETWEEN ? AND ?
-- Selectivity: Very High (contract + timestamp)
CREATE INDEX IF NOT EXISTS idx_contract_timestamp 
ON events(contract_address, block_timestamp DESC);

-- Composite index for network + contract + block queries
-- Used by: Multi-chain contract queries with block filtering
-- Query pattern: WHERE network = ? AND contract_address = ? AND block_number BETWEEN ? AND ?
-- Selectivity: Very High (most selective combination)
CREATE INDEX IF NOT EXISTS idx_network_contract_block 
ON events(network, contract_address, block_number DESC);

-- ============================================================================
-- Covering Indexes (Include Frequently Selected Columns)
-- ============================================================================

-- Covering index for common event queries
-- Includes: event_name, decoded_data for faster retrieval
-- Used by: Queries that need event details without table lookup
CREATE INDEX IF NOT EXISTS idx_block_number_covering 
ON events(block_number DESC) 
INCLUDE (event_name, decoded_data);

-- Covering index for contract queries
-- Includes: event_signature, block_timestamp for faster retrieval
-- Used by: Contract-specific queries needing event details
CREATE INDEX IF NOT EXISTS idx_contract_address_covering 
ON events(contract_address) 
INCLUDE (event_signature, block_timestamp);

-- ============================================================================
-- Partial Indexes (For Specific Conditions)
-- ============================================================================

-- Partial index for confirmed events only
-- Used by: Queries filtering for confirmed events
-- Reduces index size by excluding pending/failed events
CREATE INDEX IF NOT EXISTS idx_confirmed_events 
ON events(block_number DESC) 
WHERE status = 'confirmed';

-- Partial index for recent events
-- Used by: Recent event queries
-- Reduces index size by excluding old events
CREATE INDEX IF NOT EXISTS idx_recent_events 
ON events(block_number DESC) 
WHERE block_timestamp > (EXTRACT(EPOCH FROM NOW()) - 86400);

-- Partial index for specific contract events
-- Used by: Queries for specific high-volume contracts
-- Reduces index size by focusing on important contracts
CREATE INDEX IF NOT EXISTS idx_uniswap_events 
ON events(block_number DESC) 
WHERE contract_address = '0x1F98431c8aD98523631AE4a59f267346ea31F984';

-- ============================================================================
-- Index Statistics and Maintenance
-- ============================================================================

-- Analyze table to update statistics
-- This helps the query planner make better decisions
ANALYZE events;

-- ============================================================================
-- Index Usage Monitoring Queries
-- ============================================================================

-- Query to check index usage (PostgreSQL)
-- SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read, idx_tup_fetch
-- FROM pg_stat_user_indexes
-- WHERE tablename = 'events'
-- ORDER BY idx_scan DESC;

-- Query to find unused indexes (PostgreSQL)
-- SELECT schemaname, tablename, indexname, idx_scan
-- FROM pg_stat_user_indexes
-- WHERE tablename = 'events' AND idx_scan = 0
-- ORDER BY pg_relation_size(indexrelid) DESC;

-- Query to find missing indexes (PostgreSQL)
-- SELECT schemaname, tablename, attname, n_distinct, correlation
-- FROM pg_stats
-- WHERE tablename = 'events'
-- ORDER BY abs(correlation) DESC;

-- ============================================================================
-- Index Maintenance Procedures
-- ============================================================================

-- Rebuild fragmented indexes (PostgreSQL)
-- REINDEX INDEX idx_block_number;
-- REINDEX INDEX idx_contract_address;
-- REINDEX INDEX idx_event_signature;

-- Vacuum and analyze for optimal performance
-- VACUUM ANALYZE events;

-- ============================================================================
-- Performance Tuning Parameters
-- ============================================================================

-- Set work_mem for large sorts (PostgreSQL)
-- SET work_mem = '256MB';

-- Set effective_cache_size for better planning (PostgreSQL)
-- SET effective_cache_size = '4GB';

-- Set random_page_cost for SSD optimization (PostgreSQL)
-- SET random_page_cost = 1.1;
