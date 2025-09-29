package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"eeo/backend/internal/config"
	actionservice "eeo/backend/internal/service/action"
	"eeo/backend/internal/service/game"
	"github.com/gin-gonic/gin"
)

type mockGameService struct {
	mu sync.Mutex

	scene           game.Scene
	snapshot        game.Snapshot
	buildingResult  game.SceneBuilding
	lastEnergyInput struct {
		seconds     float64
		drainFactor float64
	}
	energyCalls         int
	deleteBuildingCalls int

	maintainResult game.MaintainEnergyResult
	maintainErr    error
	maintainCalls  int
	lastMaintainID string

	lastBuildingUpdate struct {
		id      string
		current float64
	}
	buildingUpdateCalls int

	lastRuntimeUpdate struct {
		id string
		x  float64
		y  float64
	}
	runtimeUpdateCalls int

	tablePreviews []game.TablePreview
}

func newMockGameService() *mockGameService {
	scene := game.Scene{
		ID:   "mars_outpost",
		Name: "Mars Outpost",
		Grid: game.SceneGrid{Cols: 10, Rows: 10, TileSize: 32},
	}
	snapshot := game.Snapshot{
		Scene: game.SceneMeta{ID: scene.ID, Name: scene.Name},
		Grid:  scene.Grid,
	}
	building := game.SceneBuilding{
		ID:    "storage-1",
		Label: "储能节点",
		Energy: &game.SceneEnergy{
			Type:     "storage",
			Current:  150,
			Capacity: 200,
		},
	}
	agent := game.SceneAgent{
		ID:       "ares-01",
		Label:    "ARES",
		Position: []float64{10, 10},
	}

	scene.Buildings = []game.SceneBuilding{building}
	scene.Agents = []game.SceneAgent{agent}
	snapshot.Buildings = scene.Buildings
	snapshot.Agents = scene.Agents

	return &mockGameService{
		scene:          scene,
		snapshot:       snapshot,
		buildingResult: building,
		tablePreviews: []game.TablePreview{
			{
				Name:    "system_scene_buildings",
				Columns: []string{"id", "label"},
				Rows: []map[string]any{
					{"id": building.ID, "label": building.Label},
				},
			},
		},
	}
}

func (m *mockGameService) Scene() game.Scene {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.scene
}

func (m *mockGameService) Snapshot() game.Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snapshot
}

func (m *mockGameService) UpdateSceneConfig(_ context.Context, _ game.UpdateSceneConfigInput) (game.Snapshot, error) {
	return m.Snapshot(), nil
}

func (m *mockGameService) UpdateBuildingTemplate(_ context.Context, _ game.UpdateBuildingTemplateInput) (game.Snapshot, error) {
	return m.Snapshot(), nil
}

func (m *mockGameService) UpdateAgentTemplate(_ context.Context, _ game.UpdateAgentTemplateInput) (game.Snapshot, error) {
	return m.Snapshot(), nil
}

func (m *mockGameService) UpdateSceneBuilding(_ context.Context, _ game.UpdateSceneBuildingInput) (game.Snapshot, error) {
	return m.Snapshot(), nil
}

func (m *mockGameService) ListSceneBuildings(_ context.Context, _ string, limit int) ([]game.SceneBuilding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	buildings := m.scene.Buildings
	if limit > 0 && len(buildings) > limit {
		buildings = buildings[:limit]
	}

	cloned := make([]game.SceneBuilding, len(buildings))
	copy(cloned, buildings)
	return cloned, nil
}

func (m *mockGameService) UpdateSceneAgent(_ context.Context, _ game.UpdateSceneAgentInput) (game.Snapshot, error) {
	return m.Snapshot(), nil
}

func (m *mockGameService) DeleteSceneBuilding(_ context.Context, id string) (game.Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleteBuildingCalls++

	for idx, building := range m.scene.Buildings {
		if building.ID == id {
			m.scene.Buildings = append(m.scene.Buildings[:idx], m.scene.Buildings[idx+1:]...)
			m.snapshot.Buildings = m.scene.Buildings
			return m.snapshot, nil
		}
	}

	return game.Snapshot{}, fmt.Errorf("building %s not found", id)
}

func (m *mockGameService) UpdateBuildingEnergyCurrent(_ context.Context, id string, current float64) (game.SceneBuilding, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastBuildingUpdate = struct {
		id      string
		current float64
	}{id: id, current: current}
	m.buildingUpdateCalls++
	result := m.buildingResult
	if result.Energy != nil {
		energy := *result.Energy
		energy.Current = int(current)
		result.Energy = &energy
	}
	return result, nil
}

func (m *mockGameService) AdvanceEnergyState(_ context.Context, seconds float64, drainFactor float64) (game.Scene, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastEnergyInput = struct {
		seconds     float64
		drainFactor float64
	}{seconds: seconds, drainFactor: drainFactor}
	m.energyCalls++
	return m.scene, nil
}

func (m *mockGameService) UpdateAgentRuntimePosition(_ context.Context, agentID string, posX, posY float64) (game.SceneAgent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastRuntimeUpdate = struct {
		id string
		x  float64
		y  float64
	}{id: agentID, x: posX, y: posY}
	m.runtimeUpdateCalls++

	for idx, agent := range m.scene.Agents {
		if agent.ID == agentID {
			updated := agent
			updated.Position = []float64{posX, posY}
			m.scene.Agents[idx] = updated
			m.snapshot.Agents = m.scene.Agents
			return updated, nil
		}
	}

	return game.SceneAgent{}, fmt.Errorf("agent %s not found", agentID)
}

func (m *mockGameService) MaintainEnergyNonNegative(_ context.Context, agentID string) (game.MaintainEnergyResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastMaintainID = agentID
	m.maintainCalls++
	if m.maintainErr != nil {
		return game.MaintainEnergyResult{}, m.maintainErr
	}
	if m.maintainResult.Scene.ID == "" {
		return game.MaintainEnergyResult{Scene: m.scene}, nil
	}
	return m.maintainResult, nil
}

func (m *mockGameService) PreviewDatabaseTables(_ context.Context, requested []string, limit int) ([]game.TablePreview, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.tablePreviews) == 0 {
		return []game.TablePreview{}, nil
	}

	if len(requested) == 0 {
		return m.tablePreviews, nil
	}

	requestedSet := make(map[string]struct{}, len(requested))
	for _, name := range requested {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			requestedSet[trimmed] = struct{}{}
		}
	}

	var filtered []game.TablePreview
	for _, preview := range m.tablePreviews {
		if _, ok := requestedSet[preview.Name]; ok {
			filtered = append(filtered, preview)
		}
	}

	return filtered, nil
}

func newTestServer() (*Server, *mockGameService) {
	gin.SetMode(gin.TestMode)

	mockSvc := newMockGameService()
	cfg := config.Config{}
	srv := New(cfg, mockSvc, actionservice.New(nil))
	srv.sceneStream.stop()
	return srv, mockSvc
}

func TestServerGetGameScene(t *testing.T) {
	srv, mockSvc := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/v1/game/scene", nil)
	resp := httptest.NewRecorder()
	srv.engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 OK, got %d", resp.Code)
	}

	var payload game.Scene
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload.ID != mockSvc.scene.ID {
		t.Fatalf("expected scene id %q, got %q", mockSvc.scene.ID, payload.ID)
	}
	if len(payload.Buildings) != len(mockSvc.scene.Buildings) {
		t.Fatalf("expected %d buildings, got %d", len(mockSvc.scene.Buildings), len(payload.Buildings))
	}
}

func TestServerUpdateBuildingEnergy(t *testing.T) {
	srv, mockSvc := newTestServer()

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/game/scene/buildings/"+mockSvc.buildingResult.ID+"/energy",
		strings.NewReader(`{"current":120}`),
	)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	srv.engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 OK, got %d", resp.Code)
	}

	mockSvc.mu.Lock()
	defer mockSvc.mu.Unlock()
	if mockSvc.buildingUpdateCalls == 0 {
		t.Fatalf("expected UpdateBuildingEnergyCurrent to be called")
	}
	if mockSvc.lastBuildingUpdate.id != mockSvc.buildingResult.ID {
		t.Fatalf("expected building id %q, got %q", mockSvc.buildingResult.ID, mockSvc.lastBuildingUpdate.id)
	}
	if mockSvc.lastBuildingUpdate.current != 120 {
		t.Fatalf("expected current=120, got %v", mockSvc.lastBuildingUpdate.current)
	}
}

func TestServerMaintainEnergyPost(t *testing.T) {
	srv, mockSvc := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/v1/game/scene/agents/ares-01/behaviors/maintain-energy", nil)
	resp := httptest.NewRecorder()
	srv.engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 OK, got %d", resp.Code)
	}

	mockSvc.mu.Lock()
	defer mockSvc.mu.Unlock()
	if mockSvc.maintainCalls == 0 {
		t.Fatalf("expected MaintainEnergyNonNegative to be called")
	}
}

func TestServerMaintainEnergyGetMethodNotAllowed(t *testing.T) {
	srv, mockSvc := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/v1/game/scene/agents/ares-01/behaviors/maintain-energy", nil)
	resp := httptest.NewRecorder()
	srv.engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected HTTP 405, got %d", resp.Code)
	}

	mockSvc.mu.Lock()
	defer mockSvc.mu.Unlock()
	if mockSvc.maintainCalls != 0 {
		t.Fatalf("expected MaintainEnergyNonNegative not to be called")
	}
}

func TestServerDeleteSceneBuilding(t *testing.T) {
	srv, mockSvc := newTestServer()

	req := httptest.NewRequest(http.MethodDelete, "/v1/system/scene/buildings/"+mockSvc.buildingResult.ID, nil)
	resp := httptest.NewRecorder()
	srv.engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 OK, got %d", resp.Code)
	}

	mockSvc.mu.Lock()
	defer mockSvc.mu.Unlock()
	if mockSvc.deleteBuildingCalls == 0 {
		t.Fatalf("expected DeleteSceneBuilding to be called")
	}
	if len(mockSvc.scene.Buildings) != 0 {
		t.Fatalf("expected building to be removed, got %d", len(mockSvc.scene.Buildings))
	}
}

func TestServerPreviewDatabaseTables(t *testing.T) {
	srv, mockSvc := newTestServer()
	mockSvc.mu.Lock()
	mockSvc.tablePreviews = []game.TablePreview{
		{
			Name:    "system_scene_buildings",
			Columns: []string{"id"},
			Rows:    []map[string]any{{"id": "storage-1"}},
		},
		{
			Name:    "system_scene_agents",
			Columns: []string{"id"},
			Rows:    []map[string]any{{"id": "ares-01"}},
		},
	}
	mockSvc.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/v1/system/db/preview", nil)
	resp := httptest.NewRecorder()
	srv.engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 OK, got %d", resp.Code)
	}

	var payload struct {
		Tables []game.TablePreview `json:"tables"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(payload.Tables) != len(mockSvc.tablePreviews) {
		t.Fatalf("expected %d tables, got %d", len(mockSvc.tablePreviews), len(payload.Tables))
	}
}
