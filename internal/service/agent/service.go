package agent

// Service 负责提供 Agent 相关业务能力的占位实现。
type Service struct{}

// Profile 描述一个示例 Agent 的核心属性。
type Profile struct {
	Name          string   `json:"name"`
	Role          string   `json:"role"`
	Capabilities  []string `json:"capabilities"`
	Personality   string   `json:"personality"`
	PrimarySensor string   `json:"primary_sensor"`
}

// New 返回默认的 Agent 服务实例。
func New() *Service {
	return &Service{}
}

// MockProfile 暂时返回火星前哨站主角阿瑞斯-01的静态配置，后续可替换为真实数据。
func (s *Service) MockProfile() Profile {
	return Profile{
		Name:          "ARES-01",
		Role:          "Autonomous Research & Exploration Sentinel",
		Capabilities:  []string{"Scan", "Route", "Sync"},
		Personality:   "冷静务实，偏数据驱动",
		PrimarySensor: "复合光谱雷达",
	}
}
