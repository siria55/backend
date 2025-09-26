package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"eeo/backend/internal/config"
	actionservice "eeo/backend/internal/service/action"
	"eeo/backend/internal/service/game"
	"github.com/gin-gonic/gin"
)

// Server 封装 HTTP 引导与路由注册逻辑。
type Server struct {
	cfg    config.Config
	engine *gin.Engine
}

// New 根据配置与服务依赖构造一个新的 HTTP Server。
func New(cfg config.Config, gameSvc *game.Service, actionSvc *actionservice.Service) *Server {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())
	engine.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":       "ok",
			"environment":  cfg.Environment,
			"dependencies": []string{"game", "database"},
		})
	})

	v1 := engine.Group("/v1")
	{
		v1.GET("/game/scene", func(c *gin.Context) {
			c.JSON(http.StatusOK, gameSvc.Scene())
		})

		agents := v1.Group("/agents")
		{
			agents.POST("/:agentID/actions", func(c *gin.Context) {
				agentID := c.Param("agentID")
				if agentID == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "agentID is required"})
					return
				}

				var req logActionRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				input := actionservice.LogActionInput{
					AgentID:       agentID,
					Label:         req.Label,
					ActionType:    req.ActionType,
					Payload:       json.RawMessage(req.Payload),
					IssuedBy:      req.IssuedBy,
					Source:        req.Source,
					CorrelationID: req.CorrelationID,
					ResultStatus:  req.ResultStatus,
					ResultMessage: req.ResultMessage,
					Actions:       req.Actions,
				}

				if err := actionSvc.LogAction(c.Request.Context(), input); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				c.JSON(http.StatusCreated, gin.H{"status": "logged"})
			})

			agents.GET("/:agentID/actions", func(c *gin.Context) {
				agentID := c.Param("agentID")
				if agentID == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "agentID is required"})
					return
				}

				limit := 0
				if raw := c.Query("limit"); raw != "" {
					if parsed, err := strconv.Atoi(raw); err == nil {
						limit = parsed
					}
				}

				events, err := actionSvc.ListEvents(c.Request.Context(), agentID, limit)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				c.JSON(http.StatusOK, events)
			})

			agents.GET("/:agentID/state", func(c *gin.Context) {
				agentID := c.Param("agentID")
				if agentID == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "agentID is required"})
					return
				}

				state, err := actionSvc.GetState(c.Request.Context(), agentID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				c.JSON(http.StatusOK, state)
			})
		}
	}

	return &Server{cfg: cfg, engine: engine}
}

type logActionRequest struct {
	Label         string          `json:"label"`
	ActionType    string          `json:"action_type" binding:"required"`
	Payload       json.RawMessage `json:"payload"`
	IssuedBy      string          `json:"issued_by"`
	Source        string          `json:"source"`
	CorrelationID string          `json:"correlation_id"`
	ResultStatus  string          `json:"result_status"`
	ResultMessage string          `json:"result_message"`
	Actions       []string        `json:"actions"`
}

// Run 监听并启动 HTTP 服务。
func (s *Server) Run() error {
	return s.engine.Run(s.cfg.HTTP.Address())
}
