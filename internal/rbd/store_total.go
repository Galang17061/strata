package rbd

import (
	"context"
	"strings"

	"github.com/Galang17061/strata-api/internal/domain"
)

func (s *Store) HierarchyLevelsOfSystem(ctx context.Context, rbdSystemId string) ([]int, error) {
	levels := []int{}
	return levels, s.q.SelectContext(ctx, &levels, `SELECT level FROM dbo.Hierarchy WHERE rbd_system_id = @p1 GROUP BY level ORDER BY level`, rbdSystemId)
}

func (s *Store) ChildrenExcludingSelf(ctx context.Context, hierarchyId string) ([]domain.Hierarchy, error) {
	rows := []domain.Hierarchy{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.HierarchyColumns+` FROM dbo.Hierarchy WHERE parent_id = @p1 AND hierarchy_id <> @p1 ORDER BY hierarchy_id`, hierarchyId)
}

func (s *Store) ActiveComponentsWithValue(ctx context.Context, rbdSystemId string) ([]domain.SystemComponentProperties, error) {
	rows := []domain.SystemComponentProperties{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.SystemComponentColumns+` FROM dbo.SystemComponentProperties WHERE rbd_system_id = @p1 AND active = 1 AND reliability_value IS NOT NULL ORDER BY system_component_id`, rbdSystemId)
}

func (s *Store) ActiveComponentsOfParentWithValue(ctx context.Context, parentId string) ([]domain.SystemComponentProperties, error) {
	rows := []domain.SystemComponentProperties{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.SystemComponentColumns+` FROM dbo.SystemComponentProperties WHERE parent_id = @p1 AND active = 1 AND reliability_value IS NOT NULL ORDER BY system_component_id`, parentId)
}

func (s *Store) InsertHistory(ctx context.Context, h domain.ReliabilityHistory) error {
	_, err := s.q.ExecContext(ctx, `INSERT INTO dbo.ReliabilityHistory (`+domain.HistoryColumns+`) VALUES (@p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8, @p9, @p10, @p11, @p12, @p13)`,
		h.HistoryId, h.RbdSystemId, h.HierarchyId, h.HierarchyName, h.HierarchyLevel, h.FormulaCode, h.Formula, h.CalculatedReliability, h.ReliabilityLookup, h.ComponentDetails, h.RunningHours, h.CalculationTimestamp, h.CalculatedBy)
	return err
}

func (s *Store) HistoryCount(ctx context.Context, hierarchyId string) (int, error) {
	var count int
	err := s.q.GetContext(ctx, &count, `SELECT COUNT(1) FROM dbo.ReliabilityHistory WHERE hierarchy_id = @p1`, hierarchyId)
	return count, err
}

func historyOrder(sortBy, sortOrder string) string {
	direction := "ASC"
	if strings.ToLower(sortOrder) == "desc" {
		direction = "DESC"
	}
	switch strings.ToLower(sortBy) {
	case "calculationtimestamp":
		return "calculation_timestamp " + direction
	case "calculatedreliability":
		return "calculated_reliability " + direction
	case "runninghours":
		return "running_hours " + direction
	case "hierarchyname":
		return "hierarchy_name " + direction
	default:
		return "calculation_timestamp DESC"
	}
}

func (s *Store) HistoryPage(ctx context.Context, hierarchyId string, page, pageSize int, sortBy, sortOrder string) ([]domain.ReliabilityHistory, error) {
	rows := []domain.ReliabilityHistory{}
	query := `SELECT ` + domain.HistoryColumns + ` FROM dbo.ReliabilityHistory WHERE hierarchy_id = @p1 ORDER BY ` + historyOrder(sortBy, sortOrder) + ` OFFSET @p2 ROWS FETCH NEXT @p3 ROWS ONLY`
	return rows, s.q.SelectContext(ctx, &rows, query, hierarchyId, (page-1)*pageSize, pageSize)
}

func (s *Store) FindHistory(ctx context.Context, historyId string) (*domain.ReliabilityHistory, error) {
	var row domain.ReliabilityHistory
	err := s.q.GetContext(ctx, &row, `SELECT `+domain.HistoryColumns+` FROM dbo.ReliabilityHistory WHERE history_id = @p1`, historyId)
	return optional(&row, err)
}

func (s *Store) DeleteHistory(ctx context.Context, historyId string) error {
	_, err := s.q.ExecContext(ctx, `DELETE FROM dbo.ReliabilityHistory WHERE history_id = @p1`, historyId)
	return err
}
