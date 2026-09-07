package rbd

import (
	"context"

	"github.com/Galang17061/strata-api/internal/domain"
)

type plotRow struct {
	TimeT             int            `db:"time_t"`
	SystemComponentId string         `db:"system_component_id"`
	ComponentName     string         `db:"component_name"`
	FormulaCode       *string        `db:"formula_code"`
	ReliabilityComp   *domain.Number `db:"reliability_comp"`
}

func (s *Store) PlotRowsOfSystem(ctx context.Context, rbdSystemId string) ([]plotRow, error) {
	rows := []plotRow{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT p.time_t, p.system_component_id, c.component_name, c.formula_code, p.reliability_comp FROM dbo.ReliabilityPlotComponent p INNER JOIN dbo.SystemComponentProperties c ON c.system_component_id = p.system_component_id WHERE c.rbd_system_id = $1 ORDER BY p.reliability_plot_id`, rbdSystemId)
}

func (s *Store) LastPlotId(ctx context.Context) (*string, error) {
	var id string
	err := s.q.GetContext(ctx, &id, `SELECT reliability_plot_id FROM dbo.ReliabilityPlotComponent ORDER BY reliability_plot_id DESC LIMIT 1`)
	return optional(&id, err)
}

func (s *Store) DeletePlotsOfComponent(ctx context.Context, systemComponentId string) error {
	_, err := s.q.ExecContext(ctx, `DELETE FROM dbo.ReliabilityPlotComponent WHERE system_component_id = $1`, systemComponentId)
	return err
}

func (s *Store) InsertPlots(ctx context.Context, rows []domain.ReliabilityPlotComponent) error {
	for _, p := range rows {
		if _, err := s.q.ExecContext(ctx, `INSERT INTO dbo.ReliabilityPlotComponent (`+domain.PlotColumns+`) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			p.ReliabilityPlotId, p.SystemComponentId, p.TimeT, p.ReliabilityComp, p.CreatedAt, p.UpdatedAt, p.CreatedBy, p.UpdatedBy); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) AllPlots(ctx context.Context) ([]domain.ReliabilityPlotComponent, error) {
	rows := []domain.ReliabilityPlotComponent{}
	if err := s.q.SelectContext(ctx, &rows, `SELECT `+domain.PlotColumns+` FROM dbo.ReliabilityPlotComponent ORDER BY reliability_plot_id`); err != nil {
		return nil, err
	}
	return s.attachPlotComponents(ctx, rows)
}

func (s *Store) FindPlot(ctx context.Context, reliabilityPlotId string) (*domain.ReliabilityPlotComponent, error) {
	var row domain.ReliabilityPlotComponent
	found, err := optional(&row, s.q.GetContext(ctx, &row, `SELECT `+domain.PlotColumns+` FROM dbo.ReliabilityPlotComponent WHERE reliability_plot_id = $1`, reliabilityPlotId))
	if err != nil || found == nil {
		return found, err
	}
	rows, err := s.attachPlotComponents(ctx, []domain.ReliabilityPlotComponent{*found})
	if err != nil {
		return nil, err
	}
	return &rows[0], nil
}

func (s *Store) attachPlotComponents(ctx context.Context, rows []domain.ReliabilityPlotComponent) ([]domain.ReliabilityPlotComponent, error) {
	cache := map[string]*domain.SystemComponentProperties{}
	for index := range rows {
		id := rows[index].SystemComponentId
		component, cached := cache[id]
		if !cached {
			found, err := s.FindComponent(ctx, id)
			if err != nil {
				return nil, err
			}
			component = found
			cache[id] = found
		}
		rows[index].SystemComponentProperties = component
	}
	return rows, nil
}

func (s *Store) UpdatePlot(ctx context.Context, p domain.ReliabilityPlotComponent) error {
	_, err := s.q.ExecContext(ctx, `UPDATE dbo.ReliabilityPlotComponent SET system_component_id = $2, time_t = $3, reliability_comp = $4, updated_at = $5, updated_by = $6 WHERE reliability_plot_id = $1`,
		p.ReliabilityPlotId, p.SystemComponentId, p.TimeT, p.ReliabilityComp, p.UpdatedAt, p.UpdatedBy)
	return err
}

func (s *Store) DeletePlot(ctx context.Context, reliabilityPlotId string) error {
	_, err := s.q.ExecContext(ctx, `DELETE FROM dbo.ReliabilityPlotComponent WHERE reliability_plot_id = $1`, reliabilityPlotId)
	return err
}
