INSERT INTO system_scene_buildings (
    id,
    scene_id,
    label,
    position_x,
    position_y,
    size_width,
    size_height,
    energy_type,
    energy_capacity,
    energy_current,
    energy_output,
    energy_rate
) VALUES
    ('central_dome', 'mars_outpost_min', '联合指挥穹顶', 10, 6, 8, 6, 'consumer', NULL, NULL, NULL, 160),
    ('research_lab', 'mars_outpost_min', '岩土研究站', 15, 18, 9, 6, 'consumer', NULL, NULL, NULL, 110),
    ('solar_array', 'mars_outpost_min', '太阳能阵列', 4, 24, 12, 5, 'storage', 240, 160, 120, NULL)
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
