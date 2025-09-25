package game

import "time"

// Service 负责提供游戏状态相关业务能力的占位实现。
type Service struct{}

// Snapshot 表示火星前哨站的即时状态。
type Snapshot struct {
	Cycle       string    `json:"cycle"`
	Population  int       `json:"population"`
	NextEvent   string    `json:"next_event"`
	EventTime   time.Time `json:"event_time"`
	Environment string    `json:"environment"`
}

// New 返回默认的 Game 服务实例。
func New() *Service {
	return &Service{}
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
