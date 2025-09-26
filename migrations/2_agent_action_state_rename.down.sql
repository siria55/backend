ALTER TABLE IF EXISTS agent_action_state
    RENAME COLUMN actions TO behaviors;

ALTER TABLE IF EXISTS agent_action_state
    RENAME TO agent_behavior_state;
