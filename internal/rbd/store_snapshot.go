package rbd

import (
	"context"

	"github.com/Galang17061/strata-api/internal/domain"
)

func (s *Store) InsertSnapshot(ctx context.Context, row domain.SystemSnapshot) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO dbo.SystemSnapshot (system_snapshot_id, rbd_system_id, label, kind, payload, created_by) VALUES ($1, $2, $3, $4, $5, $6)`,
		row.SystemSnapshotId, row.RbdSystemId, row.Label, row.Kind, row.Payload, row.CreatedBy)
	return err
}

func (s *Store) SnapshotsOfSystem(ctx context.Context, rbdSystemId string) ([]domain.SystemSnapshot, error) {
	rows := []domain.SystemSnapshot{}
	err := s.db.SelectContext(ctx, &rows, `SELECT system_snapshot_id, rbd_system_id, label, kind, '' AS payload, created_by, created_at FROM dbo.SystemSnapshot WHERE rbd_system_id = $1 ORDER BY created_at DESC, system_snapshot_id DESC`, rbdSystemId)
	return rows, err
}

func (s *Store) FindSnapshot(ctx context.Context, id string) (*domain.SystemSnapshot, error) {
	var row domain.SystemSnapshot
	err := s.db.GetContext(ctx, &row, `SELECT system_snapshot_id, rbd_system_id, label, kind, payload, created_by, created_at FROM dbo.SystemSnapshot WHERE system_snapshot_id = $1`, id)
	return optional(&row, err)
}

func (s *Store) DeleteSnapshot(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM dbo.SystemSnapshot WHERE system_snapshot_id = $1`, id)
	return err
}

func (s *Store) PruneAutoSnapshots(ctx context.Context, rbdSystemId string, keep int) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM dbo.SystemSnapshot WHERE rbd_system_id = $1 AND kind = 'auto' AND system_snapshot_id NOT IN (SELECT system_snapshot_id FROM dbo.SystemSnapshot WHERE rbd_system_id = $1 AND kind = 'auto' ORDER BY created_at DESC, system_snapshot_id DESC LIMIT $2)`, rbdSystemId, keep)
	return err
}
