package game

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Scene 表示火星场景的静态配置。
type Scene struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Grid       SceneGrid       `json:"grid"`
	Dimensions SceneDims       `json:"dimensions"`
	Buildings  []SceneBuilding `json:"buildings"`
	Agents     []SceneAgent    `json:"agents"`
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
	ID     string       `json:"id"`
	Label  string       `json:"label"`
	Rect   []int        `json:"rect"`
	Energy *SceneEnergy `json:"energy,omitempty"`
}

// SceneAgent 描述场景中的角色。
type SceneAgent struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	Position []int    `json:"position"`
	Color    int      `json:"color,omitempty"`
	Actions  []string `json:"actions,omitempty"`
}

// Snapshot 表示 system_* 表的整合视图。
type Snapshot struct {
	Scene      SceneMeta       `json:"scene"`
	Grid       SceneGrid       `json:"grid"`
	Dimensions SceneDims       `json:"dimensions"`
	Buildings  []SceneBuilding `json:"buildings"`
	Agents     []SceneAgent    `json:"agents"`
}

// SceneMeta 描述场景的基本信息。
type SceneMeta struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// UpdateSceneConfigInput 表示更新 system_* 场景配置所需的数据。
type UpdateSceneConfigInput struct {
	SceneID    string
	Name       string
	Grid       SceneGrid
	Dimensions SceneDims
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
        SELECT id, label, position_x, position_y, size_width, size_height,
               energy_type, energy_capacity, energy_current, energy_output, energy_rate
          FROM system_scene_buildings
         WHERE scene_id = $1
         ORDER BY id
    `, sceneID)
	if err != nil {
		return Scene{}, err
	}
	defer buildingRows.Close()

	for buildingRows.Next() {
		var (
			id, label                       string
			posX, posY, width, height       int
			energyType                      sql.NullString
			capacity, current, output, rate sql.NullInt64
		)

		if err := buildingRows.Scan(
			&id, &label, &posX, &posY, &width, &height,
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

		scene.Buildings = append(scene.Buildings, SceneBuilding{
			ID:     id,
			Label:  label,
			Rect:   []int{posX, posY, width, height},
			Energy: energy,
		})
	}
	if err := buildingRows.Err(); err != nil {
		return Scene{}, err
	}

	agentRows, err := db.QueryContext(ctx, `
        SELECT id, label, position_x, position_y, color
          FROM system_scene_agents
         WHERE scene_id = $1
         ORDER BY id
    `, sceneID)
	if err != nil {
		return Scene{}, err
	}
	defer agentRows.Close()

	agentMap := make(map[string]*SceneAgent)

	for agentRows.Next() {
		var (
			id, label string
			x, y      int
			color     sql.NullInt64
		)

		if err := agentRows.Scan(&id, &label, &x, &y, &color); err != nil {
			return Scene{}, err
		}

		agent := SceneAgent{
			ID:       id,
			Label:    label,
			Position: []int{x, y},
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

	return scene, nil
}

var sceneLoader = loadSceneFromStore
