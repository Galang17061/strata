package rbd

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Galang17061/strata-api/internal/domain"
)

func (s *Store) FindThreshold(ctx context.Context, rbdSystemId string) (*domain.Number, error) {
	var threshold domain.Number
	err := s.q.GetContext(ctx, &threshold, `SELECT threshold FROM dbo.SystemThreshold WHERE rbd_system_id = @p1`, rbdSystemId)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &threshold, nil
}

func (s *Store) SetThreshold(ctx context.Context, rbdSystemId string, threshold *domain.Number, updatedBy string) error {
	if threshold == nil {
		_, err := s.q.ExecContext(ctx, `DELETE FROM dbo.SystemThreshold WHERE rbd_system_id = @p1`, rbdSystemId)
		return err
	}
	_, err := s.q.ExecContext(ctx, `MERGE dbo.SystemThreshold AS target USING (SELECT @p1 AS rbd_system_id) AS source ON target.rbd_system_id = source.rbd_system_id WHEN MATCHED THEN UPDATE SET threshold = @p2, updated_by = @p3, updated_at = GETDATE() WHEN NOT MATCHED THEN INSERT (rbd_system_id, threshold, updated_by) VALUES (@p1, @p2, @p3);`, rbdSystemId, threshold, updatedBy)
	return err
}

type notificationRow struct {
	NotificationId string          `db:"notification_id" json:"notificationId"`
	RbdSystemId    string          `db:"rbd_system_id" json:"rbdSystemId"`
	Title          string          `db:"title" json:"title"`
	Body           string          `db:"body" json:"body"`
	CreatedAt      domain.DateTime `db:"created_at" json:"createdAt"`
}

func (s *Store) InsertNotification(ctx context.Context, rbdSystemId, title, body string) error {
	_, err := s.q.ExecContext(ctx, `INSERT INTO dbo.Notification (notification_id, rbd_system_id, title, body) VALUES (@p1, @p2, @p3, @p4)`,
		domain.NewGuid().String(), rbdSystemId, title, body)
	return err
}

func (s *Store) RecentNotifications(ctx context.Context, limit int) ([]notificationRow, error) {
	rows := []notificationRow{}
	err := s.q.SelectContext(ctx, &rows, `SELECT TOP (@p1) notification_id, rbd_system_id, title, body, created_at FROM dbo.Notification ORDER BY created_at DESC, notification_id DESC`, limit)
	return rows, err
}
