CREATE TABLE IF NOT EXISTS agents (
    id           TEXT PRIMARY KEY,
    label        TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS agent_action_events (
    id              BIGSERIAL PRIMARY KEY,
    agent_id        TEXT NOT NULL REFERENCES agents(id),
    action_type     TEXT NOT NULL,
    payload         JSONB,
    issued_by       TEXT,
    source          TEXT,
    correlation_id  TEXT,
    result_status   TEXT,
    result_message  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_action_events_agent_time
    ON agent_action_events (agent_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_agent_action_events_correlation
    ON agent_action_events (correlation_id);

CREATE TABLE IF NOT EXISTS agent_behavior_state (
    agent_id     TEXT PRIMARY KEY REFERENCES agents(id),
    behaviors    TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
