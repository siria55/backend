package action

import (
	"context"
	"encoding/json"
	"testing"
)

func TestLogActionValidation(t *testing.T) {
	svc := &Service{}

	t.Run("require agent id", func(t *testing.T) {
		err := svc.LogAction(context.Background(), LogActionInput{ActionType: "move"})
		if err == nil || err.Error() != "agent_id required" {
			t.Fatalf("expected agent_id validation error, got %v", err)
		}
	})

	t.Run("require action type", func(t *testing.T) {
		err := svc.LogAction(context.Background(), LogActionInput{AgentID: "ares"})
		if err == nil || err.Error() != "action_type required" {
			t.Fatalf("expected action_type validation error, got %v", err)
		}
	})
}

func TestNullableHelpers(t *testing.T) {
	if val := nullableString(""); val != nil {
		t.Fatalf("expected empty string to map to nil, got %v", val)
	}
	if val := nullableString("mars"); val != "mars" {
		t.Fatalf("expected non-empty string to be passthrough, got %v", val)
	}

	if val := nullableJSON(nil); val != nil {
		t.Fatalf("expected nil json to map to nil, got %v", val)
	}
	if val := nullableJSON([]byte("{}")); string(val.(json.RawMessage)) != "{}" {
		t.Fatalf("expected json bytes to be preserved")
	}
}
