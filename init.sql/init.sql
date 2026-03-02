CREATE TABLE IF NOT EXISTS events (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    event_name TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    properties JSONB
);


CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events (timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_events_name_time ON events (event_name, timestamp DESC);