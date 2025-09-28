package server

import (
	"bytes"
	"context"
	"encoding/json"
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
	energyCalls int

	lastBuildingUpdate struct {
		id      string
		current float64
	}
	buildingUpdateCalls int
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

	scene.Buildings = []game.SceneBuilding{building}
	snapshot.Buildings = scene.Buildings

	return &mockGameService{
		scene:          scene,
		snapshot:       snapshot,
		buildingResult: building,
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

func (m *mockGameService) UpdateSceneAgent(_ context.Context, _ game.UpdateSceneAgentInput) (game.Snapshot, error) {
	return m.Snapshot(), nil
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

func TestServerAdvanceGameEnergy(t *testing.T) {
	srv, mockSvc := newTestServer()

	body := bytes.NewBufferString(`{"seconds":5,"drainFactor":0.15}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/game/scene/energy/tick", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	srv.engine.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 OK, got %d", resp.Code)
	}

	mockSvc.mu.Lock()
	defer mockSvc.mu.Unlock()

	if mockSvc.energyCalls == 0 {
		t.Fatalf("expected AdvanceEnergyState to be called")
	}
	if mockSvc.lastEnergyInput.seconds != 5 {
		t.Fatalf("expected seconds=5, got %v", mockSvc.lastEnergyInput.seconds)
	}
	if mockSvc.lastEnergyInput.drainFactor != 0.15 {
		t.Fatalf("expected drainFactor=0.15, got %v", mockSvc.lastEnergyInput.drainFactor)
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
