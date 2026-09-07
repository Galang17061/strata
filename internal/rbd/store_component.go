package rbd

import (
	"context"

	"github.com/Galang17061/strata-api/internal/domain"
)

func (s *Store) ScpList(ctx context.Context, search string) ([]domain.ScpListItem, error) {
	rows := []domain.ScpListItem{}
	query := `SELECT component_name, SUM(total_component) AS total_component, SUM(active_component) AS total_active FROM dbo.SystemComponentProperties`
	if search != "" {
		return rows, s.q.SelectContext(ctx, &rows, query+` WHERE (component_name IS NOT NULL AND POSITION($1 IN component_name) > 0) OR (system_component_id IS NOT NULL AND POSITION($1 IN system_component_id) > 0) GROUP BY component_name ORDER BY component_name`, search)
	}
	return rows, s.q.SelectContext(ctx, &rows, query+` GROUP BY component_name ORDER BY component_name`)
}

func (s *Store) NamedComponents(ctx context.Context, search string, searchConnection bool) ([]domain.SystemComponentProperties, error) {
	rows := []domain.SystemComponentProperties{}
	query := `SELECT ` + domain.SystemComponentColumns + ` FROM dbo.SystemComponentProperties WHERE component_name IS NOT NULL AND component_name <> ''`
	if search == "" {
		return rows, s.q.SelectContext(ctx, &rows, query+` ORDER BY system_component_id`)
	}
	extra := `formula_code`
	if searchConnection {
		extra = `connection_type`
	}
	return rows, s.q.SelectContext(ctx, &rows, query+` AND (POSITION($1 IN component_name) > 0 OR POSITION($1 IN system_component_id) > 0 OR POSITION($1 IN vendor) > 0 OR POSITION($1 IN `+extra+`) > 0) ORDER BY system_component_id`, search)
}

func (s *Store) ComponentsByCodesInParent(ctx context.Context, codes []string, parentId string) ([]domain.SystemComponentProperties, error) {
	rows := []domain.SystemComponentProperties{}
	if len(codes) == 0 {
		return rows, nil
	}
	query, args, err := s.inQuery(`SELECT `+domain.SystemComponentColumns+` FROM dbo.SystemComponentProperties WHERE formula_code IN (?) AND parent_id = ? ORDER BY system_component_id`, codes, parentId)
	if err != nil {
		return nil, err
	}
	return rows, s.q.SelectContext(ctx, &rows, query, args...)
}

func (s *Store) HierarchiesByCodesInParent(ctx context.Context, codes []string, parentId string, levelOneOrVirtual bool) ([]domain.Hierarchy, error) {
	rows := []domain.Hierarchy{}
	if len(codes) == 0 {
		return rows, nil
	}
	query := `SELECT ` + domain.HierarchyColumns + ` FROM dbo.Hierarchy WHERE formula_code IN (?) AND parent_id = ?`
	if levelOneOrVirtual {
		query += ` AND (level = 1 OR level = 999)`
	}
	expanded, args, err := s.inQuery(query+` ORDER BY hierarchy_id`, codes, parentId)
	if err != nil {
		return nil, err
	}
	return rows, s.q.SelectContext(ctx, &rows, expanded, args...)
}

func (s *Store) OtherComponentUsesCode(ctx context.Context, code, excludeId, parentId string) (bool, error) {
	var count int
	err := s.q.GetContext(ctx, &count, `SELECT COUNT(1) FROM dbo.SystemComponentProperties WHERE formula_code = $1 AND system_component_id <> $2 AND parent_id = $3`, code, excludeId, parentId)
	return count > 0, err
}

func (s *Store) OtherHierarchyUsesCode(ctx context.Context, code, excludeId, parentId string) (bool, error) {
	var count int
	err := s.q.GetContext(ctx, &count, `SELECT COUNT(1) FROM dbo.Hierarchy WHERE formula_code = $1 AND hierarchy_id <> $2 AND parent_id = $3`, code, excludeId, parentId)
	return count > 0, err
}

func (s *Store) LevelOneHierarchies(ctx context.Context, rbdSystemId string) ([]domain.Hierarchy, error) {
	rows := []domain.Hierarchy{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.HierarchyColumns+` FROM dbo.Hierarchy WHERE rbd_system_id = $1 AND parent_id = $1 AND level = 1 ORDER BY hierarchy_id`, rbdSystemId)
}

func (s *Store) RootHierarchies(ctx context.Context, rbdSystemId string) ([]domain.Hierarchy, error) {
	rows := []domain.Hierarchy{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.HierarchyColumns+` FROM dbo.Hierarchy WHERE rbd_system_id = $1 AND parent_id = $1 ORDER BY hierarchy_id`, rbdSystemId)
}

func (s *Store) RootHierarchiesByLevel(ctx context.Context, rbdSystemId string, virtual bool) ([]domain.Hierarchy, error) {
	rows := []domain.Hierarchy{}
	condition := `level <> 999`
	if virtual {
		condition = `level = 999`
	}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.HierarchyColumns+` FROM dbo.Hierarchy WHERE rbd_system_id = $1 AND parent_id = $1 AND `+condition+` ORDER BY hierarchy_id`, rbdSystemId)
}

func (s *Store) ChildHierarchiesByVirtual(ctx context.Context, parentId string, virtual bool) ([]domain.Hierarchy, error) {
	rows := []domain.Hierarchy{}
	condition := `level <> 999`
	if virtual {
		condition = `level = 999`
	}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.HierarchyColumns+` FROM dbo.Hierarchy WHERE parent_id = $1 AND `+condition+` ORDER BY hierarchy_id`, parentId)
}

func (s *Store) HierarchiesOfSystemLevelDescending(ctx context.Context, rbdSystemId string) ([]domain.Hierarchy, error) {
	rows := []domain.Hierarchy{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.HierarchyColumns+` FROM dbo.Hierarchy WHERE rbd_system_id = $1 ORDER BY level DESC, hierarchy_id`, rbdSystemId)
}
