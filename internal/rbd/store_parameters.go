package rbd

import (
	"context"

	"github.com/Galang17061/strata-api/internal/domain"
)

func (s *Store) FailureEventsOfComponent(ctx context.Context, systemComponentId, search string) ([]domain.FailureEventHistory, error) {
	rows := []domain.FailureEventHistory{}
	query := `SELECT ` + domain.FailureEventColumns + ` FROM dbo.FailureEventHistory WHERE system_component_id = @p1`
	if search != "" {
		return rows, s.q.SelectContext(ctx, &rows, query+` AND (CHARINDEX(@p2, failure_event_id) > 0 OR CHARINDEX(@p2, system_component_id) > 0 OR CHARINDEX(@p2, CONVERT(varchar(10), failure_date, 23)) > 0) ORDER BY failure_event_id`, systemComponentId, search)
	}
	return rows, s.q.SelectContext(ctx, &rows, query+` ORDER BY failure_event_id`, systemComponentId)
}

func (s *Store) FailureEventsByHours(ctx context.Context, systemComponentId string) ([]domain.FailureEventHistory, error) {
	rows := []domain.FailureEventHistory{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.FailureEventColumns+` FROM dbo.FailureEventHistory WHERE system_component_id = @p1 ORDER BY running_hours, failure_event_id`, systemComponentId)
}

func (s *Store) FindFailureEvent(ctx context.Context, failureEventId string) (*domain.FailureEventHistory, error) {
	var row domain.FailureEventHistory
	err := s.q.GetContext(ctx, &row, `SELECT `+domain.FailureEventColumns+` FROM dbo.FailureEventHistory WHERE failure_event_id = @p1`, failureEventId)
	return optional(&row, err)
}

func (s *Store) LastFailureEventId(ctx context.Context) (*string, error) {
	var id string
	err := s.q.GetContext(ctx, &id, `SELECT TOP 1 failure_event_id FROM dbo.FailureEventHistory ORDER BY failure_event_id DESC`)
	return optional(&id, err)
}

func (s *Store) LastFailureNumber(ctx context.Context, systemComponentId string) (*int, error) {
	var number int
	err := s.q.GetContext(ctx, &number, `SELECT TOP 1 ISNULL(failure_number, 0) FROM dbo.FailureEventHistory WHERE system_component_id = @p1 ORDER BY failure_number DESC`, systemComponentId)
	return optional(&number, err)
}

func (s *Store) InsertFailureEvents(ctx context.Context, rows []domain.FailureEventHistory) error {
	for _, f := range rows {
		if _, err := s.q.ExecContext(ctx, `INSERT INTO dbo.FailureEventHistory (`+domain.FailureEventColumns+`) VALUES (@p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8, @p9)`,
			f.FailureEventId, f.SystemComponentId, f.FailureDate, f.FailureNumber, f.RunningHours, f.CreatedAt, f.UpdatedAt, f.CreatedBy, f.UpdatedBy); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) DeleteFailureEvent(ctx context.Context, failureEventId string) error {
	_, err := s.q.ExecContext(ctx, `DELETE FROM dbo.FailureEventHistory WHERE failure_event_id = @p1`, failureEventId)
	return err
}

func (s *Store) DeleteFailureEventsOfComponent(ctx context.Context, systemComponentId string) error {
	_, err := s.q.ExecContext(ctx, `DELETE FROM dbo.FailureEventHistory WHERE system_component_id = @p1`, systemComponentId)
	return err
}

func (s *Store) FailureEventsAfterNumber(ctx context.Context, systemComponentId string, number int) ([]domain.FailureEventHistory, error) {
	rows := []domain.FailureEventHistory{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.FailureEventColumns+` FROM dbo.FailureEventHistory WHERE system_component_id = @p1 AND failure_number > @p2 ORDER BY failure_number, failure_event_id`, systemComponentId, number)
}

func (s *Store) UpdateFailureNumber(ctx context.Context, failureEventId string, number int) error {
	_, err := s.q.ExecContext(ctx, `UPDATE dbo.FailureEventHistory SET failure_number = @p2 WHERE failure_event_id = @p1`, failureEventId, number)
	return err
}

func (s *Store) ExponentialParametersOfComponent(ctx context.Context, systemComponentId string) ([]domain.ExponentialParameter, error) {
	rows := []domain.ExponentialParameter{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.ExponentialColumns+` FROM dbo.ExponentialParameter WHERE system_component_id = @p1 ORDER BY exponential_parameter_id`, systemComponentId)
}

func (s *Store) WeibullParametersOfComponent(ctx context.Context, systemComponentId string) ([]domain.WeibullParameter, error) {
	rows := []domain.WeibullParameter{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.WeibullColumns+` FROM dbo.WeibullParameter WHERE system_component_id = @p1 ORDER BY weibull_parameter_id`, systemComponentId)
}

func (s *Store) WeibullParametersByHours(ctx context.Context, systemComponentId string) ([]domain.WeibullParameter, error) {
	rows := []domain.WeibullParameter{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.WeibullColumns+` FROM dbo.WeibullParameter WHERE system_component_id = @p1 ORDER BY failure_event_hours, weibull_parameter_id`, systemComponentId)
}

func (s *Store) LastWeibullIdOfOthers(ctx context.Context, systemComponentId string) (*string, error) {
	var id string
	err := s.q.GetContext(ctx, &id, `SELECT TOP 1 weibull_parameter_id FROM dbo.WeibullParameter WHERE system_component_id <> @p1 ORDER BY weibull_parameter_id DESC`, systemComponentId)
	return optional(&id, err)
}

func (s *Store) DeleteWeibullParametersOfComponent(ctx context.Context, systemComponentId string) error {
	_, err := s.q.ExecContext(ctx, `DELETE FROM dbo.WeibullParameter WHERE system_component_id = @p1`, systemComponentId)
	return err
}

func (s *Store) InsertWeibullParameters(ctx context.Context, rows []domain.WeibullParameter) error {
	for _, w := range rows {
		if _, err := s.q.ExecContext(ctx, `INSERT INTO dbo.WeibullParameter (`+domain.WeibullColumns+`) VALUES (@p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8, @p9, @p10, @p11)`,
			w.WeibullParameterId, w.SystemComponentId, w.FailureEventHours, w.N, w.FreqF, w.X, w.Y, w.CreatedAt, w.UpdatedAt, w.CreatedBy, w.UpdatedBy); err != nil {
			return err
		}
	}
	return nil
}
