CREATE TABLE IF NOT EXISTS system_building_templates (
    id              TEXT PRIMARY KEY,
    label           TEXT NOT NULL,
    energy_type     TEXT,
    energy_capacity INT,
    energy_current  INT,
    energy_output   INT,
    energy_rate     INT
);

INSERT INTO system_building_templates (id, label, energy_type, energy_capacity, energy_current, energy_output, energy_rate) VALUES
    ('central_dome', '联合指挥穹顶', 'consumer', NULL, NULL, NULL, 160),
    ('habitat_block', '居住平台', 'consumer', NULL, NULL, NULL, 90),
    ('research_lab', '岩土研究站', 'consumer', NULL, NULL, NULL, 110),
    ('solar_array', '太阳能阵列', 'storage', 240, 160, 120, NULL),
    ('logistics_hub', '物资枢纽', 'consumer', NULL, NULL, NULL, 140),
    ('medical_bay', '医疗站', 'consumer', NULL, NULL, NULL, 80),
    ('power_station', '能源塔阵列', 'storage', 420, 160, 120, NULL)
ON CONFLICT (id) DO UPDATE SET
    label = EXCLUDED.label,
    energy_type = EXCLUDED.energy_type,
    energy_capacity = EXCLUDED.energy_capacity,
    energy_current = EXCLUDED.energy_current,
    energy_output = EXCLUDED.energy_output,
    energy_rate = EXCLUDED.energy_rate;

ALTER TABLE system_scene_buildings
    ADD COLUMN IF NOT EXISTS template_id TEXT;

UPDATE system_scene_buildings
SET template_id = id
WHERE template_id IS NULL;

ALTER TABLE system_scene_buildings
    ADD CONSTRAINT system_scene_buildings_template_id_fkey
    FOREIGN KEY (template_id) REFERENCES system_building_templates(id);
