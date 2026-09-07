package rbd

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Galang17061/strata-api/internal/domain"
)

type pooledFailureStats struct {
	Components int     `db:"components"`
	Events     int     `db:"events"`
	Hours      float64 `db:"hours"`
}

func (s *Store) PooledFailureStatsOfComponent(ctx context.Context, componentName, vendor string) (pooledFailureStats, error) {
	var stats pooledFailureStats
	err := s.q.GetContext(ctx, &stats, `SELECT COUNT(*) AS components, COALESCE(SUM(sub.n), 0) AS events, COALESCE(SUM(sub.hours), 0) AS hours FROM (SELECT f.system_component_id, COUNT(*) AS n, MAX(f.running_hours) AS hours FROM dbo.FailureEventHistory f INNER JOIN dbo.SystemComponentProperties scp ON scp.system_component_id = f.system_component_id WHERE scp.component_name = $1 AND COALESCE(scp.vendor, N'') = $2 GROUP BY f.system_component_id) sub`, componentName, vendor)
	return stats, err
}

func (s *Store) PooledFailureStatsOfVendor(ctx context.Context, vendor string) (pooledFailureStats, error) {
	var stats pooledFailureStats
	err := s.q.GetContext(ctx, &stats, `SELECT COUNT(*) AS components, COALESCE(SUM(sub.n), 0) AS events, COALESCE(SUM(sub.hours), 0) AS hours FROM (SELECT f.system_component_id, COUNT(*) AS n, MAX(f.running_hours) AS hours FROM dbo.FailureEventHistory f INNER JOIN dbo.SystemComponentProperties scp ON scp.system_component_id = f.system_component_id WHERE COALESCE(scp.vendor, N'') = $1 GROUP BY f.system_component_id) sub`, vendor)
	return stats, err
}

func (s *Store) PooledFailureHours(ctx context.Context, componentName, vendor string) ([]int, error) {
	hours := []int{}
	err := s.q.SelectContext(ctx, &hours, `SELECT f.running_hours FROM dbo.FailureEventHistory f INNER JOIN dbo.SystemComponentProperties scp ON scp.system_component_id = f.system_component_id WHERE scp.component_name = $1 AND COALESCE(scp.vendor, N'') = $2 ORDER BY f.running_hours`, componentName, vendor)
	return hours, err
}

func (s *Store) MasterFailureRateFor(ctx context.Context, componentName, vendorName string) (*domain.Number, error) {
	var rate *domain.Number
	err := s.q.GetContext(ctx, &rate, `SELECT c.failure_rate FROM dbo.MasterComponent c INNER JOIN dbo.MasterManufacturer m ON m.vendor_id = c.vendor_id WHERE c.component_name = $1 AND m.manufacturer_name = $2 ORDER BY c.component_id LIMIT 1`, componentName, vendorName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return rate, err
}
