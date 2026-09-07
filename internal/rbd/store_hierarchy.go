package rbd

import (
	"context"

	"github.com/Galang17061/strata-api/internal/domain"
)

func (s *Store) SearchHierarchies(ctx context.Context, search string) ([]domain.Hierarchy, error) {
	rows := []domain.Hierarchy{}
	query := `SELECT ` + domain.HierarchyColumns + ` FROM dbo.Hierarchy`
	if search != "" {
		return rows, s.q.SelectContext(ctx, &rows, query+` WHERE (sub_system_name IS NOT NULL AND POSITION($1 IN sub_system_name) > 0) OR (formula IS NOT NULL AND POSITION($1 IN formula) > 0) OR (formula_code IS NOT NULL AND POSITION($1 IN formula_code) > 0) OR POSITION($1 IN hierarchy_id) > 0 ORDER BY hierarchy_id`, search)
	}
	return rows, s.q.SelectContext(ctx, &rows, query+` ORDER BY hierarchy_id`)
}

func (s *Store) HierarchyExists(ctx context.Context, hierarchyId string) (bool, error) {
	var count int
	err := s.q.GetContext(ctx, &count, `SELECT COUNT(1) FROM dbo.Hierarchy WHERE hierarchy_id = $1`, hierarchyId)
	return count > 0, err
}

func (s *Store) SystemExists(ctx context.Context, rbdSystemId string) (bool, error) {
	var count int
	err := s.q.GetContext(ctx, &count, `SELECT COUNT(1) FROM dbo.RbdSystemDrawing WHERE rbd_system_id = $1`, rbdSystemId)
	return count > 0, err
}

func (s *Store) HasChildHierarchies(ctx context.Context, hierarchyId string) (bool, error) {
	var count int
	err := s.q.GetContext(ctx, &count, `SELECT COUNT(1) FROM dbo.Hierarchy WHERE parent_id = $1`, hierarchyId)
	return count > 0, err
}

func (s *Store) MaxLevelOfSystem(ctx context.Context, rbdSystemId *string) (int, error) {
	var level *int
	err := s.q.GetContext(ctx, &level, `SELECT MAX(level) FROM dbo.Hierarchy WHERE rbd_system_id = $1`, rbdSystemId)
	if err != nil || level == nil {
		return 0, err
	}
	return *level, nil
}
