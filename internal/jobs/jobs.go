package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"

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

const (
	StatusCancelled = "cancelled"
	QueuePerUser    = 3
	QueueCeiling    = 60
)

const columns = `job_id, kind, rbd_system_id, status, request, result, error_message, created_by, created_by_id, created_at, started_at, finished_at`

type Job struct {
	JobId        string           `db:"job_id"`
	Kind         string           `db:"kind"`
	RbdSystemId  *string          `db:"rbd_system_id"`
	Status       string           `db:"status"`
	Request      string           `db:"request"`
	Result       *string          `db:"result"`
	ErrorMessage *string          `db:"error_message"`
	CreatedBy    *string          `db:"created_by"`
	CreatedById  *string          `db:"created_by_id"`
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

func (s *Store) Enqueue(ctx context.Context, kind, request string, rbdSystemId *string, createdBy, createdById string) (Job, error) {
	job := Job{
		JobId:       domain.NewGuid().String(),
		Kind:        kind,
		RbdSystemId: rbdSystemId,
		Status:      StatusQueued,
		Request:     request,
		CreatedBy:   domain.StringPtr(createdBy),
		CreatedById: domain.StringPtr(createdById),
	}
	err := s.db.GetContext(ctx, &job, `INSERT INTO dbo.Job (job_id, kind, rbd_system_id, status, request, created_by, created_by_id) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING `+columns,
		job.JobId, job.Kind, job.RbdSystemId, job.Status, job.Request, job.CreatedBy, job.CreatedById)
	if err != nil {
		return Job{}, err
	}
	s.Announce(ctx, job)
	return job, nil
}

func (s *Store) Claim(ctx context.Context) (*Job, error) {
	job := Job{}
	err := s.db.GetContext(ctx, &job, `UPDATE dbo.Job SET status = $1, started_at = now() WHERE job_id = (SELECT job_id FROM dbo.Job WHERE status = $2 ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED) AND status = $2 RETURNING `+columns, StatusRunning, StatusQueued)
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

func (s *Store) CountUnfinished(ctx context.Context, ownerId string) (int, error) {
	count := 0
	if ownerId == "" {
		err := s.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM dbo.Job WHERE status IN ($1, $2)`, StatusQueued, StatusRunning)
		return count, err
	}
	err := s.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM dbo.Job WHERE status IN ($1, $2) AND created_by_id = $3`, StatusQueued, StatusRunning, ownerId)
	return count, err
}

func (s *Store) Cancel(ctx context.Context, jobId string) (bool, error) {
	job := Job{}
	err := s.db.GetContext(ctx, &job, `UPDATE dbo.Job SET status = $1, finished_at = now() WHERE job_id = $2 AND status = $3 RETURNING `+columns, StatusCancelled, jobId, StatusQueued)
	if err != nil {
		if isNoRows(err) {
			return false, nil
		}
		return false, err
	}
	s.Announce(ctx, job)
	return true, nil
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

func (s *Store) Page(ctx context.Context, rbdSystemId, ownerId string, page, pageSize int) ([]Job, int, error) {
	filters := []string{}
	args := []any{}
	if rbdSystemId != "" {
		args = append(args, rbdSystemId)
		filters = append(filters, "rbd_system_id = $"+strconv.Itoa(len(args)))
	}
	if ownerId != "" {
		args = append(args, ownerId)
		filters = append(filters, "created_by_id = $"+strconv.Itoa(len(args)))
	}
	clause := ""
	if len(filters) > 0 {
		clause = " WHERE " + strings.Join(filters, " AND ")
	}
	total := 0
	if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM dbo.Job`+clause, args...); err != nil {
		return nil, 0, err
	}
	rows := []Job{}
	args = append(args, (page-1)*pageSize, pageSize)
	query := `SELECT ` + columns + ` FROM dbo.Job` + clause + ` ORDER BY created_at DESC, job_id DESC OFFSET $` + strconv.Itoa(len(args)-1) + ` ROWS FETCH NEXT $` + strconv.Itoa(len(args)) + ` ROWS ONLY`
	err := s.db.SelectContext(ctx, &rows, query, args...)
	return rows, total, err
}

func (s *Store) Announce(ctx context.Context, job Job) {
	payload, err := json.Marshal(Announcement{JobId: job.JobId, Status: job.Status, Kind: job.Kind})
	if err != nil {
		log.Printf("could not describe job %s for the announcement: %v", job.JobId, err)
		return
	}
	if _, err := s.db.ExecContext(ctx, `SELECT pg_notify($1, $2)`, Channel, string(payload)); err != nil {
		log.Printf("job %s moved to %s but nobody could be told: %v", job.JobId, job.Status, err)
	}
}

func isNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
