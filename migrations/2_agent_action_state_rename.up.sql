ALTER TABLE IF EXISTS agent_behavior_state
    RENAME TO agent_action_state;

ALTER TABLE IF EXISTS agent_action_state
    RENAME COLUMN behaviors TO actions;
