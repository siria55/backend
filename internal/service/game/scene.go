package game

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Scene 表示火星场景的静态配置。
type Scene struct {
	ID                string             `json:"id"`
	Name              string             `json:"name"`
	Grid              SceneGrid          `json:"grid"`
	Dimensions        SceneDims          `json:"dimensions"`
	Buildings         []SceneBuilding    `json:"buildings"`
	Agents            []SceneAgent       `json:"agents"`
	BuildingTemplates []BuildingTemplate `json:"buildingTemplates"`
	AgentTemplates    []AgentTemplate    `json:"agentTemplates"`
}

// SceneGrid 描述场景网格大小与基本单元尺寸。
type SceneGrid struct {
	Cols     int `json:"cols"`
	Rows     int `json:"rows"`
	TileSize int `json:"tileSize"`
}

// SceneDims 描述场景逻辑宽高。
type SceneDims struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// SceneEnergy 描述建筑的能量属性。
type SceneEnergy struct {
	Type     string `json:"type"`
	Capacity int    `json:"capacity,omitempty"`
	Current  int    `json:"current,omitempty"`
	Output   int    `json:"output,omitempty"`
	Rate     int    `json:"rate,omitempty"`
}

// SceneBuilding 描述场景中的建筑。
type SceneBuilding struct {
	ID         string       `json:"id"`
	TemplateID string       `json:"templateId,omitempty"`
	Label      string       `json:"label"`
	Rect       []int        `json:"rect"`
	Energy     *SceneEnergy `json:"energy,omitempty"`
}

// SceneAgent 描述场景中的角色。
type SceneAgent struct {
	ID         string    `json:"id"`
	TemplateID string    `json:"templateId,omitempty"`
	Label      string    `json:"label"`
	Position   []float64 `json:"position"`
	Color      int       `json:"color,omitempty"`
	Actions    []string  `json:"actions,omitempty"`
}

// Snapshot 表示 system_* 表的整合视图。
type Snapshot struct {
	Scene             SceneMeta          `json:"scene"`
	Grid              SceneGrid          `json:"grid"`
	Dimensions        SceneDims          `json:"dimensions"`
	Buildings         []SceneBuilding    `json:"buildings"`
	Agents            []SceneAgent       `json:"agents"`
	BuildingTemplates []BuildingTemplate `json:"buildingTemplates"`
	AgentTemplates    []AgentTemplate    `json:"agentTemplates"`
}

// SceneMeta 描述场景的基本信息。
type SceneMeta struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// BuildingTemplate 描述系统建筑模板。
type BuildingTemplate struct {
	ID     string       `json:"id"`
	Label  string       `json:"label"`
	Energy *SceneEnergy `json:"energy,omitempty"`
}

type AgentTemplate struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Color    int    `json:"color,omitempty"`
	Position []int  `json:"position,omitempty"`
}

// UpdateSceneConfigInput 表示更新 system_* 场景配置所需的数据。
type UpdateSceneConfigInput struct {
	SceneID    string
	Name       string
	Grid       SceneGrid
	Dimensions SceneDims
}

type UpdateTemplateEnergyInput struct {
	Type     *string
	Capacity *int
	Current  *int
	Output   *int
	Rate     *int
}

type UpdateBuildingTemplateInput struct {
	ID     string
	Label  string
	Energy *UpdateTemplateEnergyInput
}

type UpdateAgentTemplateInput struct {
	ID       string
	Label    string
	Color    *int
	Position *[2]int
}

type UpdateSceneBuildingInput struct {
	ID         string
	Label      string
	TemplateID *string
	Rect       [4]int
	Energy     *UpdateTemplateEnergyInput
}

type UpdateSceneAgentInput struct {
	ID         string
	Label      string
	TemplateID *string
	Position   [2]int
	Color      *int
	Actions    []string
}

func loadSceneFromStore(db *sql.DB, sceneID string) (Scene, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var scene Scene

	if err := db.QueryRowContext(ctx, `SELECT id, name FROM system_scenes WHERE id = $1`, sceneID).
		Scan(&scene.ID, &scene.Name); err != nil {
		if err == sql.ErrNoRows {
			return Scene{}, fmt.Errorf("scene %s not found", sceneID)
		}
		return Scene{}, err
	}

	if err := db.QueryRowContext(ctx, `SELECT cols, rows, tile_size FROM system_scene_grid WHERE scene_id = $1`, sceneID).
		Scan(&scene.Grid.Cols, &scene.Grid.Rows, &scene.Grid.TileSize); err != nil {
		if err == sql.ErrNoRows {
			return Scene{}, fmt.Errorf("grid configuration missing for scene %s", sceneID)
		}
		return Scene{}, err
	}

	if err := db.QueryRowContext(ctx, `SELECT width, height FROM system_scene_dimensions WHERE scene_id = $1`, sceneID).
		Scan(&scene.Dimensions.Width, &scene.Dimensions.Height); err != nil {
		if err == sql.ErrNoRows {
			return Scene{}, fmt.Errorf("dimensions configuration missing for scene %s", sceneID)
		}
		return Scene{}, err
	}

	buildingRows, err := db.QueryContext(ctx, `
        SELECT b.id,
               b.template_id,
               COALESCE(b.label, t.label) AS label,
               b.position_x,
               b.position_y,
               b.size_width,
               b.size_height,
               COALESCE(b.energy_type, t.energy_type) AS energy_type,
               COALESCE(b.energy_capacity, t.energy_capacity) AS energy_capacity,
               COALESCE(b.energy_current, t.energy_current) AS energy_current,
               COALESCE(b.energy_output, t.energy_output) AS energy_output,
               COALESCE(b.energy_rate, t.energy_rate) AS energy_rate
          FROM system_scene_buildings b
          LEFT JOIN system_template_buildings t ON t.id = b.template_id
         WHERE b.scene_id = $1
         ORDER BY b.id
    `, sceneID)
	if err != nil {
		return Scene{}, err
	}
	defer buildingRows.Close()

	for buildingRows.Next() {
		var (
			id, label                       string
			templateID                      sql.NullString
			posX, posY, width, height       int
			energyType                      sql.NullString
			capacity, current, output, rate sql.NullInt64
		)

		if err := buildingRows.Scan(
			&id,
			&templateID,
			&label,
			&posX, &posY, &width, &height,
			&energyType, &capacity, &current, &output, &rate,
		); err != nil {
			return Scene{}, err
		}

		var energy *SceneEnergy
		if energyType.Valid {
			energy = &SceneEnergy{Type: energyType.String}
			if capacity.Valid {
				energy.Capacity = int(capacity.Int64)
			}
			if current.Valid {
				energy.Current = int(current.Int64)
			}
			if output.Valid {
				energy.Output = int(output.Int64)
			}
			if rate.Valid {
				energy.Rate = int(rate.Int64)
			}
		}

		building := SceneBuilding{
			ID:     id,
			Label:  label,
			Rect:   []int{posX, posY, width, height},
			Energy: energy,
		}
		if templateID.Valid {
			building.TemplateID = templateID.String
		}
		scene.Buildings = append(scene.Buildings, building)
	}
	if err := buildingRows.Err(); err != nil {
		return Scene{}, err
	}

	agentRows, err := db.QueryContext(ctx, `
        SELECT s.id,
               s.template_id,
               COALESCE(s.label, t.label) AS label,
               COALESCE(r.pos_x, s.position_x::double precision) AS pos_x,
               COALESCE(r.pos_y, s.position_y::double precision) AS pos_y,
               COALESCE(s.color, t.color) AS color
          FROM system_scene_agents s
          LEFT JOIN system_template_agents t ON t.id = s.template_id
          LEFT JOIN agent_runtime_state r ON r.agent_id = s.id
         WHERE s.scene_id = $1
         ORDER BY s.id
    `, sceneID)
	if err != nil {
		return Scene{}, err
	}
	defer agentRows.Close()

	agentMap := make(map[string]*SceneAgent)

	for agentRows.Next() {
		var (
			id, label  string
			templateID sql.NullString
			x, y       float64
			color      sql.NullInt64
		)

		if err := agentRows.Scan(&id, &templateID, &label, &x, &y, &color); err != nil {
			return Scene{}, err
		}

		agent := SceneAgent{
			ID:       id,
			Label:    label,
			Position: []float64{x, y},
		}
		if templateID.Valid {
			agent.TemplateID = templateID.String
		}
		if color.Valid {
			agent.Color = int(color.Int64)
		}
		scene.Agents = append(scene.Agents, agent)
		agentMap[id] = &scene.Agents[len(scene.Agents)-1]
	}
	if err := agentRows.Err(); err != nil {
		return Scene{}, err
	}

	if len(agentMap) > 0 {
		actionRows, err := db.QueryContext(ctx, `
            SELECT agent_id, action
              FROM system_scene_agent_actions
             WHERE agent_id = ANY (SELECT id FROM system_scene_agents WHERE scene_id = $1)
             ORDER BY agent_id, action
        `, sceneID)
		if err != nil {
			return Scene{}, err
		}
		defer actionRows.Close()

		for actionRows.Next() {
			var agentID, action string
			if err := actionRows.Scan(&agentID, &action); err != nil {
				return Scene{}, err
			}
			if agent, ok := agentMap[agentID]; ok {
				agent.Actions = append(agent.Actions, action)
			}
		}
		if err := actionRows.Err(); err != nil {
			return Scene{}, err
		}
	}

	templateRows, err := db.QueryContext(ctx, `
        SELECT id, label, energy_type, energy_capacity, energy_current, energy_output, energy_rate
          FROM system_template_buildings
         ORDER BY id
    `)
	if err != nil {
		return Scene{}, err
	}
	defer templateRows.Close()

	for templateRows.Next() {
		var (
			id, label                       string
			energyType                      sql.NullString
			capacity, current, output, rate sql.NullInt64
		)

		if err := templateRows.Scan(&id, &label, &energyType, &capacity, &current, &output, &rate); err != nil {
			return Scene{}, err
		}

		var energy *SceneEnergy
		if energyType.Valid {
			energy = &SceneEnergy{Type: energyType.String}
			if capacity.Valid {
				energy.Capacity = int(capacity.Int64)
			}
			if current.Valid {
				energy.Current = int(current.Int64)
			}
			if output.Valid {
				energy.Output = int(output.Int64)
			}
			if rate.Valid {
				energy.Rate = int(rate.Int64)
			}
		}

		scene.BuildingTemplates = append(scene.BuildingTemplates, BuildingTemplate{
			ID:     id,
			Label:  label,
			Energy: energy,
		})
	}
	agentTemplateRows, err := db.QueryContext(ctx, `
        SELECT id, label, color, default_position_x, default_position_y
          FROM system_template_agents
         ORDER BY id
    `)
	if err != nil {
		return Scene{}, err
	}
	defer agentTemplateRows.Close()

	for agentTemplateRows.Next() {
		var (
			id, label string
			color     sql.NullInt64
			posX      sql.NullInt64
			posY      sql.NullInt64
		)

		if err := agentTemplateRows.Scan(&id, &label, &color, &posX, &posY); err != nil {
			return Scene{}, err
		}

		tpl := AgentTemplate{
			ID:    id,
			Label: label,
		}
		if color.Valid {
			tpl.Color = int(color.Int64)
		}
		if posX.Valid && posY.Valid {
			tpl.Position = []int{int(posX.Int64), int(posY.Int64)}
		}

		scene.AgentTemplates = append(scene.AgentTemplates, tpl)
	}
	if err := agentTemplateRows.Err(); err != nil {
		return Scene{}, err
	}

	return scene, nil
}

var sceneLoader = loadSceneFromStore
