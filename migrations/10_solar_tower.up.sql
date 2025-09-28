INSERT INTO system_template_buildings (id, label, energy_type, energy_output)
VALUES ('solar_tower', '太阳能塔', 'storage', 220)
ON CONFLICT (id) DO UPDATE
    SET label = EXCLUDED.label,
        energy_type = EXCLUDED.energy_type,
        energy_output = EXCLUDED.energy_output,
        energy_capacity = NULL,
        energy_current = NULL,
        energy_rate = NULL;

INSERT INTO system_scene_buildings (
    id,
    scene_id,
    template_id,
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
) VALUES (
    'solar_tower_01',
    'mars_outpost_min',
    'solar_tower',
    '太阳能塔 01',
    44,
    22,
    4,
    4,
    NULL,
    NULL,
    NULL,
    NULL,
    NULL
)
ON CONFLICT (id) DO UPDATE
    SET scene_id = EXCLUDED.scene_id,
        template_id = EXCLUDED.template_id,
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
