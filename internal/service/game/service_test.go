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
