package game

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lib/pq"
)

const (
	defaultPreviewLimit = 25
	maxPreviewLimit     = 200
)

// TablePreview 表示数据库表的预览数据。
type TablePreview struct {
	Name    string           `json:"name"`
	Columns []string         `json:"columns"`
	Rows    []map[string]any `json:"rows"`
}

// PreviewDatabaseTables 返回数据库表的预览数据。
func (s *Service) PreviewDatabaseTables(ctx context.Context, requested []string, limit int) ([]TablePreview, error) {
	if s.db == nil {
		return nil, errors.New("database connection unavailable")
	}

	limit = clampPreviewLimit(limit)

	allTables, err := s.listPublicTables(ctx)
	if err != nil {
		return nil, err
	}

	if len(allTables) == 0 {
		return []TablePreview{}, nil
	}

	tableSet := make(map[string]struct{}, len(allTables))
	for _, name := range allTables {
		tableSet[name] = struct{}{}
	}

	var tables []string
	if len(requested) == 0 {
		tables = allTables
	} else {
		seen := make(map[string]struct{}, len(requested))
		for _, raw := range requested {
			trimmed := strings.TrimSpace(raw)
			if trimmed == "" {
				continue
			}
			if _, ok := seen[trimmed]; ok {
				continue
			}
			if _, ok := tableSet[trimmed]; ok {
				tables = append(tables, trimmed)
				seen[trimmed] = struct{}{}
			}
		}
		if len(tables) == 0 {
			return nil, fmt.Errorf("no matching tables found for preview")
		}
	}

	previews := make([]TablePreview, 0, len(tables))
	for _, table := range tables {
		preview, err := s.previewTable(ctx, table, limit)
		if err != nil {
			return nil, err
		}
		previews = append(previews, preview)
	}

	return previews, nil
}

func (s *Service) listPublicTables(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT table_name
          FROM information_schema.tables
         WHERE table_schema = 'public'
           AND table_type = 'BASE TABLE'
         ORDER BY table_name
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Strings(tables)
	return tables, nil
}

func (s *Service) previewTable(ctx context.Context, table string, limit int) (TablePreview, error) {
	query := fmt.Sprintf("SELECT * FROM %s LIMIT %d", pq.QuoteIdentifier(table), limit)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return TablePreview{}, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return TablePreview{}, err
	}

	rawValues := make([]any, len(columns))
	scanTargets := make([]any, len(columns))
	for i := range scanTargets {
		scanTargets[i] = &rawValues[i]
	}

	data := make([]map[string]any, 0, limit)
	for rows.Next() {
		for idx := range rawValues {
			rawValues[idx] = nil
		}

		if err := rows.Scan(scanTargets...); err != nil {
			return TablePreview{}, err
		}

		row := make(map[string]any, len(columns))
		for idx, col := range columns {
			row[col] = normalizeSQLValue(rawValues[idx])
		}
		data = append(data, row)
	}
	if err := rows.Err(); err != nil {
		return TablePreview{}, err
	}

	return TablePreview{
		Name:    table,
		Columns: columns,
		Rows:    data,
	}, nil
}

func normalizeSQLValue(value any) any {
	switch v := value.(type) {
	case nil:
		return nil
	case []byte:
		return string(v)
	case time.Time:
		return v.Format(time.RFC3339Nano)
	default:
		return v
	}
}

func clampPreviewLimit(limit int) int {
	if limit <= 0 {
		return defaultPreviewLimit
	}
	if limit > maxPreviewLimit {
		return maxPreviewLimit
	}
	return limit
}
