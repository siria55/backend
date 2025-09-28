package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"eeo/backend/docs"
	"eeo/backend/internal/config"
	actionservice "eeo/backend/internal/service/action"
	"eeo/backend/internal/service/game"
	"github.com/gin-gonic/gin"
)

// Server 封装 HTTP 引导与路由注册逻辑。
type Server struct {
	cfg         config.Config
	engine      *gin.Engine
	gameSvc     GameService
	actionSvc   *actionservice.Service
	sceneStream *sceneStream
}

// New 根据配置与服务依赖构造一个新的 HTTP Server。
func New(cfg config.Config, gameSvc GameService, actionSvc *actionservice.Service) *Server {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	srv := &Server{
		cfg:         cfg,
		engine:      gin.New(),
		gameSvc:     gameSvc,
		actionSvc:   actionSvc,
		sceneStream: newSceneStream(gameSvc, time.Second, 1, game.DefaultDrainFactor),
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
		gameRoutes := v1.Group("/game")
		{
			gameRoutes.GET("/scene", s.getGameScene)
			gameRoutes.GET("/scene/stream", s.streamGameScene)
			gameRoutes.POST("/scene/buildings/:buildingID/energy", s.updateGameBuildingEnergy)
			gameRoutes.PUT("/scene/agents/:agentID/position", s.updateGameAgentPosition)
			gameRoutes.POST("/scene/agents/:agentID/behaviors/maintain-energy", s.maintainEnergyBalance)
		}

		system := v1.Group("/system")
		{
			system.GET("/scene", s.getSystemScene)
			system.PUT("/scene", s.updateSystemScene)
			system.PUT("/templates/buildings/:id", s.updateSystemBuildingTemplate)
			system.PUT("/templates/agents/:id", s.updateSystemAgentTemplate)
			system.PUT("/scene/buildings/:id", s.updateSystemSceneBuilding)
			system.DELETE("/scene/buildings/:id", s.deleteSystemSceneBuilding)
			system.PUT("/scene/agents/:id", s.updateSystemSceneAgent)
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

func (s *Server) streamGameScene(c *gin.Context) {
	if s.sceneStream == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "scene stream is not available"})
		return
	}
	s.sceneStream.handle(c)
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

func (s *Server) updateGameBuildingEnergy(c *gin.Context) {
	buildingID := c.Param("buildingID")
	if strings.TrimSpace(buildingID) == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "buildingID is required"})
		return
	}

	var req BuildingEnergyUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	building, err := s.gameSvc.UpdateBuildingEnergyCurrent(c.Request.Context(), buildingID, req.Current)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, building)
}

func (s *Server) updateGameAgentPosition(c *gin.Context) {
	agentID := c.Param("agentID")
	if strings.TrimSpace(agentID) == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "agentID is required"})
		return
	}

	var req AgentPositionUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	agent, err := s.gameSvc.UpdateAgentRuntimePosition(c.Request.Context(), agentID, req.X, req.Y)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, agent)
}

func (s *Server) maintainEnergyBalance(c *gin.Context) {
	agentID := c.Param("agentID")
	if strings.TrimSpace(agentID) == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "agentID is required"})
		return
	}

	result, err := s.gameSvc.MaintainEnergyNonNegative(c.Request.Context(), agentID)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, game.ErrInvalidSceneEntity):
			status = http.StatusBadRequest
		case errors.Is(err, game.ErrSolarTemplateMissing):
			status = http.StatusFailedDependency
		case errors.Is(err, game.ErrNoAvailablePlacement):
			status = http.StatusConflict
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	if s.sceneStream != nil {
		s.sceneStream.broadcast(result.Scene)
	}

	response := MaintainEnergyResponse{
		Scene:         result.Scene,
		Created:       result.Created,
		NetFlowBefore: result.NetFlowBefore,
		NetFlowAfter:  result.NetFlowAfter,
		TowersBuilt:   result.TowersBuilt,
	}

	c.JSON(http.StatusOK, response)
}

func (s *Server) updateSystemBuildingTemplate(c *gin.Context) {
	id := c.Param("id")
	if strings.TrimSpace(id) == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "id is required"})
		return
	}

	var req TemplateBuildingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	input := game.UpdateBuildingTemplateInput{
		ID:     id,
		Label:  req.Label,
		Energy: energyRequestToInput(req.Energy),
	}

	snapshot, err := s.gameSvc.UpdateBuildingTemplate(c.Request.Context(), input)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, game.ErrInvalidTemplate) {
			status = http.StatusBadRequest
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, snapshot)
}

func (s *Server) updateSystemAgentTemplate(c *gin.Context) {
	id := c.Param("id")
	if strings.TrimSpace(id) == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "id is required"})
		return
	}

	var req TemplateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	var position *[2]int
	if len(req.DefaultPosition) == 2 {
		coords := [2]int{req.DefaultPosition[0], req.DefaultPosition[1]}
		position = &coords
	}

	input := game.UpdateAgentTemplateInput{
		ID:       id,
		Label:    req.Label,
		Color:    req.Color,
		Position: position,
	}

	snapshot, err := s.gameSvc.UpdateAgentTemplate(c.Request.Context(), input)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, game.ErrInvalidTemplate) {
			status = http.StatusBadRequest
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, snapshot)
}

func (s *Server) updateSystemSceneBuilding(c *gin.Context) {
	id := c.Param("id")
	if strings.TrimSpace(id) == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "id is required"})
		return
	}

	var req SceneBuildingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if len(req.Rect) != 4 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "rect must contain [x, y, width, height]"})
		return
	}

	rect := [4]int{req.Rect[0], req.Rect[1], req.Rect[2], req.Rect[3]}

	input := game.UpdateSceneBuildingInput{
		ID:         id,
		Label:      req.Label,
		TemplateID: normalizeStringPointer(req.TemplateID),
		Rect:       rect,
		Energy:     energyRequestToInput(req.Energy),
	}

	snapshot, err := s.gameSvc.UpdateSceneBuilding(c.Request.Context(), input)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, game.ErrInvalidSceneEntity) {
			status = http.StatusBadRequest
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, snapshot)
}

func (s *Server) deleteSystemSceneBuilding(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "id is required"})
		return
	}

	snapshot, err := s.gameSvc.DeleteSceneBuilding(c.Request.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, game.ErrInvalidSceneEntity) {
			status = http.StatusBadRequest
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, snapshot)
}

func (s *Server) updateSystemSceneAgent(c *gin.Context) {
	id := c.Param("id")
	if strings.TrimSpace(id) == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "id is required"})
		return
	}

	var req SceneAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if len(req.Position) != 2 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "position must contain [x, y]"})
		return
	}

	position := [2]int{req.Position[0], req.Position[1]}

	input := game.UpdateSceneAgentInput{
		ID:         id,
		Label:      req.Label,
		TemplateID: normalizeStringPointer(req.TemplateID),
		Position:   position,
		Color:      req.Color,
		Actions:    req.Actions,
	}

	snapshot, err := s.gameSvc.UpdateSceneAgent(c.Request.Context(), input)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, game.ErrInvalidSceneEntity) {
			status = http.StatusBadRequest
		}
		if errors.Is(err, game.ErrInvalidTemplate) {
			status = http.StatusBadRequest
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
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

type TemplateEnergyRequest struct {
	Type     *string `json:"type"`
	Capacity *int    `json:"capacity"`
	Current  *int    `json:"current"`
	Output   *int    `json:"output"`
	Rate     *int    `json:"rate"`
}

type TemplateBuildingRequest struct {
	Label  string                 `json:"label"`
	Energy *TemplateEnergyRequest `json:"energy"`
}

type TemplateAgentRequest struct {
	Label           string `json:"label"`
	Color           *int   `json:"color"`
	DefaultPosition []int  `json:"defaultPosition"`
}

type SceneBuildingRequest struct {
	Label      string                 `json:"label"`
	TemplateID *string                `json:"templateId"`
	Rect       []int                  `json:"rect"`
	Energy     *TemplateEnergyRequest `json:"energy"`
}

type SceneAgentRequest struct {
	Label      string   `json:"label"`
	TemplateID *string  `json:"templateId"`
	Position   []int    `json:"position"`
	Color      *int     `json:"color"`
	Actions    []string `json:"actions"`
}

// StatusResponse 表示通用状态响应。
type StatusResponse struct {
	Status string `json:"status"`
}

// ErrorResponse 表示通用错误响应。
type ErrorResponse struct {
	Error string `json:"error"`
}

type BuildingEnergyUpdateRequest struct {
	Current float64 `json:"current" binding:"required"`
}

type AgentPositionUpdateRequest struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type MaintainEnergyResponse struct {
	Scene         game.Scene           `json:"scene"`
	Created       []game.SceneBuilding `json:"created"`
	NetFlowBefore float64              `json:"netFlowBefore"`
	NetFlowAfter  float64              `json:"netFlowAfter"`
	TowersBuilt   int                  `json:"towersBuilt"`
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

func energyRequestToInput(payload *TemplateEnergyRequest) *game.UpdateTemplateEnergyInput {
	if payload == nil {
		return nil
	}
	return &game.UpdateTemplateEnergyInput{
		Type:     normalizeStringPointer(payload.Type),
		Capacity: payload.Capacity,
		Current:  payload.Current,
		Output:   payload.Output,
		Rate:     payload.Rate,
	}
}

func normalizeStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	result := trimmed
	return &result
}
