package game

import (
	"context"
	"errors"
	"fmt"
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
	Schema  string           `json:"schema"`
	Name    string           `json:"name"`
	Columns []string         `json:"columns"`
	Rows    []map[string]any `json:"rows"`
}

type tableRef struct {
	schema string
	name   string
}

// PreviewDatabaseTables 返回数据库表的预览数据。
func (s *Service) PreviewDatabaseTables(ctx context.Context, requested []string, limit int) ([]TablePreview, error) {
	if s.db == nil {
		return nil, errors.New("database connection unavailable")
	}

	limit = clampPreviewLimit(limit)

	allTables, err := s.listUserTables(ctx)
	if err != nil {
		return nil, err
	}

	if len(allTables) == 0 {
		return []TablePreview{}, nil
	}

	tableMap := make(map[string]tableRef, len(allTables))
	for _, tbl := range allTables {
		key := strings.ToLower(tbl.schema + "." + tbl.name)
		tableMap[key] = tbl
	}

	lookupByName := make(map[string]tableRef)
	for _, tbl := range allTables {
		key := strings.ToLower(tbl.name)
		if _, exists := lookupByName[key]; !exists {
			lookupByName[key] = tbl
		}
	}

	var tables []tableRef
	if len(requested) == 0 {
		tables = allTables
	} else {
		seen := make(map[string]struct{}, len(requested))
		for _, raw := range requested {
			trimmed := strings.TrimSpace(raw)
			if trimmed == "" {
				continue
			}
			key := strings.ToLower(trimmed)
			if _, ok := seen[key]; ok {
				continue
			}
			if tbl, ok := tableMap[key]; ok {
				tables = append(tables, tbl)
				seen[key] = struct{}{}
				continue
			}
			if tbl, ok := lookupByName[key]; ok {
				tables = append(tables, tbl)
				seen[key] = struct{}{}
			}
		}
		if len(tables) == 0 {
			return nil, fmt.Errorf("no matching tables found for preview")
		}
	}

	previews := make([]TablePreview, 0, len(tables))
	for _, table := range tables {
		preview, err := s.previewTable(ctx, table.schema, table.name, limit)
		if err != nil {
			return nil, err
		}
		previews = append(previews, preview)
	}

	return previews, nil
}

func (s *Service) listUserTables(ctx context.Context) ([]tableRef, error) {
	rows, err := s.db.QueryContext(ctx, `
        SELECT schemaname, tablename
          FROM pg_tables
         WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
         ORDER BY schemaname, tablename
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []tableRef
	for rows.Next() {
		var schema, name string
		if err := rows.Scan(&schema, &name); err != nil {
			return nil, err
		}
		tables = append(tables, tableRef{schema: schema, name: name})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tables, nil
}

func (s *Service) previewTable(ctx context.Context, schema, table string, limit int) (TablePreview, error) {
	query := fmt.Sprintf("SELECT * FROM %s.%s LIMIT %d",
		pq.QuoteIdentifier(schema), pq.QuoteIdentifier(table), limit)
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
		Schema:  schema,
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
