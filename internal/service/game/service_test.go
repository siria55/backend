package game

import "testing"

func TestNewLoadsEmbeddedScene(t *testing.T) {
	svc, err := New()
	if err != nil {
		t.Fatalf("expected scene to load without error, got %v", err)
	}

	scene := svc.Scene()
	if scene.ID == "" {
		t.Fatalf("expected scene id to be populated")
	}
	if scene.Grid.Cols <= 0 || scene.Grid.Rows <= 0 {
		t.Fatalf("expected positive grid dimensions, got %+v", scene.Grid)
	}
	if len(scene.Buildings) == 0 {
		t.Fatalf("expected embedded scene to declare at least one building")
	}
	if len(scene.Agents) == 0 {
		t.Fatalf("expected embedded scene to declare at least one agent")
	}
}
