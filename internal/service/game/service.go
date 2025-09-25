package game

import "time"

// Service 负责提供游戏状态相关业务能力的占位实现。
type Service struct {
	scene Scene
}

// Snapshot 表示火星前哨站的即时状态。
type Snapshot struct {
	Cycle       string    `json:"cycle"`
	Population  int       `json:"population"`
	NextEvent   string    `json:"next_event"`
	EventTime   time.Time `json:"event_time"`
	Environment string    `json:"environment"`
}

// New 返回默认的 Game 服务实例，并加载初始场景配置。
func New() (*Service, error) {
	scene, err := loadDefaultScene()
	if err != nil {
		return nil, err
	}
	return &Service{scene: scene}, nil
}

// Snapshot 生成一个示例游戏状态，未来可由游戏引擎实时填充。
func (s *Service) Snapshot() Snapshot {
	return Snapshot{
		Cycle:       "Sol-128",
		Population:  42,
		NextEvent:   "沙尘暴预警",
		EventTime:   time.Now().UTC().Add(15 * time.Minute),
		Environment: "低压、低重力、穹顶保护开启",
	}
}

// Scene 返回静态场景配置。
func (s *Service) Scene() Scene {
	return s.scene
}
