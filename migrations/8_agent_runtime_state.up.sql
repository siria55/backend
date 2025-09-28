CREATE TABLE IF NOT EXISTS agent_runtime_state (
    agent_id   TEXT PRIMARY KEY REFERENCES system_scene_agents(id) ON DELETE CASCADE,
    pos_x      INT NOT NULL,
    pos_y      INT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
