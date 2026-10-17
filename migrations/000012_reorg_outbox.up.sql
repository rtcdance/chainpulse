-- Persistent reorg repair jobs and their transactional outbox (T06 / F07).
--
-- Before this, HandleReorg rolled back events and then published notifications
-- best-effort: a publish failure was logged and forgotten, and an interrupted
-- reorg left no record that repair was owed. Two tables fix that:
--
--   reorg_jobs    — the repair work itself, so an interrupted reorg is
--                   discoverable and resumable after a restart.
--   reorg_outbox  — the notifications owed for a job, written in the SAME
--                   transaction as the job. A crash can therefore leave
--                   "job recorded, notification not yet sent", which a drainer
--                   picks up, but never "rollback done, notification lost".
--
-- The outbox is drained at least once; consumers must be idempotent (each
-- notification carries the job id and generation).
CREATE TABLE IF NOT EXISTS reorg_jobs (
    id         VARCHAR(255) PRIMARY KEY,
    chain_id   VARCHAR(64)  NOT NULL,
    generation BIGINT       NOT NULL,
    from_block BIGINT       NOT NULL,
    to_block   BIGINT       NOT NULL,
    status     VARCHAR(32)  NOT NULL,
    created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reorg_jobs_pending
    ON reorg_jobs (chain_id, created_at)
    WHERE status <> 'done';

CREATE TABLE IF NOT EXISTS reorg_outbox (
    id           BIGSERIAL PRIMARY KEY,
    job_id       VARCHAR(255) NOT NULL REFERENCES reorg_jobs(id) ON DELETE CASCADE,
    topic        VARCHAR(128) NOT NULL,
    payload      JSONB        NOT NULL,
    attempts     INT          NOT NULL DEFAULT 0,
    published_at TIMESTAMP,
    created_at   TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- The drainer's query is "oldest unpublished first", so index exactly that.
CREATE INDEX IF NOT EXISTS idx_reorg_outbox_pending
    ON reorg_outbox (id)
    WHERE published_at IS NULL;
