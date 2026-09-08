package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/Galang17061/strata-api/internal/domain"
)

const (
	StatusQueued  = "queued"
	StatusRunning = "running"
	StatusDone    = "done"
	StatusFailed  = "failed"
	Channel       = "strata_jobs"
)

const columns = `job_id, kind, rbd_system_id, status, request, result, error_message, created_by, created_at, started_at, finished_at`

type Job struct {
	JobId        string           `db:"job_id"`
	Kind         string           `db:"kind"`
	RbdSystemId  *string          `db:"rbd_system_id"`
	Status       string           `db:"status"`
	Request      string           `db:"request"`
	Result       *string          `db:"result"`
	ErrorMessage *string          `db:"error_message"`
	CreatedBy    *string          `db:"created_by"`
	CreatedAt    domain.DateTime  `db:"created_at"`
	StartedAt    *domain.DateTime `db:"started_at"`
	FinishedAt   *domain.DateTime `db:"finished_at"`
}

type View struct {
	JobId        string           `json:"jobId"`
	Kind         string           `json:"kind"`
	RbdSystemId  *string          `json:"rbdSystemId"`
	Status       string           `json:"status"`
	Request      json.RawMessage  `json:"request"`
	Result       json.RawMessage  `json:"result"`
	ErrorMessage *string          `json:"errorMessage"`
	CreatedBy    *string          `json:"createdBy"`
	CreatedAt    domain.DateTime  `json:"createdAt"`
	StartedAt    *domain.DateTime `json:"startedAt"`
	FinishedAt   *domain.DateTime `json:"finishedAt"`
}

func (j Job) View() View {
	return View{
		JobId:        j.JobId,
		Kind:         j.Kind,
		RbdSystemId:  j.RbdSystemId,
		Status:       j.Status,
		Request:      rawJSON(&j.Request),
		Result:       rawJSON(j.Result),
		ErrorMessage: j.ErrorMessage,
		CreatedBy:    j.CreatedBy,
		CreatedAt:    j.CreatedAt,
		StartedAt:    j.StartedAt,
		FinishedAt:   j.FinishedAt,
	}
}

func rawJSON(text *string) json.RawMessage {
	if text == nil || *text == "" {
		return nil
	}
	if !json.Valid([]byte(*text)) {
		return nil
	}
	return json.RawMessage(*text)
}

type Announcement struct {
	JobId  string `json:"jobId"`
	Status string `json:"status"`
	Kind   string `json:"kind"`
}

type Store struct {
	db *sqlx.DB
}

func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Enqueue(ctx context.Context, kind, request string, rbdSystemId *string, createdBy string) (Job, error) {
	job := Job{
		JobId:       domain.NewGuid().String(),
		Kind:        kind,
		RbdSystemId: rbdSystemId,
		Status:      StatusQueued,
		Request:     request,
		CreatedBy:   domain.StringPtr(createdBy),
	}
	err := s.db.GetContext(ctx, &job, `INSERT INTO dbo.Job (job_id, kind, rbd_system_id, status, request, created_by) VALUES ($1, $2, $3, $4, $5, $6) RETURNING `+columns,
		job.JobId, job.Kind, job.RbdSystemId, job.Status, job.Request, job.CreatedBy)
	if err != nil {
		return Job{}, err
	}
	s.Announce(ctx, job)
	return job, nil
}

func (s *Store) Claim(ctx context.Context) (*Job, error) {
	job := Job{}
	err := s.db.GetContext(ctx, &job, `UPDATE dbo.Job SET status = $1, started_at = now() WHERE job_id = (SELECT job_id FROM dbo.Job WHERE status = $2 ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED) RETURNING `+columns, StatusRunning, StatusQueued)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	s.Announce(ctx, job)
	return &job, nil
}

func (s *Store) Finish(ctx context.Context, jobId, result string) error {
	job := Job{}
	err := s.db.GetContext(ctx, &job, `UPDATE dbo.Job SET status = $1, result = $2, finished_at = now() WHERE job_id = $3 RETURNING `+columns, StatusDone, result, jobId)
	if err != nil {
		return err
	}
	s.Announce(ctx, job)
	return nil
}

func (s *Store) Fail(ctx context.Context, jobId, message string) error {
	job := Job{}
	err := s.db.GetContext(ctx, &job, `UPDATE dbo.Job SET status = $1, error_message = $2, finished_at = now() WHERE job_id = $3 RETURNING `+columns, StatusFailed, message, jobId)
	if err != nil {
		return err
	}
	s.Announce(ctx, job)
	return nil
}

func (s *Store) ReleaseStranded(ctx context.Context, olderThanMinutes int) (int64, error) {
	result, err := s.db.ExecContext(ctx, `UPDATE dbo.Job SET status = $1, started_at = NULL WHERE status = $2 AND started_at < now() - make_interval(mins => $3)`, StatusQueued, StatusRunning, olderThanMinutes)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *Store) Find(ctx context.Context, jobId string) (*Job, error) {
	job := Job{}
	err := s.db.GetContext(ctx, &job, `SELECT `+columns+` FROM dbo.Job WHERE job_id = $1`, jobId)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return &job, nil
}

func (s *Store) Page(ctx context.Context, rbdSystemId string, page, pageSize int) ([]Job, int, error) {
	total := 0
	rows := []Job{}
	offset := (page - 1) * pageSize
	if rbdSystemId == "" {
		if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM dbo.Job`); err != nil {
			return nil, 0, err
		}
		err := s.db.SelectContext(ctx, &rows, `SELECT `+columns+` FROM dbo.Job ORDER BY created_at DESC, job_id DESC OFFSET $1 ROWS FETCH NEXT $2 ROWS ONLY`, offset, pageSize)
		return rows, total, err
	}
	if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM dbo.Job WHERE rbd_system_id = $1`, rbdSystemId); err != nil {
		return nil, 0, err
	}
	err := s.db.SelectContext(ctx, &rows, `SELECT `+columns+` FROM dbo.Job WHERE rbd_system_id = $1 ORDER BY created_at DESC, job_id DESC OFFSET $2 ROWS FETCH NEXT $3 ROWS ONLY`, rbdSystemId, offset, pageSize)
	return rows, total, err
}

func (s *Store) Announce(ctx context.Context, job Job) {
	payload, err := json.Marshal(Announcement{JobId: job.JobId, Status: job.Status, Kind: job.Kind})
	if err != nil {
		return
	}
	s.db.ExecContext(ctx, `SELECT pg_notify($1, $2)`, Channel, string(payload))
}

func isNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
