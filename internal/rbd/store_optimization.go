package rbd

import (
	"context"

	"github.com/Galang17061/strata-api/internal/domain"
)

func (s *Store) ProjectNameExists(ctx context.Context, name string) (bool, error) {
	var count int
	err := s.q.GetContext(ctx, &count, `SELECT COUNT(1) FROM dbo.MasterProject WHERE project_name = $1`, name)
	return count > 0, err
}

func (s *Store) LastProjectId(ctx context.Context) (*string, error) {
	var id string
	err := s.q.GetContext(ctx, &id, `SELECT project_id FROM dbo.MasterProject ORDER BY LENGTH(project_id) DESC, project_id DESC LIMIT 1`)
	return optional(&id, err)
}

func (s *Store) InsertOptimizationRun(ctx context.Context, run domain.OptimizationRun) error {
	_, err := s.q.ExecContext(ctx, `INSERT INTO dbo.OptimizationRun (optimization_run_id, rbd_system_id, mode, max_budget, target_reliability, weight_cost, weight_reliability, running_hours, population_size, max_generations, crossover_probability, mutation_probability, seed, choices, result_project_id, result_rbd_system_id, created_at, created_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`,
		run.OptimizationRunId, run.RbdSystemId, run.Mode, run.MaxBudget, run.TargetReliability, run.WeightCost, run.WeightReliability, run.RunningHours, run.PopulationSize, run.MaxGenerations, run.CrossoverProbability, run.MutationProbability, run.Seed, run.Choices, run.ResultProjectId, run.ResultRbdSystemId, run.CreatedAt, run.CreatedBy)
	return err
}

func (s *Store) InsertProject(ctx context.Context, p domain.MasterProject) error {
	_, err := s.q.ExecContext(ctx, `INSERT INTO dbo.MasterProject (project_id, project_name, hierarchy_depth, created_at, updated_at, created_by, updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		p.ProjectId, p.ProjectName, p.HierarchyDepth, p.CreatedAt, p.UpdatedAt, p.CreatedBy, p.UpdatedBy)
	return err
}
