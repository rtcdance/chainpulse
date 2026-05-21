-- ChainPulse Database Schema
-- Initialized by PostgreSQL container on first startup

-- Indexed blockchain events
CREATE TABLE IF NOT EXISTS events (
    id VARCHAR(255) PRIMARY KEY,
    chain_id VARCHAR(64) NOT NULL,
    block_number BIGINT NOT NULL,
    block_hash VARCHAR(128) NOT NULL,
    transaction_hash VARCHAR(128) NOT NULL,
    log_index INTEGER NOT NULL,
    contract_address VARCHAR(128) NOT NULL,
    event_name VARCHAR(255) NOT NULL,
    event_data JSONB,
    timestamp BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_events_chain_id ON events(chain_id);
CREATE INDEX IF NOT EXISTS idx_events_block_number ON events(block_number);
CREATE INDEX IF NOT EXISTS idx_events_contract_address ON events(contract_address);
CREATE INDEX IF NOT EXISTS idx_events_event_name ON events(event_name);
CREATE INDEX IF NOT EXISTS idx_events_transaction_hash ON events(transaction_hash);

-- Indexed blockchain blocks
CREATE TABLE IF NOT EXISTS blocks (
    id VARCHAR(255) PRIMARY KEY,
    chain_id VARCHAR(64) NOT NULL,
    number BIGINT NOT NULL,
    hash VARCHAR(128) NOT NULL,
    parent_hash VARCHAR(128),
    timestamp BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_blocks_chain_id ON blocks(chain_id);
CREATE INDEX IF NOT EXISTS idx_blocks_number ON blocks(number);

-- Dead Letter Queue events
CREATE TABLE IF NOT EXISTS dlq_events (
    id VARCHAR(255) PRIMARY KEY,
    chain_id VARCHAR(64) NOT NULL,
    original_event_id VARCHAR(255),
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    status VARCHAR(32) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_dlq_events_status ON dlq_events(status);
CREATE INDEX IF NOT EXISTS idx_dlq_events_chain_id ON dlq_events(chain_id);

-- Per-chain indexing state tracker
CREATE TABLE IF NOT EXISTS indexing_state (
    chain_id VARCHAR(64) PRIMARY KEY,
    last_indexed_block BIGINT DEFAULT 0,
    last_indexed_hash VARCHAR(128),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
