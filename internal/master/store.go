package master

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/Galang17061/strata-api/internal/domain"
)

type Store struct {
	db *sqlx.DB
}

func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

const manufacturerColumns = `vendor_id, manufacturer_name, logo_image, valid_until, created_at, updated_at, created_by, updated_by`

func (s *Store) ListManufacturers(ctx context.Context, search string) ([]domain.MasterManufacturer, error) {
	rows := []domain.MasterManufacturer{}
	query := `SELECT ` + manufacturerColumns + ` FROM dbo.MasterManufacturer`
	if search != "" {
		query += ` WHERE CHARINDEX(@p1, manufacturer_name) > 0 OR CHARINDEX(@p1, vendor_id) > 0`
		return rows, s.db.SelectContext(ctx, &rows, query+` ORDER BY vendor_id`, search)
	}
	return rows, s.db.SelectContext(ctx, &rows, query+` ORDER BY vendor_id`)
}

func (s *Store) FindManufacturer(ctx context.Context, vendorId string) (*domain.MasterManufacturer, error) {
	var row domain.MasterManufacturer
	err := s.db.GetContext(ctx, &row, `SELECT `+manufacturerColumns+` FROM dbo.MasterManufacturer WHERE vendor_id = @p1`, vendorId)
	return optional(&row, err)
}

func (s *Store) ManufacturerByName(ctx context.Context, name string) (*domain.MasterManufacturer, error) {
	var row domain.MasterManufacturer
	err := s.db.GetContext(ctx, &row, `SELECT TOP 1 `+manufacturerColumns+` FROM dbo.MasterManufacturer WHERE manufacturer_name = @p1 ORDER BY vendor_id`, name)
	return optional(&row, err)
}

func (s *Store) LastManufacturerId(ctx context.Context) (*string, error) {
	var id string
	err := s.db.GetContext(ctx, &id, `SELECT TOP 1 vendor_id FROM dbo.MasterManufacturer ORDER BY vendor_id DESC`)
	return optional(&id, err)
}

func (s *Store) InsertManufacturer(ctx context.Context, m domain.MasterManufacturer) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO dbo.MasterManufacturer (`+manufacturerColumns+`) VALUES (@p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8)`,
		m.VendorId, m.ManufacturerName, m.LogoImage, m.ValidUntil, m.CreatedAt, m.UpdatedAt, m.CreatedBy, m.UpdatedBy)
	return err
}

func (s *Store) UpdateManufacturer(ctx context.Context, m domain.MasterManufacturer) error {
	_, err := s.db.ExecContext(ctx, `UPDATE dbo.MasterManufacturer SET manufacturer_name = @p2, logo_image = @p3, valid_until = @p4, updated_at = @p5, updated_by = @p6 WHERE vendor_id = @p1`,
		m.VendorId, m.ManufacturerName, m.LogoImage, m.ValidUntil, m.UpdatedAt, m.UpdatedBy)
	return err
}

func (s *Store) DeleteManufacturer(ctx context.Context, vendorId string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM dbo.MasterManufacturer WHERE vendor_id = @p1`, vendorId)
	return err
}

const componentViewSelect = `SELECT c.component_id, c.component_name, m.vendor_id, m.manufacturer_name, c.cost, c.compatibility, c.failure_rate, c.created_at, c.updated_at, c.created_by, c.updated_by, c.serial_number FROM dbo.MasterComponent c INNER JOIN dbo.MasterManufacturer m ON m.vendor_id = c.vendor_id`

func (s *Store) ListComponentViews(ctx context.Context, search string) ([]domain.MasterComponentView, error) {
	rows := []domain.MasterComponentView{}
	if search != "" {
		return rows, s.db.SelectContext(ctx, &rows, componentViewSelect+` WHERE CHARINDEX(@p1, c.component_name) > 0 OR (m.manufacturer_name IS NOT NULL AND CHARINDEX(@p1, m.manufacturer_name) > 0) OR CHARINDEX(@p1, c.component_id) > 0 ORDER BY c.component_id`, search)
	}
	return rows, s.db.SelectContext(ctx, &rows, componentViewSelect+` ORDER BY c.component_id`)
}

func (s *Store) FindComponentView(ctx context.Context, componentId string) (*domain.MasterComponentView, error) {
	var row domain.MasterComponentView
	err := s.db.GetContext(ctx, &row, componentViewSelect+` WHERE c.component_id = @p1`, componentId)
	return optional(&row, err)
}

const componentColumns = `component_id, component_name, vendor_id, failure_rate, cost, compatibility, created_at, updated_at, created_by, updated_by, serial_number`

func (s *Store) ListComponents(ctx context.Context) ([]domain.MasterComponent, error) {
	rows := []domain.MasterComponent{}
	return rows, s.db.SelectContext(ctx, &rows, `SELECT `+componentColumns+` FROM dbo.MasterComponent ORDER BY component_id`)
}

func (s *Store) FindComponent(ctx context.Context, componentId string) (*domain.MasterComponent, error) {
	var row domain.MasterComponent
	err := s.db.GetContext(ctx, &row, `SELECT `+componentColumns+` FROM dbo.MasterComponent WHERE component_id = @p1`, componentId)
	return optional(&row, err)
}

func (s *Store) ComponentByNameAndVendor(ctx context.Context, name, vendorId, excludeId string) (*domain.MasterComponent, error) {
	var row domain.MasterComponent
	err := s.db.GetContext(ctx, &row, `SELECT TOP 1 `+componentColumns+` FROM dbo.MasterComponent WHERE component_name = @p1 AND vendor_id = @p2 AND component_id <> @p3 ORDER BY component_id`, name, vendorId, excludeId)
	return optional(&row, err)
}

func (s *Store) LastComponentId(ctx context.Context) (*string, error) {
	var id string
	err := s.db.GetContext(ctx, &id, `SELECT TOP 1 component_id FROM dbo.MasterComponent ORDER BY component_id DESC`)
	return optional(&id, err)
}

func (s *Store) InsertComponent(ctx context.Context, c domain.MasterComponent) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO dbo.MasterComponent (`+componentColumns+`) VALUES (@p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8, @p9, @p10, @p11)`,
		c.ComponentId, c.ComponentName, c.VendorId, c.FailureRate, c.Cost, c.Compatibility, c.CreatedAt, c.UpdatedAt, c.CreatedBy, c.UpdatedBy, c.SerialNumber)
	return err
}

func (s *Store) UpdateComponent(ctx context.Context, c domain.MasterComponent) error {
	_, err := s.db.ExecContext(ctx, `UPDATE dbo.MasterComponent SET component_name = @p2, vendor_id = @p3, failure_rate = @p4, cost = @p5, compatibility = @p6, serial_number = @p7, updated_at = @p8, updated_by = @p9 WHERE component_id = @p1`,
		c.ComponentId, c.ComponentName, c.VendorId, c.FailureRate, c.Cost, c.Compatibility, c.SerialNumber, c.UpdatedAt, c.UpdatedBy)
	return err
}

func (s *Store) DeleteComponent(ctx context.Context, componentId string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM dbo.MasterComponent WHERE component_id = @p1`, componentId)
	return err
}

type exportRow struct {
	ComponentName    string         `db:"component_name"`
	ManufacturerName sql.NullString `db:"manufacturer_name"`
	FailureRate      *domain.Number `db:"failure_rate"`
	Cost             *string        `db:"cost"`
}

func (s *Store) ExportRows(ctx context.Context) ([]exportRow, error) {
	rows := []exportRow{}
	return rows, s.db.SelectContext(ctx, &rows, `SELECT c.component_name, m.manufacturer_name, c.failure_rate, c.cost FROM dbo.MasterComponent c INNER JOIN dbo.MasterManufacturer m ON m.vendor_id = c.vendor_id ORDER BY c.component_name, m.manufacturer_name`)
}

func (s *Store) RecentComponentProperties(ctx context.Context, count int) ([]domain.SystemComponentProperties, error) {
	rows := []domain.SystemComponentProperties{}
	return rows, s.db.SelectContext(ctx, &rows, `SELECT TOP (@p1) `+domain.SystemComponentColumns+` FROM dbo.SystemComponentProperties ORDER BY updated_at DESC`, count)
}

const projectColumns = `project_id, project_name, hierarchy_depth, created_at, updated_at, created_by, updated_by`

func (s *Store) ListProjects(ctx context.Context, search string) ([]domain.MasterProject, error) {
	rows := []domain.MasterProject{}
	if search != "" {
		return rows, s.db.SelectContext(ctx, &rows, `SELECT `+projectColumns+` FROM dbo.MasterProject WHERE CHARINDEX(@p1, project_name) > 0 OR CHARINDEX(@p1, project_id) > 0 ORDER BY project_id`, search)
	}
	return rows, s.db.SelectContext(ctx, &rows, `SELECT `+projectColumns+` FROM dbo.MasterProject ORDER BY project_id`)
}

func (s *Store) FindProject(ctx context.Context, projectId string) (*domain.MasterProject, error) {
	var row domain.MasterProject
	err := s.db.GetContext(ctx, &row, `SELECT `+projectColumns+` FROM dbo.MasterProject WHERE project_id = @p1`, projectId)
	return optional(&row, err)
}

func (s *Store) ProjectByName(ctx context.Context, name string) (*domain.MasterProject, error) {
	var row domain.MasterProject
	err := s.db.GetContext(ctx, &row, `SELECT TOP 1 `+projectColumns+` FROM dbo.MasterProject WHERE project_name = @p1 ORDER BY project_id`, name)
	return optional(&row, err)
}

func (s *Store) LastProjectId(ctx context.Context) (*string, error) {
	var id string
	err := s.db.GetContext(ctx, &id, `SELECT TOP 1 project_id FROM dbo.MasterProject ORDER BY project_id DESC`)
	return optional(&id, err)
}

func (s *Store) InsertProject(ctx context.Context, p domain.MasterProject) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO dbo.MasterProject (`+projectColumns+`) VALUES (@p1, @p2, @p3, @p4, @p5, @p6, @p7)`,
		p.ProjectId, p.ProjectName, p.HierarchyDepth, p.CreatedAt, p.UpdatedAt, p.CreatedBy, p.UpdatedBy)
	return err
}

func (s *Store) UpdateProject(ctx context.Context, p domain.MasterProject) error {
	_, err := s.db.ExecContext(ctx, `UPDATE dbo.MasterProject SET project_name = @p2, updated_at = @p3, updated_by = @p4 WHERE project_id = @p1`,
		p.ProjectId, p.ProjectName, p.UpdatedAt, p.UpdatedBy)
	return err
}

const projectRbdSelect = `SELECT r.rbd_system_id, p.project_id, p.project_name, r.drawing_name, r.system_name, r.reliability_total, r.created_at, r.updated_at FROM dbo.MasterProject p INNER JOIN dbo.RbdSystemDrawing r ON r.project_id = p.project_id`

func (s *Store) ListProjectSystems(ctx context.Context, search string) ([]domain.MasterProjectRbd, error) {
	rows := []domain.MasterProjectRbd{}
	if search != "" {
		return rows, s.db.SelectContext(ctx, &rows, projectRbdSelect+` WHERE CHARINDEX(@p1, p.project_name) > 0 OR CHARINDEX(@p1, p.project_id) > 0 OR CHARINDEX(@p1, r.system_name) > 0 OR CHARINDEX(@p1, r.drawing_name) > 0 OR CHARINDEX(@p1, r.rbd_system_id) > 0 ORDER BY p.project_id, r.rbd_system_id`, search)
	}
	return rows, s.db.SelectContext(ctx, &rows, projectRbdSelect+` ORDER BY p.project_id, r.rbd_system_id`)
}

func (s *Store) RecentProjectSystems(ctx context.Context, count int) ([]domain.MasterProjectRbd, error) {
	rows := []domain.MasterProjectRbd{}
	return rows, s.db.SelectContext(ctx, &rows, `SELECT TOP (@p1) r.rbd_system_id, p.project_id, p.project_name, r.drawing_name, r.system_name, r.reliability_total, r.created_at, r.updated_at FROM dbo.RbdSystemDrawing r INNER JOIN dbo.MasterProject p ON p.project_id = r.project_id ORDER BY r.updated_at DESC`, count)
}

func (s *Store) HighReliabilitySystems(ctx context.Context, count int) ([]domain.MasterProjectRbd, error) {
	rows := []domain.MasterProjectRbd{}
	return rows, s.db.SelectContext(ctx, &rows, `SELECT TOP (@p1) r.rbd_system_id, p.project_id, p.project_name, r.drawing_name, r.system_name, r.reliability_total, r.created_at, r.updated_at FROM dbo.RbdSystemDrawing r INNER JOIN dbo.MasterProject p ON p.project_id = r.project_id WHERE r.reliability_total IS NOT NULL ORDER BY r.reliability_total DESC`, count)
}

func (s *Store) SystemsOfProject(ctx context.Context, projectId string) ([]domain.RbdSystemDrawing, error) {
	rows := []domain.RbdSystemDrawing{}
	return rows, s.db.SelectContext(ctx, &rows, `SELECT `+domain.RbdSystemColumns+` FROM dbo.RbdSystemDrawing WHERE project_id = @p1 ORDER BY rbd_system_id`, projectId)
}

func (s *Store) HierarchiesOfSystem(ctx context.Context, rbdSystemId string) ([]domain.Hierarchy, error) {
	rows := []domain.Hierarchy{}
	return rows, s.db.SelectContext(ctx, &rows, `SELECT `+domain.HierarchyColumns+` FROM dbo.Hierarchy WHERE rbd_system_id = @p1 ORDER BY hierarchy_id`, rbdSystemId)
}

func (s *Store) ComponentsOfSystem(ctx context.Context, rbdSystemId string) ([]domain.SystemComponentProperties, error) {
	rows := []domain.SystemComponentProperties{}
	return rows, s.db.SelectContext(ctx, &rows, `SELECT `+domain.SystemComponentColumns+` FROM dbo.SystemComponentProperties WHERE rbd_system_id = @p1 ORDER BY system_component_id`, rbdSystemId)
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
