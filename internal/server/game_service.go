package server

import (
	"context"

	"eeo/backend/internal/service/game"
)

// GameService 定义了 HTTP 层所需的游戏服务能力，方便在测试中替换实现。
type GameService interface {
	Scene() game.Scene
	Snapshot() game.Snapshot
	UpdateSceneConfig(context.Context, game.UpdateSceneConfigInput) (game.Snapshot, error)
	UpdateBuildingTemplate(context.Context, game.UpdateBuildingTemplateInput) (game.Snapshot, error)
	UpdateAgentTemplate(context.Context, game.UpdateAgentTemplateInput) (game.Snapshot, error)
	UpdateSceneBuilding(context.Context, game.UpdateSceneBuildingInput) (game.Snapshot, error)
	UpdateSceneAgent(context.Context, game.UpdateSceneAgentInput) (game.Snapshot, error)
	UpdateBuildingEnergyCurrent(context.Context, string, float64) (game.SceneBuilding, error)
	AdvanceEnergyState(context.Context, float64, float64) (game.Scene, error)
	UpdateAgentRuntimePosition(context.Context, string, float64, float64) (game.SceneAgent, error)
	MaintainEnergyNonNegative(context.Context, string) (game.MaintainEnergyResult, error)
}
