package game

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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
	Scene         Scene            `json:"scene"`
	Created       []SceneBuilding  `json:"created"`
	NetFlowBefore float64          `json:"netFlowBefore"`
	NetFlowAfter  float64          `json:"netFlowAfter"`
	TowersBuilt   int              `json:"towersBuilt"`
	Relocation    *AgentRelocation `json:"relocation,omitempty"`
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

type AgentRelocation struct {
	ID       string     `json:"id"`
	Position [2]float64 `json:"position"`
}

func (m *EnergyMaintainer) Maintain(ctx context.Context, scene Scene, agent SceneAgent) (MaintainEnergyResult, Scene, *AgentRelocation, error) {
	balance := computeEnergyBalance(scene)
	netFlow := balance.output - balance.consumption

	result := MaintainEnergyResult{
		Scene:         scene,
		NetFlowBefore: netFlow,
		NetFlowAfter:  netFlow,
	}

	if netFlow >= 0 {
		return result, scene, nil, nil
	}

	template, ok := findSolarTemplate(scene)
	if !ok || template.Energy == nil || template.Energy.Output <= 0 {
		return MaintainEnergyResult{}, Scene{}, nil, ErrSolarTemplateMissing
	}

	towerOutput := float64(template.Energy.Output)
	deficit := balance.consumption - balance.output
	if deficit <= 0 {
		return result, scene, nil, nil
	}

	towersNeeded := int(math.Ceil(deficit / towerOutput))
	if towersNeeded <= 0 {
		return result, scene, nil, nil
	}

	log.Printf("EnergyMaintainer: start agent=%s netFlow=%.2f deficit=%.2f towers=%d", agent.ID, netFlow, deficit, towersNeeded)

	width, height := determineSolarTowerFootprint(scene.Buildings)
	baseOccupied := append([]SceneBuilding(nil), scene.Buildings...)
	occupied := append([]SceneBuilding(nil), baseOccupied...)
	baseIndex := nextSolarTowerIndex(baseOccupied)
	nextIndex := baseIndex
	planned := make([]plannedTower, 0, towersNeeded)

	agentTile := clampTile(agent.Position, scene.Dimensions)
	currentTile := agentTile
	var relocation *AgentRelocation
	visitedTiles := map[[2]int]struct{}{
		agentTile: struct{}{},
	}

	for len(planned) < towersNeeded {
		x, y, ok := findAdjacentPlacementForAgent(currentTile, width, height, occupied, scene.Dimensions)
		if !ok {
			tile, placement, found := findRelocationAndPlacement(currentTile, width, height, occupied, scene.Dimensions)
			if !found {
				log.Printf("EnergyMaintainer: no placement available agent=%s built=%d/%d", agent.ID, len(planned), towersNeeded)
				return MaintainEnergyResult{}, Scene{}, relocation, ErrNoAvailablePlacement
			}
			if _, seen := visitedTiles[tile]; seen {
				log.Printf("EnergyMaintainer: relocation revisited agent=%s tile=(%d,%d)", agent.ID, tile[0], tile[1])
				return MaintainEnergyResult{}, Scene{}, relocation, ErrNoAvailablePlacement
			}
			visitedTiles[tile] = struct{}{}
			currentTile = tile
			if relocation == nil || relocation.Position[0] != float64(tile[0]) || relocation.Position[1] != float64(tile[1]) {
				relocation = &AgentRelocation{ID: agent.ID, Position: [2]float64{float64(tile[0]), float64(tile[1])}}
				log.Printf("EnergyMaintainer: relocation agent=%s to (%d,%d)", agent.ID, tile[0], tile[1])
			}
			x, y = placement[0], placement[1]
		}

		if !areaIsFree(occupied, x, y, width, height) {
			log.Printf("EnergyMaintainer: placement blocked agent=%s at (%d,%d)", agent.ID, x, y)
			return MaintainEnergyResult{}, Scene{}, relocation, ErrNoAvailablePlacement
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
		log.Printf("EnergyMaintainer: planned tower %d/%d at (%d,%d)", len(planned), towersNeeded, x, y)
	}

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return MaintainEnergyResult{}, Scene{}, relocation, err
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
			return MaintainEnergyResult{}, Scene{}, relocation, err
		}
	}

	if err = tx.Commit(); err != nil {
		return MaintainEnergyResult{}, Scene{}, relocation, err
	}

	updatedScene, err := m.loadScene(m.db, scene.ID)
	if err != nil {
		return MaintainEnergyResult{}, Scene{}, relocation, err
	}

	created := collectCreatedBuildings(updatedScene.Buildings, planned)
	balanceAfter := computeEnergyBalance(updatedScene)
	result.Scene = updatedScene
	result.Created = created
	result.NetFlowAfter = balanceAfter.output - balanceAfter.consumption
	result.TowersBuilt = len(created)
	result.Relocation = relocation

	log.Printf("EnergyMaintainer: success agent=%s towersBuilt=%d relocation=%v", agent.ID, len(planned), relocation != nil)

	return result, updatedScene, relocation, nil
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

func clampTile(position []float64, dims SceneDims) [2]int {
	var rawX, rawY float64
	if len(position) >= 1 {
		rawX = position[0]
	}
	if len(position) >= 2 {
		rawY = position[1]
	}
	x := int(math.Floor(rawX))
	y := int(math.Floor(rawY))
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if dims.Width > 0 && x >= dims.Width {
		x = dims.Width - 1
	}
	if dims.Height > 0 && y >= dims.Height {
		y = dims.Height - 1
	}
	return [2]int{x, y}
}

func findAdjacentPlacementForAgent(agentTile [2]int, width, height int, occupied []SceneBuilding, dims SceneDims) (int, int, bool) {
	candidates := [][2]int{
		{agentTile[0] - width, agentTile[1]},  // left
		{agentTile[0] + 1, agentTile[1]},      // right
		{agentTile[0], agentTile[1] - height}, // top
		{agentTile[0], agentTile[1] + 1},      // bottom
	}

	for _, candidate := range candidates {
		x, y := candidate[0], candidate[1]
		if x < 0 || y < 0 {
			continue
		}
		if dims.Width > 0 && x+width > dims.Width {
			continue
		}
		if dims.Height > 0 && y+height > dims.Height {
			continue
		}
		if areaIsFree(occupied, x, y, width, height) {
			return x, y, true
		}
	}

	return 0, 0, false
}

type point struct {
	x int
	y int
}

func findRelocationAndPlacement(start [2]int, width, height int, occupied []SceneBuilding, dims SceneDims) ([2]int, [2]int, bool) {
	if dims.Width <= 0 || dims.Height <= 0 {
		return [2]int{}, [2]int{}, false
	}

	visited := make([][]bool, dims.Width)
	for i := range visited {
		visited[i] = make([]bool, dims.Height)
	}

	queue := make([]point, 0, dims.Width*dims.Height)
	head := 0
	queue = append(queue, point{start[0], start[1]})
	if start[0] >= 0 && start[0] < dims.Width && start[1] >= 0 && start[1] < dims.Height {
		visited[start[0]][start[1]] = true
	}

	for head < len(queue) {
		cur := queue[head]
		head++

		if cur.x < 0 || cur.x >= dims.Width || cur.y < 0 || cur.y >= dims.Height {
			continue
		}

		if !tileIsBlocked(cur.x, cur.y, occupied) {
			if px, py, ok := findAdjacentPlacementForAgent([2]int{cur.x, cur.y}, width, height, occupied, dims); ok {
				return [2]int{cur.x, cur.y}, [2]int{px, py}, true
			}
		}

		for _, nb := range neighbors4(cur) {
			if nb.x < 0 || nb.x >= dims.Width || nb.y < 0 || nb.y >= dims.Height {
				continue
			}
			if visited[nb.x][nb.y] {
				continue
			}
			visited[nb.x][nb.y] = true
			queue = append(queue, nb)
		}
	}

	return [2]int{}, [2]int{}, false
}

func tileIsBlocked(x, y int, buildings []SceneBuilding) bool {
	for _, building := range buildings {
		if len(building.Rect) != 4 {
			continue
		}
		bx, by, bw, bh := building.Rect[0], building.Rect[1], building.Rect[2], building.Rect[3]
		if x >= bx && x < bx+bw && y >= by && y < by+bh {
			return true
		}
	}
	return false
}

func neighbors4(p point) []point {
	return []point{
		{p.x - 1, p.y},
		{p.x + 1, p.y},
		{p.x, p.y - 1},
		{p.x, p.y + 1},
	}
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
