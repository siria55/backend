package game

import (
	"database/sql"
	"testing"
)

func TestNewUsesSceneLoader(t *testing.T) {
	original := sceneLoader
	t.Cleanup(func() { sceneLoader = original })

	var called bool
	sceneLoader = func(_ *sql.DB, sceneID string) (Scene, error) {
		called = true
		return Scene{ID: sceneID, Name: "test"}, nil
	}

	svc, err := New(nil, "mars_outpost_min")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !called {
		t.Fatalf("expected loader to be called")
	}
	if svc.Scene().ID != "mars_outpost_min" {
		t.Fatalf("unexpected scene id: %s", svc.Scene().ID)
	}
}

func TestEnsureBuildingPlacementDetectsOverlap(t *testing.T) {
	svc := &Service{
		scene: Scene{
			Buildings: []SceneBuilding{
				{ID: "b1", Rect: []int{0, 0, 4, 4}},
				{ID: "b2", Rect: []int{10, 10, 3, 3}},
			},
		},
	}

	if err := svc.ensureBuildingPlacement("b3", [4]int{2, 2, 4, 4}); err == nil {
		t.Fatalf("expected overlap to be detected")
	}

	if err := svc.ensureBuildingPlacement("b1", [4]int{5, 5, 2, 2}); err != nil {
		t.Fatalf("expected non-overlapping placement to pass, got %v", err)
	}
}
