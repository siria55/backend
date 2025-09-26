package game

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// Service 负责提供游戏场景配置等业务能力。
type Service struct {
	db    *sql.DB
	scene Scene
}

// New 返回默认的 Game 服务实例，并加载初始场景配置。
func New(db *sql.DB, sceneID string) (*Service, error) {
	scene, err := sceneLoader(db, sceneID)
	if err != nil {
		return nil, err
	}
	return &Service{db: db, scene: scene}, nil
}

// Scene 返回静态场景配置。
func (s *Service) Scene() Scene {
	return s.scene
}

// Snapshot 返回整合后的系统场景原始数据，供管理端查看。
func (s *Service) Snapshot() Snapshot {
	return Snapshot{
		Scene:      SceneMeta{ID: s.scene.ID, Name: s.scene.Name},
		Grid:       s.scene.Grid,
		Dimensions: s.scene.Dimensions,
		Buildings:  s.scene.Buildings,
		Agents:     s.scene.Agents,
	}
}

var ErrInvalidSceneConfig = errors.New("invalid scene config")

// UpdateSceneConfig 更新 system_* 表中的场景基础配置，并返回最新快照。
func (s *Service) UpdateSceneConfig(ctx context.Context, in UpdateSceneConfigInput) (Snapshot, error) {
	if err := validateSceneConfig(in); err != nil {
		return Snapshot{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Snapshot{}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	name := strings.TrimSpace(in.Name)
	res, err := tx.ExecContext(ctx, `UPDATE system_scenes SET name = $1 WHERE id = $2`, name, in.SceneID)
	if err != nil {
		return Snapshot{}, err
	}
	if rows, errRows := res.RowsAffected(); errRows == nil && rows == 0 {
		return Snapshot{}, fmt.Errorf("%w: scene %s not found", ErrInvalidSceneConfig, in.SceneID)
	}

	if _, err = tx.ExecContext(ctx, `
		INSERT INTO system_scene_grid (scene_id, cols, rows, tile_size)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (scene_id)
		DO UPDATE SET cols = EXCLUDED.cols, rows = EXCLUDED.rows, tile_size = EXCLUDED.tile_size
	`, in.SceneID, in.Grid.Cols, in.Grid.Rows, in.Grid.TileSize); err != nil {
		return Snapshot{}, err
	}

	if _, err = tx.ExecContext(ctx, `
		INSERT INTO system_scene_dimensions (scene_id, width, height)
		VALUES ($1, $2, $3)
		ON CONFLICT (scene_id)
		DO UPDATE SET width = EXCLUDED.width, height = EXCLUDED.height
	`, in.SceneID, in.Dimensions.Width, in.Dimensions.Height); err != nil {
		return Snapshot{}, err
	}

	if err = tx.Commit(); err != nil {
		return Snapshot{}, err
	}

	updated, err := sceneLoader(s.db, in.SceneID)
	if err != nil {
		return Snapshot{}, err
	}
	s.scene = updated
	return s.Snapshot(), nil
}

func validateSceneConfig(in UpdateSceneConfigInput) error {
	if strings.TrimSpace(in.SceneID) == "" {
		return fmt.Errorf("%w: scene_id required", ErrInvalidSceneConfig)
	}
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("%w: name required", ErrInvalidSceneConfig)
	}
	if in.Grid.Cols <= 0 {
		return fmt.Errorf("%w: grid.cols must be positive", ErrInvalidSceneConfig)
	}
	if in.Grid.Rows <= 0 {
		return fmt.Errorf("%w: grid.rows must be positive", ErrInvalidSceneConfig)
	}
	if in.Grid.TileSize <= 0 {
		return fmt.Errorf("%w: grid.tileSize must be positive", ErrInvalidSceneConfig)
	}
	if in.Dimensions.Width <= 0 {
		return fmt.Errorf("%w: dimensions.width must be positive", ErrInvalidSceneConfig)
	}
	if in.Dimensions.Height <= 0 {
		return fmt.Errorf("%w: dimensions.height must be positive", ErrInvalidSceneConfig)
	}
	return nil
}
