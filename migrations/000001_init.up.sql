CREATE TABLE IF NOT EXISTS metrics (
    id         TEXT        NOT NULL,
    mtype      TEXT        NOT NULL CHECK (mtype IN ('gauge','counter')),
    value      DOUBLE PRECISION,
    delta      BIGINT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (id, mtype)
    );
CREATE INDEX IF NOT EXISTS idx_metrics_updated_at ON metrics(updated_at);