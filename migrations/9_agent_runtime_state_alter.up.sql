ALTER TABLE agent_runtime_state
    ALTER COLUMN pos_x TYPE DOUBLE PRECISION USING pos_x::double precision,
    ALTER COLUMN pos_y TYPE DOUBLE PRECISION USING pos_y::double precision;
