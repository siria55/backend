package game

import (
	_ "embed"
	"encoding/json"
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
	ID        string   `json:"id"`
	Label     string   `json:"label"`
	Position  []int    `json:"position"`
	Color     int      `json:"color,omitempty"`
	Behaviors []string `json:"behaviors,omitempty"`
}

//go:embed assets/mars_outpost.json
var marsOutpostJSON []byte

func loadDefaultScene() (Scene, error) {
	var scene Scene
	if err := json.Unmarshal(marsOutpostJSON, &scene); err != nil {
		return Scene{}, err
	}
	return scene, nil
}
