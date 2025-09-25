package game

// Service 负责提供游戏场景配置等业务能力。
type Service struct {
	scene Scene
}

// New 返回默认的 Game 服务实例，并加载初始场景配置。
func New() (*Service, error) {
	scene, err := loadDefaultScene()
	if err != nil {
		return nil, err
	}
	return &Service{scene: scene}, nil
}

// Scene 返回静态场景配置。
func (s *Service) Scene() Scene {
	return s.scene
}
