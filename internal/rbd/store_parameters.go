package rbd

import (
	"context"

	"github.com/Galang17061/strata-api/internal/domain"
)

func (s *Store) FailureEventsOfComponent(ctx context.Context, systemComponentId, search string) ([]domain.FailureEventHistory, error) {
	rows := []domain.FailureEventHistory{}
	query := `SELECT ` + domain.FailureEventColumns + ` FROM dbo.FailureEventHistory WHERE system_component_id = $1`
	if search != "" {
		return rows, s.q.SelectContext(ctx, &rows, query+` AND (POSITION($2 IN failure_event_id) > 0 OR POSITION($2 IN system_component_id) > 0 OR POSITION($2 IN to_char(failure_date, 'YYYY-MM-DD')) > 0) ORDER BY failure_event_id`, systemComponentId, search)
	}
	return rows, s.q.SelectContext(ctx, &rows, query+` ORDER BY failure_event_id`, systemComponentId)
}

func (s *Store) FailureEventsByHours(ctx context.Context, systemComponentId string) ([]domain.FailureEventHistory, error) {
	rows := []domain.FailureEventHistory{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.FailureEventColumns+` FROM dbo.FailureEventHistory WHERE system_component_id = $1 ORDER BY running_hours, failure_event_id`, systemComponentId)
}

func (s *Store) FindFailureEvent(ctx context.Context, failureEventId string) (*domain.FailureEventHistory, error) {
	var row domain.FailureEventHistory
	err := s.q.GetContext(ctx, &row, `SELECT `+domain.FailureEventColumns+` FROM dbo.FailureEventHistory WHERE failure_event_id = $1`, failureEventId)
	return optional(&row, err)
}

func (s *Store) LastFailureEventId(ctx context.Context) (*string, error) {
	var id string
	err := s.q.GetContext(ctx, &id, `SELECT failure_event_id FROM dbo.FailureEventHistory ORDER BY failure_event_id DESC LIMIT 1`)
	return optional(&id, err)
}

func (s *Store) LastFailureNumber(ctx context.Context, systemComponentId string) (*int, error) {
	var number int
	err := s.q.GetContext(ctx, &number, `SELECT COALESCE(failure_number, 0) FROM dbo.FailureEventHistory WHERE system_component_id = $1 ORDER BY failure_number DESC LIMIT 1`, systemComponentId)
	return optional(&number, err)
}

func (s *Store) InsertFailureEvents(ctx context.Context, rows []domain.FailureEventHistory) error {
	for _, f := range rows {
		if _, err := s.q.ExecContext(ctx, `INSERT INTO dbo.FailureEventHistory (`+domain.FailureEventColumns+`) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			f.FailureEventId, f.SystemComponentId, f.FailureDate, f.FailureNumber, f.RunningHours, f.CreatedAt, f.UpdatedAt, f.CreatedBy, f.UpdatedBy); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) DeleteFailureEvent(ctx context.Context, failureEventId string) error {
	_, err := s.q.ExecContext(ctx, `DELETE FROM dbo.FailureEventHistory WHERE failure_event_id = $1`, failureEventId)
	return err
}

func (s *Store) DeleteFailureEventsOfComponent(ctx context.Context, systemComponentId string) error {
	_, err := s.q.ExecContext(ctx, `DELETE FROM dbo.FailureEventHistory WHERE system_component_id = $1`, systemComponentId)
	return err
}

func (s *Store) FailureEventsAfterNumber(ctx context.Context, systemComponentId string, number int) ([]domain.FailureEventHistory, error) {
	rows := []domain.FailureEventHistory{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.FailureEventColumns+` FROM dbo.FailureEventHistory WHERE system_component_id = $1 AND failure_number > $2 ORDER BY failure_number, failure_event_id`, systemComponentId, number)
}

func (s *Store) UpdateFailureNumber(ctx context.Context, failureEventId string, number int) error {
	_, err := s.q.ExecContext(ctx, `UPDATE dbo.FailureEventHistory SET failure_number = $2 WHERE failure_event_id = $1`, failureEventId, number)
	return err
}

func (s *Store) PoissonParametersOfComponent(ctx context.Context, systemComponentId string) ([]domain.PoissonParameter, error) {
	rows := []domain.PoissonParameter{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.PoissonColumns+` FROM dbo.PoissonParameter WHERE system_component_id = $1 ORDER BY poisson_parameter_id`, systemComponentId)
}

func (s *Store) PoissonParametersByHours(ctx context.Context, systemComponentId string) ([]domain.PoissonParameter, error) {
	rows := []domain.PoissonParameter{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.PoissonColumns+` FROM dbo.PoissonParameter WHERE system_component_id = $1 ORDER BY failure_event_hours, poisson_parameter_id`, systemComponentId)
}

func (s *Store) LastPoissonIdOfOthers(ctx context.Context, systemComponentId string) (*string, error) {
	var id string
	err := s.q.GetContext(ctx, &id, `SELECT poisson_parameter_id FROM dbo.PoissonParameter WHERE system_component_id <> $1 ORDER BY poisson_parameter_id DESC LIMIT 1`, systemComponentId)
	return optional(&id, err)
}

func (s *Store) DeletePoissonParametersOfComponent(ctx context.Context, systemComponentId string) error {
	_, err := s.q.ExecContext(ctx, `DELETE FROM dbo.PoissonParameter WHERE system_component_id = $1`, systemComponentId)
	return err
}

func (s *Store) InsertPoissonParameters(ctx context.Context, rows []domain.PoissonParameter) error {
	for _, p := range rows {
		if _, err := s.q.ExecContext(ctx, `INSERT INTO dbo.PoissonParameter (`+domain.PoissonColumns+`) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			p.PoissonParameterId, p.SystemComponentId, p.FailureEventHours, p.N, p.Rate, p.CreatedAt, p.UpdatedAt, p.CreatedBy, p.UpdatedBy); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ExponentialParametersOfComponent(ctx context.Context, systemComponentId string) ([]domain.ExponentialParameter, error) {
	rows := []domain.ExponentialParameter{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.ExponentialColumns+` FROM dbo.ExponentialParameter WHERE system_component_id = $1 ORDER BY exponential_parameter_id`, systemComponentId)
}

func (s *Store) WeibullParametersOfComponent(ctx context.Context, systemComponentId string) ([]domain.WeibullParameter, error) {
	rows := []domain.WeibullParameter{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.WeibullColumns+` FROM dbo.WeibullParameter WHERE system_component_id = $1 ORDER BY weibull_parameter_id`, systemComponentId)
}

func (s *Store) WeibullParametersByHours(ctx context.Context, systemComponentId string) ([]domain.WeibullParameter, error) {
	rows := []domain.WeibullParameter{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.WeibullColumns+` FROM dbo.WeibullParameter WHERE system_component_id = $1 ORDER BY failure_event_hours, weibull_parameter_id`, systemComponentId)
}

func (s *Store) LastWeibullIdOfOthers(ctx context.Context, systemComponentId string) (*string, error) {
	var id string
	err := s.q.GetContext(ctx, &id, `SELECT weibull_parameter_id FROM dbo.WeibullParameter WHERE system_component_id <> $1 ORDER BY weibull_parameter_id DESC LIMIT 1`, systemComponentId)
	return optional(&id, err)
}

func (s *Store) DeleteWeibullParametersOfComponent(ctx context.Context, systemComponentId string) error {
	_, err := s.q.ExecContext(ctx, `DELETE FROM dbo.WeibullParameter WHERE system_component_id = $1`, systemComponentId)
	return err
}

func (s *Store) InsertWeibullParameters(ctx context.Context, rows []domain.WeibullParameter) error {
	for _, w := range rows {
		if _, err := s.q.ExecContext(ctx, `INSERT INTO dbo.WeibullParameter (`+domain.WeibullColumns+`) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			w.WeibullParameterId, w.SystemComponentId, w.FailureEventHours, w.N, w.FreqF, w.X, w.Y, w.CreatedAt, w.UpdatedAt, w.CreatedBy, w.UpdatedBy); err != nil {
			return err
		}
	}
	return nil
}
