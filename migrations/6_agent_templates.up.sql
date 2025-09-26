CREATE TABLE IF NOT EXISTS system_agent_templates (
    id         TEXT PRIMARY KEY,
    label      TEXT NOT NULL,
    color      INT,
    default_position_x INT,
    default_position_y INT
);

INSERT INTO system_agent_templates (id, label, color, default_position_x, default_position_y) VALUES
    ('ares', '阿瑞斯型指挥体', 11541703, 18, 14),
    ('support', '支援型无人机', 11576490, 22, 14)
ON CONFLICT (id) DO UPDATE SET
    label = EXCLUDED.label,
    color = EXCLUDED.color,
    default_position_x = EXCLUDED.default_position_x,
    default_position_y = EXCLUDED.default_position_y;

ALTER TABLE system_scene_agents
    ADD COLUMN IF NOT EXISTS template_id TEXT;

UPDATE system_scene_agents
SET template_id = CASE
        WHEN id LIKE 'ares%' THEN 'ares'
        WHEN id LIKE 'support%' THEN 'support'
        ELSE id
    END
WHERE template_id IS NULL;

ALTER TABLE system_scene_agents
    ADD CONSTRAINT system_scene_agents_template_id_fkey
    FOREIGN KEY (template_id) REFERENCES system_agent_templates(id);
