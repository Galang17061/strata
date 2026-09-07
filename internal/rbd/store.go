package rbd

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/Galang17061/strata-api/internal/domain"
)

type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
}

type Store struct {
	db *sqlx.DB
	q  Querier
}

func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db, q: db}
}

func (s *Store) Transact(ctx context.Context, fn func(tx *Store) error) error {
	if _, already := s.q.(*sqlx.Tx); already {
		return fn(s)
	}
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(&Store{db: s.db, q: tx}); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *Store) inQuery(query string, args ...any) (string, []any, error) {
	expanded, expandedArgs, err := sqlx.In(query, args...)
	if err != nil {
		return "", nil, err
	}
	return s.db.Rebind(expanded), expandedArgs, nil
}

func optional[T any](value *T, err error) (*T, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (s *Store) FindSystem(ctx context.Context, rbdSystemId string) (*domain.RbdSystemDrawing, error) {
	var row domain.RbdSystemDrawing
	err := s.q.GetContext(ctx, &row, `SELECT `+domain.RbdSystemColumns+` FROM dbo.RbdSystemDrawing WHERE rbd_system_id = $1`, rbdSystemId)
	return optional(&row, err)
}

func (s *Store) FirstSystem(ctx context.Context) (*domain.RbdSystemDrawing, error) {
	var row domain.RbdSystemDrawing
	err := s.q.GetContext(ctx, &row, `SELECT `+domain.RbdSystemColumns+` FROM dbo.RbdSystemDrawing ORDER BY rbd_system_id LIMIT 1`)
	return optional(&row, err)
}

func (s *Store) ListSystemViews(ctx context.Context, projectId *string) ([]domain.SystemView, error) {
	rows := []domain.SystemView{}
	query := `SELECT r.rbd_system_id, r.system_name, r.project_id, p.project_name, r.drawing_name FROM dbo.RbdSystemDrawing r INNER JOIN dbo.MasterProject p ON p.project_id = r.project_id`
	if projectId == nil {
		return rows, s.q.SelectContext(ctx, &rows, query+` ORDER BY r.rbd_system_id`)
	}
	return rows, s.q.SelectContext(ctx, &rows, query+` WHERE r.project_id = $1 ORDER BY r.rbd_system_id`, *projectId)
}

func (s *Store) SystemIds(ctx context.Context) ([]string, error) {
	ids := []string{}
	return ids, s.q.SelectContext(ctx, &ids, `SELECT rbd_system_id FROM dbo.RbdSystemDrawing WHERE rbd_system_id IS NOT NULL ORDER BY rbd_system_id`)
}

func (s *Store) InsertSystem(ctx context.Context, r domain.RbdSystemDrawing) error {
	_, err := s.q.ExecContext(ctx, `INSERT INTO dbo.RbdSystemDrawing (`+domain.RbdSystemColumns+`) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		r.RbdSystemId, r.ProjectId, r.DrawingName, r.SystemName, r.RunningHours, r.ReliabilityTotal, r.Formula, r.CreatedAt, r.UpdatedAt, r.CreatedBy, r.UpdatedBy)
	return err
}

func (s *Store) UpdateSystem(ctx context.Context, r domain.RbdSystemDrawing) error {
	_, err := s.q.ExecContext(ctx, `UPDATE dbo.RbdSystemDrawing SET project_id = $2, drawing_name = $3, system_name = $4, running_hours = $5, reliability_total = $6, formula = $7, updated_at = $8, updated_by = $9 WHERE rbd_system_id = $1`,
		r.RbdSystemId, r.ProjectId, r.DrawingName, r.SystemName, r.RunningHours, r.ReliabilityTotal, r.Formula, r.UpdatedAt, r.UpdatedBy)
	return err
}

func (s *Store) DeleteSystem(ctx context.Context, rbdSystemId string) error {
	_, err := s.q.ExecContext(ctx, `DELETE FROM dbo.RbdSystemDrawing WHERE rbd_system_id = $1`, rbdSystemId)
	return err
}

func (s *Store) FindProject(ctx context.Context, projectId string) (*domain.MasterProject, error) {
	var row domain.MasterProject
	err := s.q.GetContext(ctx, &row, `SELECT project_id, project_name, hierarchy_depth, created_at, updated_at, created_by, updated_by FROM dbo.MasterProject WHERE project_id = $1`, projectId)
	return optional(&row, err)
}

func (s *Store) TouchSystem(ctx context.Context, rbdSystemId string) error {
	if rbdSystemId == "" {
		return nil
	}
	_, err := s.q.ExecContext(ctx, `UPDATE dbo.RbdSystemDrawing SET updated_at = $2 WHERE rbd_system_id = $1`, rbdSystemId, domain.Now())
	return err
}

func (s *Store) TouchSystemOfHierarchy(ctx context.Context, hierarchyId string) error {
	hierarchy, err := s.FindHierarchy(ctx, hierarchyId)
	if err != nil || hierarchy == nil {
		return err
	}
	return s.TouchSystem(ctx, domain.Deref(hierarchy.RbdSystemId))
}

func (s *Store) TouchSystemOfComponent(ctx context.Context, systemComponentId string) error {
	component, err := s.FindComponent(ctx, systemComponentId)
	if err != nil || component == nil {
		return err
	}
	return s.TouchSystem(ctx, domain.Deref(component.RbdSystemId))
}

func (s *Store) ProjectDepthOfSystem(ctx context.Context, rbdSystemId string) (int, error) {
	var depth int
	err := s.q.GetContext(ctx, &depth, `SELECT p.hierarchy_depth FROM dbo.RbdSystemDrawing r INNER JOIN dbo.MasterProject p ON p.project_id = r.project_id WHERE r.rbd_system_id = $1`, rbdSystemId)
	if errors.Is(err, sql.ErrNoRows) {
		return 3, nil
	}
	if err != nil {
		return 3, err
	}
	return depth, nil
}

func (s *Store) FindHierarchy(ctx context.Context, hierarchyId string) (*domain.Hierarchy, error) {
	var row domain.Hierarchy
	err := s.q.GetContext(ctx, &row, `SELECT `+domain.HierarchyColumns+` FROM dbo.Hierarchy WHERE hierarchy_id = $1`, hierarchyId)
	return optional(&row, err)
}

func (s *Store) HierarchiesOfSystem(ctx context.Context, rbdSystemId string) ([]domain.Hierarchy, error) {
	rows := []domain.Hierarchy{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.HierarchyColumns+` FROM dbo.Hierarchy WHERE rbd_system_id = $1 ORDER BY hierarchy_id`, rbdSystemId)
}

func (s *Store) HierarchiesOfSystemByLevel(ctx context.Context, rbdSystemId string, includeVirtual bool) ([]domain.Hierarchy, error) {
	rows := []domain.Hierarchy{}
	query := `SELECT ` + domain.HierarchyColumns + ` FROM dbo.Hierarchy WHERE rbd_system_id = $1`
	if !includeVirtual {
		query += ` AND level <> 999`
	}
	return rows, s.q.SelectContext(ctx, &rows, query+` ORDER BY level, hierarchy_id`, rbdSystemId)
}

func (s *Store) ChildHierarchies(ctx context.Context, parentId string) ([]domain.Hierarchy, error) {
	rows := []domain.Hierarchy{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.HierarchyColumns+` FROM dbo.Hierarchy WHERE parent_id = $1 ORDER BY hierarchy_id`, parentId)
}

func (s *Store) LastHierarchyId(ctx context.Context) (*string, error) {
	var id string
	err := s.q.GetContext(ctx, &id, `SELECT hierarchy_id FROM dbo.Hierarchy ORDER BY hierarchy_id DESC LIMIT 1`)
	return optional(&id, err)
}

func (s *Store) HierarchyIds(ctx context.Context) ([]string, error) {
	ids := []string{}
	return ids, s.q.SelectContext(ctx, &ids, `SELECT hierarchy_id FROM dbo.Hierarchy WHERE hierarchy_id IS NOT NULL ORDER BY hierarchy_id`)
}

func (s *Store) HierarchyCodesOfSystem(ctx context.Context, rbdSystemId, prefix string) ([]string, error) {
	codes := []string{}
	return codes, s.q.SelectContext(ctx, &codes, `SELECT formula_code FROM dbo.Hierarchy WHERE rbd_system_id = $1 AND formula_code IS NOT NULL AND formula_code LIKE $2 + '%' ORDER BY hierarchy_id`, rbdSystemId, prefix)
}

func (s *Store) HierarchyCodeExists(ctx context.Context, rbdSystemId, code string) (bool, error) {
	var count int
	err := s.q.GetContext(ctx, &count, `SELECT COUNT(1) FROM dbo.Hierarchy WHERE rbd_system_id = $1 AND formula_code = $2`, rbdSystemId, code)
	return count > 0, err
}

func (s *Store) InsertHierarchies(ctx context.Context, rows []domain.Hierarchy) error {
	for _, h := range rows {
		if _, err := s.q.ExecContext(ctx, `INSERT INTO dbo.Hierarchy (`+domain.HierarchyColumns+`) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
			h.HierarchyId, h.RbdSystemId, h.ParentId, h.Level, h.SubSystemName, h.Formula, h.FormulaCode, h.ConnectionType, h.RealibilityValue, h.RunningHours, h.PositionX, h.PositionY, h.SourceId, h.TargetId); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) UpdateHierarchy(ctx context.Context, h domain.Hierarchy) error {
	_, err := s.q.ExecContext(ctx, `UPDATE dbo.Hierarchy SET rbd_system_id = $2, parent_id = $3, level = $4, sub_system_name = $5, formula = $6, formula_code = $7, connection_type = $8, realibility_value = $9, running_hours = $10, position_x = $11, position_y = $12, source_id = $13, target_id = $14 WHERE hierarchy_id = $1`,
		h.HierarchyId, h.RbdSystemId, h.ParentId, h.Level, h.SubSystemName, h.Formula, h.FormulaCode, h.ConnectionType, h.RealibilityValue, h.RunningHours, h.PositionX, h.PositionY, h.SourceId, h.TargetId)
	return err
}

func (s *Store) DeleteHierarchies(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	query, args, err := s.inQuery(`DELETE FROM dbo.ReliabilityHistory WHERE hierarchy_id IN (?)`, ids)
	if err != nil {
		return err
	}
	if _, err := s.q.ExecContext(ctx, query, args...); err != nil {
		return err
	}
	query, args, err = s.inQuery(`DELETE FROM dbo.Hierarchy WHERE hierarchy_id IN (?)`, ids)
	if err != nil {
		return err
	}
	_, err = s.q.ExecContext(ctx, query, args...)
	return err
}

func (s *Store) FindComponent(ctx context.Context, systemComponentId string) (*domain.SystemComponentProperties, error) {
	var row domain.SystemComponentProperties
	err := s.q.GetContext(ctx, &row, `SELECT `+domain.SystemComponentColumns+` FROM dbo.SystemComponentProperties WHERE system_component_id = $1`, systemComponentId)
	return optional(&row, err)
}

func (s *Store) ComponentsOfSystem(ctx context.Context, rbdSystemId string) ([]domain.SystemComponentProperties, error) {
	rows := []domain.SystemComponentProperties{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.SystemComponentColumns+` FROM dbo.SystemComponentProperties WHERE rbd_system_id = $1 ORDER BY system_component_id`, rbdSystemId)
}

func (s *Store) ComponentsOfParent(ctx context.Context, parentId string, activeOnly bool) ([]domain.SystemComponentProperties, error) {
	rows := []domain.SystemComponentProperties{}
	query := `SELECT ` + domain.SystemComponentColumns + ` FROM dbo.SystemComponentProperties WHERE parent_id = $1`
	if activeOnly {
		query += ` AND active = 1`
	}
	return rows, s.q.SelectContext(ctx, &rows, query+` ORDER BY system_component_id`, parentId)
}

func (s *Store) LastComponentId(ctx context.Context) (*string, error) {
	var id string
	err := s.q.GetContext(ctx, &id, `SELECT system_component_id FROM dbo.SystemComponentProperties ORDER BY system_component_id DESC LIMIT 1`)
	return optional(&id, err)
}

func (s *Store) ComponentIds(ctx context.Context) ([]string, error) {
	ids := []string{}
	return ids, s.q.SelectContext(ctx, &ids, `SELECT system_component_id FROM dbo.SystemComponentProperties WHERE system_component_id IS NOT NULL ORDER BY system_component_id`)
}

func (s *Store) ComponentCodesStartingWith(ctx context.Context, prefix string) ([]string, error) {
	codes := []string{}
	return codes, s.q.SelectContext(ctx, &codes, `SELECT formula_code FROM dbo.SystemComponentProperties WHERE formula_code IS NOT NULL AND formula_code LIKE $1 + '%' ORDER BY system_component_id`, prefix)
}

func (s *Store) ComponentCodeExists(ctx context.Context, rbdSystemId, code string) (bool, error) {
	var count int
	err := s.q.GetContext(ctx, &count, `SELECT COUNT(1) FROM dbo.SystemComponentProperties WHERE rbd_system_id = $1 AND formula_code = $2`, rbdSystemId, code)
	return count > 0, err
}

func (s *Store) InsertComponents(ctx context.Context, rows []domain.SystemComponentProperties) error {
	for _, c := range rows {
		if _, err := s.q.ExecContext(ctx, `INSERT INTO dbo.SystemComponentProperties (`+domain.SystemComponentColumns+`) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30)`,
			c.SystemComponentId, c.RbdSystemId, domain.Deref(c.ParentId), c.ComponentName, c.ComponentTagNumber, c.Active, c.Vendor, c.FormulaCode, c.DistributionType, c.FailureRate, c.RunningHours, c.ScaleParameter, c.ShapeParameter, c.ConnectionType, c.ConnectionToId, c.PositionX, c.PositionY, c.SourcePosition, c.TargetPosition, c.IdNode, c.ReliabilityValue, c.ActiveComponent, c.TotalComponent, c.Regresi, c.Mtbf, domain.DerefInt(c.AllowedFailures, 0), c.CreatedAt, c.UpdatedAt, c.CreatedBy, c.UpdatedBy); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) UpdateComponent(ctx context.Context, c domain.SystemComponentProperties) error {
	_, err := s.q.ExecContext(ctx, `UPDATE dbo.SystemComponentProperties SET rbd_system_id = $2, parent_id = $3, component_name = $4, component_tag_number = $5, active = $6, vendor = $7, formula_code = $8, distribution_type = $9, failure_rate = $10, running_hours = $11, scale_parameter = $12, shape_parameter = $13, connection_type = $14, connection_to_id = $15, position_x = $16, position_y = $17, source_position = $18, target_position = $19, id_node = $20, reliability_value = $21, active_component = $22, total_component = $23, regresi = $24, mtbf = $25, allowed_failures = $28, updated_at = $26, updated_by = $27 WHERE system_component_id = $1`,
		c.SystemComponentId, c.RbdSystemId, domain.Deref(c.ParentId), c.ComponentName, c.ComponentTagNumber, c.Active, c.Vendor, c.FormulaCode, c.DistributionType, c.FailureRate, c.RunningHours, c.ScaleParameter, c.ShapeParameter, c.ConnectionType, c.ConnectionToId, c.PositionX, c.PositionY, c.SourcePosition, c.TargetPosition, c.IdNode, c.ReliabilityValue, c.ActiveComponent, c.TotalComponent, c.Regresi, c.Mtbf, c.UpdatedAt, c.UpdatedBy, domain.DerefInt(c.AllowedFailures, 0))
	return err
}

func (s *Store) DeleteComponentDependents(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	statements := []string{
		`DELETE FROM dbo.FailureEventHistory WHERE system_component_id IN (?)`,
		`DELETE FROM dbo.ExponentialParameter WHERE system_component_id IN (?)`,
		`DELETE FROM dbo.WeibullParameter WHERE system_component_id IN (?)`,
		`DELETE FROM dbo.PoissonParameter WHERE system_component_id IN (?)`,
		`DELETE FROM dbo.ReliabilityPlotComponent WHERE system_component_id IN (?)`,
	}
	for _, statement := range statements {
		query, args, err := s.inQuery(statement, ids)
		if err != nil {
			return err
		}
		if _, err := s.q.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) DeleteComponents(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	query, args, err := s.inQuery(`DELETE FROM dbo.SystemComponentProperties WHERE system_component_id IN (?)`, ids)
	if err != nil {
		return err
	}
	_, err = s.q.ExecContext(ctx, query, args...)
	return err
}

func (s *Store) AllEdges(ctx context.Context) ([]domain.SystemComponentDrawing, error) {
	rows := []domain.SystemComponentDrawing{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT `+domain.EdgeColumns+` FROM dbo.SystemComponentDrawing ORDER BY id_edge`)
}

func (s *Store) EdgesTouching(ctx context.Context, nodeIds []string) ([]domain.SystemComponentDrawing, error) {
	rows := []domain.SystemComponentDrawing{}
	if len(nodeIds) == 0 {
		return rows, nil
	}
	query, args, err := s.inQuery(`SELECT `+domain.EdgeColumns+` FROM dbo.SystemComponentDrawing WHERE source_id IN (?) OR target_id IN (?) ORDER BY id_edge`, nodeIds, nodeIds)
	if err != nil {
		return nil, err
	}
	return rows, s.q.SelectContext(ctx, &rows, query, args...)
}

func (s *Store) DeleteEdgesTouching(ctx context.Context, nodeIds []string) error {
	if len(nodeIds) == 0 {
		return nil
	}
	query, args, err := s.inQuery(`DELETE FROM dbo.SystemComponentDrawing WHERE source_id IN (?) OR target_id IN (?)`, nodeIds, nodeIds)
	if err != nil {
		return err
	}
	_, err = s.q.ExecContext(ctx, query, args...)
	return err
}

func (s *Store) InsertEdges(ctx context.Context, rows []domain.SystemComponentDrawing) error {
	for _, edge := range rows {
		if _, err := s.q.ExecContext(ctx, `INSERT INTO dbo.SystemComponentDrawing (`+domain.EdgeColumns+`) VALUES ($1, $2, $3)`, edge.IdEdge, edge.SourceId, edge.TargetId); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) DeleteHistoryOfSystem(ctx context.Context, rbdSystemId string) error {
	_, err := s.q.ExecContext(ctx, `DELETE FROM dbo.ReliabilityHistory WHERE rbd_system_id = $1`, rbdSystemId)
	return err
}

func (s *Store) MasterComponentsWithVendor(ctx context.Context) ([]domain.MasterComponentWithVendor, error) {
	rows := []domain.MasterComponentWithVendor{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT c.component_id, c.component_name, c.vendor_id, c.failure_rate, c.cost, c.compatibility, c.serial_number, m.manufacturer_name FROM dbo.MasterComponent c INNER JOIN dbo.MasterManufacturer m ON m.vendor_id = c.vendor_id ORDER BY c.component_id`)
}

func (s *Store) MasterComponentsByName(ctx context.Context, name string) ([]domain.MasterComponentWithVendor, error) {
	rows := []domain.MasterComponentWithVendor{}
	return rows, s.q.SelectContext(ctx, &rows, `SELECT c.component_id, c.component_name, c.vendor_id, c.failure_rate, c.cost, c.compatibility, c.serial_number, m.manufacturer_name FROM dbo.MasterComponent c INNER JOIN dbo.MasterManufacturer m ON m.vendor_id = c.vendor_id WHERE c.component_name = $1 ORDER BY c.component_id`, name)
}

func (s *Store) MasterComponentByNameAndVendor(ctx context.Context, name string, vendor *string) (*domain.MasterComponentWithVendor, error) {
	var row domain.MasterComponentWithVendor
	query := `SELECT c.component_id, c.component_name, c.vendor_id, c.failure_rate, c.cost, c.compatibility, c.serial_number, m.manufacturer_name FROM dbo.MasterComponent c INNER JOIN dbo.MasterManufacturer m ON m.vendor_id = c.vendor_id WHERE c.component_name = $1 LIMIT 1`
	if vendor == nil {
		return optional(&row, s.q.GetContext(ctx, &row, query+` ORDER BY c.component_id`, name))
	}
	return optional(&row, s.q.GetContext(ctx, &row, query+` AND m.manufacturer_name = $2 ORDER BY c.component_id`, name, *vendor))
}
