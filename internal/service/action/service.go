package action

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/lib/pq"
)

type Service struct {
	db *sql.DB
}

type Event struct {
	ID            int64           `json:"id"`
	AgentID       string          `json:"agent_id"`
	ActionType    string          `json:"action_type"`
	Payload       json.RawMessage `json:"payload,omitempty"`
	IssuedBy      string          `json:"issued_by,omitempty"`
	Source        string          `json:"source,omitempty"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	ResultStatus  string          `json:"result_status,omitempty"`
	ResultMessage string          `json:"result_message,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

type State struct {
	AgentID   string    `json:"agent_id"`
	Actions   []string  `json:"actions"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LogActionInput struct {
	AgentID       string          `json:"agent_id"`
	Label         string          `json:"label,omitempty"`
	ActionType    string          `json:"action_type"`
	Payload       json.RawMessage `json:"payload,omitempty"`
	IssuedBy      string          `json:"issued_by,omitempty"`
	Source        string          `json:"source,omitempty"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	ResultStatus  string          `json:"result_status,omitempty"`
	ResultMessage string          `json:"result_message,omitempty"`
	Actions       []string        `json:"actions"`
}

func New(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) LogAction(ctx context.Context, in LogActionInput) error {
	if in.AgentID == "" {
		return errors.New("agent_id required")
	}
	if in.ActionType == "" {
		return errors.New("action_type required")
	}

	label := in.Label
	if label == "" {
		label = in.AgentID
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(
		ctx,
		`INSERT INTO agents (id, label) VALUES ($1, $2)
         ON CONFLICT (id) DO UPDATE SET label = EXCLUDED.label`,
		in.AgentID, label,
	); err != nil {
		return err
	}

	if _, err = tx.ExecContext(
		ctx,
		`INSERT INTO agent_action_events
            (agent_id, action_type, payload, issued_by, source, correlation_id, result_status, result_message)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		in.AgentID,
		in.ActionType,
		nullableJSON(in.Payload),
		nullableString(in.IssuedBy),
		nullableString(in.Source),
		nullableString(in.CorrelationID),
		nullableString(in.ResultStatus),
		nullableString(in.ResultMessage),
	); err != nil {
		return err
	}

	if in.Actions != nil {
		if _, err = tx.ExecContext(
			ctx,
			`INSERT INTO agent_action_state (agent_id, actions, updated_at)
			 VALUES ($1, $2, now())
			 ON CONFLICT (agent_id)
			 DO UPDATE SET actions = EXCLUDED.actions, updated_at = EXCLUDED.updated_at`,
			in.AgentID,
			pq.Array(in.Actions),
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Service) ListEvents(ctx context.Context, agentID string, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, agent_id, action_type, payload, issued_by, source, correlation_id, result_status, result_message, created_at
         FROM agent_action_events
         WHERE agent_id = $1
         ORDER BY created_at DESC
         LIMIT $2`,
		agentID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var evt Event
		if err := rows.Scan(
			&evt.ID,
			&evt.AgentID,
			&evt.ActionType,
			&evt.Payload,
			&evt.IssuedBy,
			&evt.Source,
			&evt.CorrelationID,
			&evt.ResultStatus,
			&evt.ResultMessage,
			&evt.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, evt)
	}
	return events, rows.Err()
}

func (s *Service) GetState(ctx context.Context, agentID string) (State, error) {
	var st State
	err := s.db.QueryRowContext(
		ctx,
		`SELECT agent_id, actions, updated_at FROM agent_action_state WHERE agent_id = $1`,
		agentID,
	).Scan(&st.AgentID, pq.Array(&st.Actions), &st.UpdatedAt)
	if err == sql.ErrNoRows {
		return State{AgentID: agentID, Actions: []string{}, UpdatedAt: time.Time{}}, nil
	}
	return st, err
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullableJSON(raw json.RawMessage) interface{} {
	if len(raw) == 0 {
		return nil
	}
	return raw
}
