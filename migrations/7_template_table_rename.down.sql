ALTER TABLE system_scene_agents
    DROP CONSTRAINT IF EXISTS system_scene_agents_template_id_fkey;

ALTER TABLE system_template_agents
    RENAME TO system_agent_templates;

ALTER TABLE system_scene_agents
    ADD CONSTRAINT system_scene_agents_template_id_fkey
    FOREIGN KEY (template_id) REFERENCES system_agent_templates(id);

ALTER TABLE system_scene_buildings
    DROP CONSTRAINT IF EXISTS system_scene_buildings_template_id_fkey;

ALTER TABLE system_template_buildings
    RENAME TO system_building_templates;

ALTER TABLE system_scene_buildings
    ADD CONSTRAINT system_scene_buildings_template_id_fkey
    FOREIGN KEY (template_id) REFERENCES system_building_templates(id);
