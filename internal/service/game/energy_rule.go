package game

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type energyBalance struct {
	consumption float64
	output      float64
	storage     []SceneBuilding
}

// MaintainEnergyResult 描述“保持电量不减少”指令的执行结果。
type MaintainEnergyResult struct {
	Scene         Scene
	Created       []SceneBuilding
	NetFlowBefore float64
	NetFlowAfter  float64
	TowersBuilt   int
}

// EnergyMaintainer 负责实现能量守恒相关规则。
type EnergyMaintainer struct {
	db        *sql.DB
	loadScene func(*sql.DB, string) (Scene, error)
}

func newEnergyMaintainer(db *sql.DB, loader func(*sql.DB, string) (Scene, error)) *EnergyMaintainer {
	if loader == nil {
		loader = sceneLoader
	}
	return &EnergyMaintainer{db: db, loadScene: loader}
}

func (m *EnergyMaintainer) Maintain(ctx context.Context, scene Scene) (MaintainEnergyResult, Scene, error) {
	balance := computeEnergyBalance(scene)
	netFlow := balance.output - balance.consumption

	result := MaintainEnergyResult{
		Scene:         scene,
		NetFlowBefore: netFlow,
		NetFlowAfter:  netFlow,
	}

	if netFlow >= 0 {
		return result, scene, nil
	}

	template, ok := findSolarTemplate(scene)
	if !ok || template.Energy == nil || template.Energy.Output <= 0 {
		return MaintainEnergyResult{}, Scene{}, ErrSolarTemplateMissing
	}

	towerOutput := float64(template.Energy.Output)
	deficit := balance.consumption - balance.output
	if deficit <= 0 {
		return result, scene, nil
	}

	towersNeeded := int(math.Ceil(deficit / towerOutput))
	if towersNeeded <= 0 {
		return result, scene, nil
	}

	width, height := determineSolarTowerFootprint(scene.Buildings)
	occupied := append([]SceneBuilding(nil), scene.Buildings...)
	nextIndex := nextSolarTowerIndex(occupied)
	planned := make([]plannedTower, 0, towersNeeded)

	for len(planned) < towersNeeded {
		x, y, ok := findAvailablePlacement(occupied, scene.Dimensions, width, height)
		if !ok {
			return MaintainEnergyResult{}, Scene{}, ErrNoAvailablePlacement
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

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return MaintainEnergyResult{}, Scene{}, err
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
        `, tower.id, scene.ID, solarTowerTemplateID, tower.label, tower.x, tower.y, tower.width, tower.height); err != nil {
			return MaintainEnergyResult{}, Scene{}, err
		}
	}

	if err = tx.Commit(); err != nil {
		return MaintainEnergyResult{}, Scene{}, err
	}

	updatedScene, err := m.loadScene(m.db, scene.ID)
	if err != nil {
		return MaintainEnergyResult{}, Scene{}, err
	}

	created := collectCreatedBuildings(updatedScene.Buildings, planned)
	balanceAfter := computeEnergyBalance(updatedScene)
	result.Scene = updatedScene
	result.Created = created
	result.NetFlowAfter = balanceAfter.output - balanceAfter.consumption
	result.TowersBuilt = len(created)

	return result, updatedScene, nil
}

func findSolarTemplate(scene Scene) (*BuildingTemplate, bool) {
	for i := range scene.BuildingTemplates {
		tpl := &scene.BuildingTemplates[i]
		if tpl.ID == solarTowerTemplateID {
			return tpl, true
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

func collectCreatedBuildings(buildings []SceneBuilding, planned []plannedTower) []SceneBuilding {
	created := make([]SceneBuilding, 0, len(planned))
	if len(planned) == 0 {
		return created
	}

	plannedIDs := make(map[string]struct{}, len(planned))
	for _, p := range planned {
		plannedIDs[p.id] = struct{}{}
	}

	for _, building := range buildings {
		if _, ok := plannedIDs[building.ID]; ok {
			created = append(created, building)
		}
	}
	return created
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
