package rbd

import (
	"context"

	"github.com/Galang17061/strata-api/internal/domain"
)

func (s *Store) ProjectNameExists(ctx context.Context, name string) (bool, error) {
	var count int
	err := s.q.GetContext(ctx, &count, `SELECT COUNT(1) FROM dbo.MasterProject WHERE project_name = @p1`, name)
	return count > 0, err
}

func (s *Store) LastProjectId(ctx context.Context) (*string, error) {
	var id string
	err := s.q.GetContext(ctx, &id, `SELECT TOP 1 project_id FROM dbo.MasterProject ORDER BY LEN(project_id) DESC, project_id DESC`)
	return optional(&id, err)
}

func (s *Store) InsertProject(ctx context.Context, p domain.MasterProject) error {
	_, err := s.q.ExecContext(ctx, `INSERT INTO dbo.MasterProject (project_id, project_name, hierarchy_depth, created_at, updated_at, created_by, updated_by) VALUES (@p1, @p2, @p3, @p4, @p5, @p6, @p7)`,
		p.ProjectId, p.ProjectName, p.HierarchyDepth, p.CreatedAt, p.UpdatedAt, p.CreatedBy, p.UpdatedBy)
	return err
}
