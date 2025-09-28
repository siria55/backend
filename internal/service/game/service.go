package game

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Service 负责提供游戏场景配置等业务能力。
type Service struct {
	db    *sql.DB
	scene Scene
}

const (
	DefaultDrainFactor   = 1.0
	solarTowerTemplateID = "solar_tower"
)

// MaintainEnergyResult 描述“保持电量不减少”指令的执行结果。
type MaintainEnergyResult struct {
	Scene         Scene
	Created       []SceneBuilding
	NetFlowBefore float64
	NetFlowAfter  float64
	TowersBuilt   int
}

// New 返回默认的 Game 服务实例，并加载初始场景配置。
func New(db *sql.DB, sceneID string) (*Service, error) {
	scene, err := sceneLoader(db, sceneID)
	if err != nil {
		return nil, err
	}
	return &Service{db: db, scene: scene}, nil
}

// Scene 返回静态场景配置。
func (s *Service) Scene() Scene {
	return s.scene
}

// Snapshot 返回整合后的系统场景原始数据，供管理端查看。
func (s *Service) Snapshot() Snapshot {
	return Snapshot{
		Scene:             SceneMeta{ID: s.scene.ID, Name: s.scene.Name},
		Grid:              s.scene.Grid,
		Dimensions:        s.scene.Dimensions,
		Buildings:         s.scene.Buildings,
		Agents:            s.scene.Agents,
		BuildingTemplates: s.scene.BuildingTemplates,
		AgentTemplates:    s.scene.AgentTemplates,
	}
}

var (
	ErrInvalidSceneConfig   = errors.New("invalid scene config")
	ErrInvalidTemplate      = errors.New("invalid template")
	ErrInvalidSceneEntity   = errors.New("invalid scene entity")
	ErrSolarTemplateMissing = errors.New("solar tower template unavailable")
	ErrNoAvailablePlacement = errors.New("no available placement for solar tower")
)

// UpdateSceneConfig 更新 system_* 表中的场景基础配置，并返回最新快照。
func (s *Service) UpdateSceneConfig(ctx context.Context, in UpdateSceneConfigInput) (Snapshot, error) {
	if err := validateSceneConfig(in); err != nil {
		return Snapshot{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Snapshot{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	name := strings.TrimSpace(in.Name)
	res, err := tx.ExecContext(ctx, `UPDATE system_scenes SET name = $1 WHERE id = $2`, name, in.SceneID)
	if err != nil {
		return Snapshot{}, err
	}
	if rows, errRows := res.RowsAffected(); errRows == nil && rows == 0 {
		return Snapshot{}, fmt.Errorf("%w: scene %s not found", ErrInvalidSceneConfig, in.SceneID)
	}

	if _, err = tx.ExecContext(ctx, `
		INSERT INTO system_scene_grid (scene_id, cols, rows, tile_size)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (scene_id)
		DO UPDATE SET cols = EXCLUDED.cols, rows = EXCLUDED.rows, tile_size = EXCLUDED.tile_size
	`, in.SceneID, in.Grid.Cols, in.Grid.Rows, in.Grid.TileSize); err != nil {
		return Snapshot{}, err
	}

	if _, err = tx.ExecContext(ctx, `
		INSERT INTO system_scene_dimensions (scene_id, width, height)
		VALUES ($1, $2, $3)
		ON CONFLICT (scene_id)
		DO UPDATE SET width = EXCLUDED.width, height = EXCLUDED.height
	`, in.SceneID, in.Dimensions.Width, in.Dimensions.Height); err != nil {
		return Snapshot{}, err
	}

	if err = tx.Commit(); err != nil {
		return Snapshot{}, err
	}

	updated, err := sceneLoader(s.db, in.SceneID)
	if err != nil {
		return Snapshot{}, err
	}
	s.scene = updated
	return s.Snapshot(), nil
}

func validateSceneConfig(in UpdateSceneConfigInput) error {
	if strings.TrimSpace(in.SceneID) == "" {
		return fmt.Errorf("%w: scene_id required", ErrInvalidSceneConfig)
	}
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("%w: name required", ErrInvalidSceneConfig)
	}
	if in.Grid.Cols <= 0 {
		return fmt.Errorf("%w: grid.cols must be positive", ErrInvalidSceneConfig)
	}
	if in.Grid.Rows <= 0 {
		return fmt.Errorf("%w: grid.rows must be positive", ErrInvalidSceneConfig)
	}
	if in.Grid.TileSize <= 0 {
		return fmt.Errorf("%w: grid.tileSize must be positive", ErrInvalidSceneConfig)
	}
	if in.Dimensions.Width <= 0 {
		return fmt.Errorf("%w: dimensions.width must be positive", ErrInvalidSceneConfig)
	}
	if in.Dimensions.Height <= 0 {
		return fmt.Errorf("%w: dimensions.height must be positive", ErrInvalidSceneConfig)
	}
	return nil
}

// UpdateBuildingTemplate 更新或创建系统建筑模板。
func (s *Service) UpdateBuildingTemplate(ctx context.Context, in UpdateBuildingTemplateInput) (Snapshot, error) {
	if strings.TrimSpace(in.ID) == "" {
		return Snapshot{}, fmt.Errorf("%w: id required", ErrInvalidTemplate)
	}
	if strings.TrimSpace(in.Label) == "" {
		return Snapshot{}, fmt.Errorf("%w: label required", ErrInvalidTemplate)
	}

	energyType, capacity, current, output, rate := extractTemplateEnergy(in.Energy)
	if energyType.Valid {
		normalized := strings.ToLower(energyType.String)
		if normalized != "storage" && normalized != "consumer" {
			return Snapshot{}, fmt.Errorf("%w: energy.type must be storage or consumer", ErrInvalidTemplate)
		}
		energyType.String = normalized
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO system_template_buildings (id, label, energy_type, energy_capacity, energy_current, energy_output, energy_rate)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id)
		DO UPDATE SET label = EXCLUDED.label,
		              energy_type = EXCLUDED.energy_type,
		              energy_capacity = EXCLUDED.energy_capacity,
		              energy_current = EXCLUDED.energy_current,
		              energy_output = EXCLUDED.energy_output,
		              energy_rate = EXCLUDED.energy_rate
	`, strings.TrimSpace(in.ID), strings.TrimSpace(in.Label), energyType, capacity, current, output, rate); err != nil {
		return Snapshot{}, err
	}

	if err := s.reloadScene(); err != nil {
		return Snapshot{}, err
	}

	return s.Snapshot(), nil
}

// UpdateAgentTemplate 更新或创建系统 Agent 模板。
func (s *Service) UpdateAgentTemplate(ctx context.Context, in UpdateAgentTemplateInput) (Snapshot, error) {
	if strings.TrimSpace(in.ID) == "" {
		return Snapshot{}, fmt.Errorf("%w: id required", ErrInvalidTemplate)
	}
	if strings.TrimSpace(in.Label) == "" {
		return Snapshot{}, fmt.Errorf("%w: label required", ErrInvalidTemplate)
	}

	posX := sql.NullInt64{}
	posY := sql.NullInt64{}
	if in.Position != nil {
		coords := *in.Position
		posX = sql.NullInt64{Int64: int64(coords[0]), Valid: true}
		posY = sql.NullInt64{Int64: int64(coords[1]), Valid: true}
	}

	color := nullInt64(in.Color)

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO system_template_agents (id, label, color, default_position_x, default_position_y)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id)
		DO UPDATE SET label = EXCLUDED.label,
		              color = EXCLUDED.color,
		              default_position_x = EXCLUDED.default_position_x,
		              default_position_y = EXCLUDED.default_position_y
	`, strings.TrimSpace(in.ID), strings.TrimSpace(in.Label), color, posX, posY); err != nil {
		return Snapshot{}, err
	}

	if err := s.reloadScene(); err != nil {
		return Snapshot{}, err
	}

	return s.Snapshot(), nil
}

// UpdateSceneBuilding 更新或创建场景中的建筑实例。
func (s *Service) UpdateSceneBuilding(ctx context.Context, in UpdateSceneBuildingInput) (Snapshot, error) {
	if strings.TrimSpace(in.ID) == "" {
		return Snapshot{}, fmt.Errorf("%w: id required", ErrInvalidSceneEntity)
	}
	if strings.TrimSpace(in.Label) == "" {
		return Snapshot{}, fmt.Errorf("%w: label required", ErrInvalidSceneEntity)
	}
	if in.Rect[2] <= 0 || in.Rect[3] <= 0 {
		return Snapshot{}, fmt.Errorf("%w: width/height must be positive", ErrInvalidSceneEntity)
	}

	templateID := nullTrimmedString(in.TemplateID)
	energyType, capacity, current, output, rate := extractTemplateEnergy(in.Energy)
	if energyType.Valid {
		normalized := strings.ToLower(energyType.String)
		if normalized != "storage" && normalized != "consumer" {
			return Snapshot{}, fmt.Errorf("%w: energy.type must be storage or consumer", ErrInvalidSceneEntity)
		}
		energyType.String = normalized
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO system_scene_buildings (id, scene_id, template_id, label, position_x, position_y, size_width, size_height, energy_type, energy_capacity, energy_current, energy_output, energy_rate)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (id)
		DO UPDATE SET template_id = EXCLUDED.template_id,
		              label = EXCLUDED.label,
		              position_x = EXCLUDED.position_x,
		              position_y = EXCLUDED.position_y,
		              size_width = EXCLUDED.size_width,
		              size_height = EXCLUDED.size_height,
		              energy_type = EXCLUDED.energy_type,
		              energy_capacity = EXCLUDED.energy_capacity,
		              energy_current = EXCLUDED.energy_current,
		              energy_output = EXCLUDED.energy_output,
		              energy_rate = EXCLUDED.energy_rate
	`, strings.TrimSpace(in.ID), s.scene.ID, templateID, strings.TrimSpace(in.Label), in.Rect[0], in.Rect[1], in.Rect[2], in.Rect[3], energyType, capacity, current, output, rate); err != nil {
		return Snapshot{}, err
	}

	if err := s.reloadScene(); err != nil {
		return Snapshot{}, err
	}

	return s.Snapshot(), nil
}

// UpdateSceneAgent 更新或创建场景中的 Agent 实例以及动作列表。
func (s *Service) UpdateSceneAgent(ctx context.Context, in UpdateSceneAgentInput) (Snapshot, error) {
	if strings.TrimSpace(in.ID) == "" {
		return Snapshot{}, fmt.Errorf("%w: id required", ErrInvalidSceneEntity)
	}
	if strings.TrimSpace(in.Label) == "" {
		return Snapshot{}, fmt.Errorf("%w: label required", ErrInvalidSceneEntity)
	}

	templateID := nullTrimmedString(in.TemplateID)
	color := nullInt64(in.Color)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Snapshot{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, `
		INSERT INTO system_scene_agents (id, scene_id, template_id, label, position_x, position_y, color)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id)
		DO UPDATE SET template_id = EXCLUDED.template_id,
		              label = EXCLUDED.label,
		              position_x = EXCLUDED.position_x,
		              position_y = EXCLUDED.position_y,
		              color = EXCLUDED.color
	`, strings.TrimSpace(in.ID), s.scene.ID, templateID, strings.TrimSpace(in.Label), in.Position[0], in.Position[1], color); err != nil {
		return Snapshot{}, err
	}

	if _, err = tx.ExecContext(ctx, `DELETE FROM system_scene_agent_actions WHERE agent_id = $1`, strings.TrimSpace(in.ID)); err != nil {
		return Snapshot{}, err
	}

	if len(in.Actions) > 0 {
		for _, action := range in.Actions {
			trimmed := strings.TrimSpace(action)
			if trimmed == "" {
				continue
			}
			if _, err = tx.ExecContext(ctx, `INSERT INTO system_scene_agent_actions (agent_id, action) VALUES ($1, $2) ON CONFLICT (agent_id, action) DO NOTHING`, strings.TrimSpace(in.ID), trimmed); err != nil {
				return Snapshot{}, err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return Snapshot{}, err
	}

	if err := s.reloadScene(); err != nil {
		return Snapshot{}, err
	}

	return s.Snapshot(), nil
}

// UpdateAgentRuntimePosition 更新运行时 Agent 坐标。
func (s *Service) UpdateAgentRuntimePosition(ctx context.Context, agentID string, posX, posY float64) (SceneAgent, error) {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return SceneAgent{}, fmt.Errorf("%w: agent id required", ErrInvalidSceneEntity)
	}

	if _, err := s.db.ExecContext(ctx, `
        INSERT INTO agent_runtime_state (agent_id, pos_x, pos_y)
        VALUES ($1, $2, $3)
        ON CONFLICT (agent_id)
        DO UPDATE SET pos_x = EXCLUDED.pos_x,
                      pos_y = EXCLUDED.pos_y,
                      updated_at = NOW()
    `, agentID, posX, posY); err != nil {
		return SceneAgent{}, err
	}

	if err := s.reloadScene(); err != nil {
		return SceneAgent{}, err
	}

	for _, agent := range s.scene.Agents {
		if agent.ID == agentID {
			return agent, nil
		}
	}

	return SceneAgent{}, fmt.Errorf("%w: agent %s not found", ErrInvalidSceneEntity, agentID)
}

func extractTemplateEnergy(in *UpdateTemplateEnergyInput) (sql.NullString, sql.NullInt64, sql.NullInt64, sql.NullInt64, sql.NullInt64) {
	if in == nil {
		return sql.NullString{}, sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}, sql.NullInt64{}
	}
	return nullTrimmedString(in.Type), nullInt64(in.Capacity), nullInt64(in.Current), nullInt64(in.Output), nullInt64(in.Rate)
}

func nullTrimmedString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: trimmed, Valid: true}
}

func nullInt64(i *int) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*i), Valid: true}
}

func (s *Service) reloadScene() error {
	updated, err := sceneLoader(s.db, s.scene.ID)
	if err != nil {
		return err
	}
	s.scene = updated
	return nil
}

// UpdateBuildingEnergyCurrent 更新指定建筑的当前能量值，并返回更新后的建筑信息。
func (s *Service) UpdateBuildingEnergyCurrent(ctx context.Context, buildingID string, currentValue float64) (SceneBuilding, error) {
	buildingID = strings.TrimSpace(buildingID)
	if buildingID == "" {
		return SceneBuilding{}, fmt.Errorf("%w: building id required", ErrInvalidSceneEntity)
	}

	currentInt := int(math.Round(currentValue))
	if currentInt < 0 {
		currentInt = 0
	}

	res, err := s.db.ExecContext(ctx, `
		UPDATE system_scene_buildings
		   SET energy_current = $1
		 WHERE id = $2
	`, currentInt, buildingID)
	if err != nil {
		return SceneBuilding{}, err
	}
	if rows, errRows := res.RowsAffected(); errRows == nil && rows == 0 {
		return SceneBuilding{}, fmt.Errorf("%w: building %s not found", ErrInvalidSceneEntity, buildingID)
	}

	if err := s.reloadScene(); err != nil {
		return SceneBuilding{}, err
	}

	for _, building := range s.scene.Buildings {
		if building.ID == buildingID {
			if building.Energy != nil && building.Energy.Type == "storage" {
				building.Energy.Current = currentInt
			}
			return building, nil
		}
	}

	return SceneBuilding{}, fmt.Errorf("%w: building %s not found after update", ErrInvalidSceneEntity, buildingID)
}

type energyBalance struct {
	consumption float64
	output      float64
	storage     []SceneBuilding
}

func computeEnergyBalance(scene Scene) energyBalance {
	var balance energyBalance
	for _, building := range scene.Buildings {
		if building.Energy == nil {
			continue
		}
		switch strings.ToLower(building.Energy.Type) {
		case "consumer":
			balance.consumption += float64(building.Energy.Rate)
		case "storage":
			balance.output += float64(building.Energy.Output)
			balance.storage = append(balance.storage, building)
		}
	}
	return balance
}

// AdvanceEnergyState 根据耗能计算更新储能节点的剩余能量。
func (s *Service) AdvanceEnergyState(ctx context.Context, seconds float64, drainFactor float64) (Scene, error) {
	if seconds <= 0 {
		seconds = 1
	}
	if drainFactor <= 0 {
		drainFactor = DefaultDrainFactor
	}

	balance := computeEnergyBalance(s.scene)
	totalConsumption := balance.consumption
	totalOutput := balance.output
	storageBuildings := balance.storage

	if len(storageBuildings) == 0 {
		return s.scene, nil
	}

	netLoad := totalConsumption - totalOutput
	if netLoad == 0 {
		return s.scene, nil
	}

	change := netLoad * drainFactor * seconds
	if change == 0 {
		return s.scene, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Scene{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if netLoad > 0 {
		for _, building := range storageBuildings {
			current := float64(building.Energy.Current)
			updated := int(math.Max(math.Round(current-change), 0))
			if updated == building.Energy.Current {
				continue
			}
			if _, err = tx.ExecContext(ctx, `UPDATE system_scene_buildings SET energy_current = $1 WHERE id = $2`, updated, building.ID); err != nil {
				return Scene{}, err
			}
		}
	} else { // netLoad < 0, surplus energy
		gain := -change
		for _, building := range storageBuildings {
			capacity := float64(building.Energy.Capacity)
			if capacity <= 0 {
				continue
			}
			current := float64(building.Energy.Current)
			updated := int(math.Min(math.Round(current+gain), capacity))
			if updated == building.Energy.Current {
				continue
			}
			if _, err = tx.ExecContext(ctx, `UPDATE system_scene_buildings SET energy_current = $1 WHERE id = $2`, updated, building.ID); err != nil {
				return Scene{}, err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return Scene{}, err
	}

	if err := s.reloadScene(); err != nil {
		return Scene{}, err
	}

	return s.scene, nil
}

// MaintainEnergyNonNegative 根据当前能耗情况自动补充太阳能塔直至净变化不再为负。
func (s *Service) MaintainEnergyNonNegative(ctx context.Context, agentID string) (MaintainEnergyResult, error) {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return MaintainEnergyResult{}, fmt.Errorf("%w: agent id required", ErrInvalidSceneEntity)
	}

	balance := computeEnergyBalance(s.scene)
	netFlowBefore := balance.output - balance.consumption
	result := MaintainEnergyResult{
		Scene:         s.scene,
		NetFlowBefore: netFlowBefore,
		NetFlowAfter:  netFlowBefore,
	}

	if netFlowBefore >= 0 {
		return result, nil
	}

	template, ok := s.findSolarTowerTemplate()
	if !ok || template.Energy == nil || template.Energy.Output <= 0 {
		return MaintainEnergyResult{}, ErrSolarTemplateMissing
	}

	towerOutput := float64(template.Energy.Output)
	if towerOutput <= 0 {
		return MaintainEnergyResult{}, ErrSolarTemplateMissing
	}

	deficit := balance.consumption - balance.output
	if deficit <= 0 {
		return result, nil
	}

	towersNeeded := int(math.Ceil(deficit / towerOutput))
	if towersNeeded <= 0 {
		return result, nil
	}

	width, height := determineSolarTowerFootprint(s.scene.Buildings)
	if width <= 0 {
		width = 4
	}
	if height <= 0 {
		height = 4
	}

	occupied := append([]SceneBuilding(nil), s.scene.Buildings...)
	nextIndex := nextSolarTowerIndex(occupied)
	planned := make([]plannedTower, 0, towersNeeded)

	for len(planned) < towersNeeded {
		x, y, ok := findAvailablePlacement(occupied, s.scene.Dimensions, width, height)
		if !ok {
			return MaintainEnergyResult{}, ErrNoAvailablePlacement
		}
		nextIndex++
		id := fmt.Sprintf("solar_tower_auto_%02d", nextIndex)
		label := fmt.Sprintf("太阳能塔 自动 %02d", nextIndex)
		planned = append(planned, plannedTower{
			id:     id,
			label:  label,
			x:      x,
			y:      y,
			width:  width,
			height: height,
		})
		occupied = append(occupied, SceneBuilding{ID: id, TemplateID: solarTowerTemplateID, Rect: []int{x, y, width, height}})
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return MaintainEnergyResult{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	for _, tower := range planned {
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO system_scene_buildings (id, scene_id, template_id, label, position_x, position_y, size_width, size_height)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (id)
			DO UPDATE SET scene_id = EXCLUDED.scene_id,
			              template_id = EXCLUDED.template_id,
			              label = EXCLUDED.label,
			              position_x = EXCLUDED.position_x,
			              position_y = EXCLUDED.position_y,
			              size_width = EXCLUDED.size_width,
			              size_height = EXCLUDED.size_height
		`, tower.id, s.scene.ID, solarTowerTemplateID, tower.label, tower.x, tower.y, tower.width, tower.height); err != nil {
			return MaintainEnergyResult{}, err
		}
	}

	if err = tx.Commit(); err != nil {
		return MaintainEnergyResult{}, err
	}

	if err := s.reloadScene(); err != nil {
		return MaintainEnergyResult{}, err
	}

	created := make([]SceneBuilding, 0, len(planned))
	createdIDs := make(map[string]struct{}, len(planned))
	for _, tower := range planned {
		createdIDs[tower.id] = struct{}{}
	}
	for _, building := range s.scene.Buildings {
		if _, ok := createdIDs[building.ID]; ok {
			created = append(created, building)
		}
	}

	balanceAfter := computeEnergyBalance(s.scene)
	result.Scene = s.scene
	result.Created = created
	result.NetFlowAfter = balanceAfter.output - balanceAfter.consumption
	result.TowersBuilt = len(created)

	return result, nil
}

func (s *Service) findSolarTowerTemplate() (*BuildingTemplate, bool) {
	for i := range s.scene.BuildingTemplates {
		if s.scene.BuildingTemplates[i].ID == solarTowerTemplateID {
			return &s.scene.BuildingTemplates[i], true
		}
	}
	return nil, false
}

type plannedTower struct {
	id     string
	label  string
	x      int
	y      int
	width  int
	height int
}

func determineSolarTowerFootprint(buildings []SceneBuilding) (int, int) {
	for _, building := range buildings {
		if building.TemplateID == solarTowerTemplateID || strings.HasPrefix(building.ID, "solar_tower") {
			if len(building.Rect) == 4 {
				return building.Rect[2], building.Rect[3]
			}
		}
	}
	return 4, 4
}

func nextSolarTowerIndex(buildings []SceneBuilding) int {
	maxIndex := 0
	for _, building := range buildings {
		if !strings.HasPrefix(building.ID, "solar_tower") {
			continue
		}
		parts := strings.Split(building.ID, "_")
		if len(parts) == 0 {
			continue
		}
		last := parts[len(parts)-1]
		value, err := strconv.Atoi(strings.TrimLeft(last, "0"))
		if err != nil {
			value, err = strconv.Atoi(last)
		}
		if err == nil && value > maxIndex {
			maxIndex = value
		}
	}
	return maxIndex
}

func findAvailablePlacement(buildings []SceneBuilding, dims SceneDims, width, height int) (int, int, bool) {
	if width <= 0 || height <= 0 {
		return 0, 0, false
	}
	maxX := dims.Width - width
	maxY := dims.Height - height
	if maxX < 0 || maxY < 0 {
		return 0, 0, false
	}
	for y := 0; y <= maxY; y++ {
		for x := 0; x <= maxX; x++ {
			if areaIsFree(buildings, x, y, width, height) {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

func areaIsFree(buildings []SceneBuilding, x, y, width, height int) bool {
	for _, building := range buildings {
		if len(building.Rect) != 4 {
			continue
		}
		bx, by, bw, bh := building.Rect[0], building.Rect[1], building.Rect[2], building.Rect[3]
		if rectanglesOverlap(x, y, width, height, bx, by, bw, bh) {
			return false
		}
	}
	return true
}

func rectanglesOverlap(ax, ay, aw, ah, bx, by, bw, bh int) bool {
	return ax < bx+bw && ax+aw > bx && ay < by+bh && ay+ah > by
}
