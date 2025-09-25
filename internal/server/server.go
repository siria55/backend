package server

import (
	"net/http"

	"eeo/backend/internal/config"
	"eeo/backend/internal/service/agent"
	"eeo/backend/internal/service/game"
	"github.com/gin-gonic/gin"
)

// Server 封装 HTTP 引导与路由注册逻辑。
type Server struct {
	cfg    config.Config
	engine *gin.Engine
}

// New 根据配置与服务依赖构造一个新的 HTTP Server。
func New(cfg config.Config, agentSvc *agent.Service, gameSvc *game.Service) *Server {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":       "ok",
			"environment":  cfg.Environment,
			"dependencies": []string{"agent", "game"},
		})
	})

	v1 := engine.Group("/v1")
	{
		v1.GET("/agents/mock", func(c *gin.Context) {
			c.JSON(http.StatusOK, agentSvc.MockProfile())
		})

		v1.GET("/game/state", func(c *gin.Context) {
			c.JSON(http.StatusOK, gameSvc.Snapshot())
		})
	}

	return &Server{cfg: cfg, engine: engine}
}

// Run 监听并启动 HTTP 服务。
func (s *Server) Run() error {
	return s.engine.Run(s.cfg.HTTP.Address())
}
