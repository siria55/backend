ALTER TABLE agent_runtime_state
    ALTER COLUMN pos_x TYPE INT USING ROUND(pos_x)::int,
    ALTER COLUMN pos_y TYPE INT USING ROUND(pos_y)::int;
