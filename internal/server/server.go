package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"eeo/backend/docs"
	"eeo/backend/internal/config"
	actionservice "eeo/backend/internal/service/action"
	"eeo/backend/internal/service/game"
	"github.com/gin-gonic/gin"
)

// Server 封装 HTTP 引导与路由注册逻辑。
type Server struct {
	cfg       config.Config
	engine    *gin.Engine
	gameSvc   *game.Service
	actionSvc *actionservice.Service
}

// New 根据配置与服务依赖构造一个新的 HTTP Server。
func New(cfg config.Config, gameSvc *game.Service, actionSvc *actionservice.Service) *Server {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	srv := &Server{
		cfg:       cfg,
		engine:    gin.New(),
		gameSvc:   gameSvc,
		actionSvc: actionSvc,
	}

	srv.engine.Use(gin.Logger(), gin.Recovery())
	srv.engine.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	srv.registerRoutes()

	return srv
}

// Run 监听并启动 HTTP 服务。
func (s *Server) Run() error {
	return s.engine.Run(s.cfg.HTTP.Address())
}

func (s *Server) registerRoutes() {
	s.engine.GET("/healthz", s.healthz)
	s.engine.GET("/swagger", s.swaggerUI)
	s.engine.GET("/swagger/doc.json", s.swaggerSpec)

	v1 := s.engine.Group("/v1")
	{
		v1.GET("/game/scene", s.getGameScene)

		system := v1.Group("/system")
		{
			system.GET("/scene", s.getSystemScene)
			system.PUT("/scene", s.updateSystemScene)
		}

		agents := v1.Group("/agents")
		{
			agents.POST("/:agentID/actions", s.createAgentAction)
			agents.GET("/:agentID/actions", s.listAgentActions)
			agents.GET("/:agentID/state", s.getAgentState)
		}
	}
}

// healthz 返回服务基础健康状态。
func (s *Server) healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":       "ok",
		"environment":  s.cfg.Environment,
		"dependencies": []string{"game", "database"},
	})
}

// getGameScene 获取当前火星场景配置。
func (s *Server) getGameScene(c *gin.Context) {
	c.JSON(http.StatusOK, s.gameSvc.Scene())
}

// getSystemScene 返回 system_* 场景快照。
func (s *Server) getSystemScene(c *gin.Context) {
	c.JSON(http.StatusOK, s.gameSvc.Snapshot())
}

// updateSystemScene 更新 system_* 场景配置。
func (s *Server) updateSystemScene(c *gin.Context) {
	var req SystemSceneUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "name is required"})
		return
	}

	snapshot, err := s.gameSvc.UpdateSceneConfig(c.Request.Context(), game.UpdateSceneConfigInput{
		SceneID: req.SceneID,
		Name:    req.Name,
		Grid: game.SceneGrid{
			Cols:     req.Grid.Cols,
			Rows:     req.Grid.Rows,
			TileSize: req.Grid.TileSize,
		},
		Dimensions: game.SceneDims{
			Width:  req.Dimensions.Width,
			Height: req.Dimensions.Height,
		},
	})
	if err != nil {
		if errors.Is(err, game.ErrInvalidSceneConfig) {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, snapshot)
}

// createAgentAction 记录 Agent 行为事件。
func (s *Server) createAgentAction(c *gin.Context) {
	agentID := c.Param("agentID")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "agentID is required"})
		return
	}

	var req AgentActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
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

	if err := s.actionSvc.LogAction(c.Request.Context(), input); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, StatusResponse{Status: "logged"})
}

// listAgentActions 返回 Agent 行为事件列表。
func (s *Server) listAgentActions(c *gin.Context) {
	agentID := c.Param("agentID")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "agentID is required"})
		return
	}

	limit := 0
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	events, err := s.actionSvc.ListEvents(c.Request.Context(), agentID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, events)
}

// getAgentState 返回 Agent 行为状态。
func (s *Server) getAgentState(c *gin.Context) {
	agentID := c.Param("agentID")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "agentID is required"})
		return
	}

	state, err := s.actionSvc.GetState(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, state)
}

// swaggerSpec 返回 Swagger JSON。
func (s *Server) swaggerSpec(c *gin.Context) {
	c.Data(http.StatusOK, "application/json", docs.Spec())
}

// swaggerUI 返回 Swagger UI 页面。
func (s *Server) swaggerUI(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerUIHTML))
}

// AgentActionRequest 为记录 Agent 行为的请求体。
type AgentActionRequest struct {
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

// SystemSceneUpdateRequest 为更新系统场景配置的请求体。
type SystemSceneUpdateRequest struct {
	SceneID    string                  `json:"scene_id" binding:"required"`
	Name       string                  `json:"name" binding:"required"`
	Grid       SystemSceneUpdateGrid   `json:"grid" binding:"required"`
	Dimensions SystemSceneUpdateBounds `json:"dimensions" binding:"required"`
}

// SystemSceneUpdateGrid 表示场景网格的更新参数。
type SystemSceneUpdateGrid struct {
	Cols     int `json:"cols"`
	Rows     int `json:"rows"`
	TileSize int `json:"tileSize"`
}

// SystemSceneUpdateBounds 表示场景边界的更新参数。
type SystemSceneUpdateBounds struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// StatusResponse 表示通用状态响应。
type StatusResponse struct {
	Status string `json:"status"`
}

// ErrorResponse 表示通用错误响应。
type ErrorResponse struct {
	Error string `json:"error"`
}

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="zh-CN">
  <head>
    <meta charset="utf-8">
    <title>Mars Outpost API</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css" />
    <style>
      html, body { margin: 0; padding: 0; height: 100%; }
      #swagger-ui { height: 100%; }
    </style>
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
    <script>
      window.onload = function() {
        SwaggerUIBundle({
          url: '/swagger/doc.json',
          dom_id: '#swagger-ui',
          presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
          layout: 'BaseLayout'
        });
      }
    </script>
  </body>
</html>`
