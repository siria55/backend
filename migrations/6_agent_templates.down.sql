ALTER TABLE system_scene_agents
    DROP CONSTRAINT IF EXISTS system_scene_agents_template_id_fkey;

ALTER TABLE system_scene_agents
    DROP COLUMN IF EXISTS template_id;

DROP TABLE IF EXISTS system_agent_templates;
