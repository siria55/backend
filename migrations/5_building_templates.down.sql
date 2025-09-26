ALTER TABLE system_scene_buildings
    DROP CONSTRAINT IF EXISTS system_scene_buildings_template_id_fkey;

ALTER TABLE system_scene_buildings
    DROP COLUMN IF EXISTS template_id;

DROP TABLE IF EXISTS system_building_templates;
