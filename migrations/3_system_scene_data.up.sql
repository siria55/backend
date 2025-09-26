CREATE TABLE IF NOT EXISTS system_scenes (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS system_scene_grid (
    scene_id  TEXT PRIMARY KEY REFERENCES system_scenes(id) ON DELETE CASCADE,
    cols      INT  NOT NULL,
    rows      INT  NOT NULL,
    tile_size INT  NOT NULL
);

CREATE TABLE IF NOT EXISTS system_scene_dimensions (
    scene_id TEXT PRIMARY KEY REFERENCES system_scenes(id) ON DELETE CASCADE,
    width    INT NOT NULL,
    height   INT NOT NULL
);

CREATE TABLE IF NOT EXISTS system_scene_buildings (
    id              TEXT PRIMARY KEY,
    scene_id        TEXT NOT NULL REFERENCES system_scenes(id) ON DELETE CASCADE,
    label           TEXT NOT NULL,
    position_x      INT  NOT NULL,
    position_y      INT  NOT NULL,
    size_width      INT  NOT NULL,
    size_height     INT  NOT NULL,
    energy_type     TEXT,
    energy_capacity INT,
    energy_current  INT,
    energy_output   INT,
    energy_rate     INT
);

CREATE TABLE IF NOT EXISTS system_scene_agents (
    id         TEXT PRIMARY KEY,
    scene_id   TEXT NOT NULL REFERENCES system_scenes(id) ON DELETE CASCADE,
    label      TEXT NOT NULL,
    position_x INT  NOT NULL,
    position_y INT  NOT NULL,
    color      INT
);

CREATE TABLE IF NOT EXISTS system_scene_agent_actions (
    agent_id TEXT NOT NULL REFERENCES system_scene_agents(id) ON DELETE CASCADE,
    action   TEXT NOT NULL,
    PRIMARY KEY (agent_id, action)
);

INSERT INTO system_scenes (id, name) VALUES
    ('mars_outpost_min', '火星前哨站 · 原型')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name;

INSERT INTO system_scene_grid (scene_id, cols, rows, tile_size) VALUES
    ('mars_outpost_min', 200, 200, 1)
ON CONFLICT (scene_id) DO UPDATE SET cols = EXCLUDED.cols,
    rows = EXCLUDED.rows,
    tile_size = EXCLUDED.tile_size;

INSERT INTO system_scene_dimensions (scene_id, width, height) VALUES
    ('mars_outpost_min', 200, 200)
ON CONFLICT (scene_id) DO UPDATE SET width = EXCLUDED.width,
    height = EXCLUDED.height;

INSERT INTO system_scene_buildings (id, scene_id, label, position_x, position_y, size_width, size_height, energy_type, energy_capacity, energy_current, energy_output, energy_rate) VALUES
    ('central_dome', 'mars_outpost_min', '联合指挥穹顶', 10, 6, 8, 6, 'consumer', NULL, NULL, NULL, 160),
    ('habitat_block', 'mars_outpost_min', '居住平台', 24, 6, 7, 6, 'consumer', NULL, NULL, NULL, 90),
    ('research_lab', 'mars_outpost_min', '岩土研究站', 15, 18, 9, 6, 'consumer', NULL, NULL, NULL, 110),
    ('solar_array', 'mars_outpost_min', '太阳能阵列', 4, 24, 12, 5, 'storage', 240, 160, 120, NULL),
    ('logistics_hub', 'mars_outpost_min', '物资枢纽', 30, 20, 9, 6, 'consumer', NULL, NULL, NULL, 140),
    ('medical_bay', 'mars_outpost_min', '医疗站', 40, 10, 6, 5, 'consumer', NULL, NULL, NULL, 80),
    ('power_station', 'mars_outpost_min', '能源塔阵列', 32, 10, 7, 5, 'storage', 420, 160, 120, NULL)
ON CONFLICT (id) DO UPDATE SET
    scene_id = EXCLUDED.scene_id,
    label = EXCLUDED.label,
    position_x = EXCLUDED.position_x,
    position_y = EXCLUDED.position_y,
    size_width = EXCLUDED.size_width,
    size_height = EXCLUDED.size_height,
    energy_type = EXCLUDED.energy_type,
    energy_capacity = EXCLUDED.energy_capacity,
    energy_current = EXCLUDED.energy_current,
    energy_output = EXCLUDED.energy_output,
    energy_rate = EXCLUDED.energy_rate;

INSERT INTO system_scene_agents (id, scene_id, label, position_x, position_y, color) VALUES
    ('ares-01', 'mars_outpost_min', '阿瑞斯-01 指挥体', 18, 14, 11541703),
    ('support-02', 'mars_outpost_min', '支援单位-02', 22, 14, 11576490)
ON CONFLICT (id) DO UPDATE SET
    scene_id = EXCLUDED.scene_id,
    label = EXCLUDED.label,
    position_x = EXCLUDED.position_x,
    position_y = EXCLUDED.position_y,
    color = EXCLUDED.color;

INSERT INTO system_scene_agent_actions (agent_id, action) VALUES
    ('ares-01', 'move_left'),
    ('ares-01', 'move_right'),
    ('ares-01', 'move_up'),
    ('ares-01', 'move_down'),
    ('support-02', 'move_left'),
    ('support-02', 'move_right'),
    ('support-02', 'move_up'),
    ('support-02', 'move_down')
ON CONFLICT (agent_id, action) DO NOTHING;
