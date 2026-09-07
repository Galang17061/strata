package audit

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/Galang17061/strata-api/internal/auth"
	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/web"
)

type Store struct {
	db *sqlx.DB
}

func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

type Entry struct {
	AuditTrailId string          `db:"audit_trail_id" json:"auditTrailId"`
	UserName     string          `db:"user_name" json:"userName"`
	Method       string          `db:"method" json:"method"`
	Path         string          `db:"path" json:"path"`
	StatusCode   int             `db:"status_code" json:"statusCode"`
	CreatedAt    domain.DateTime `db:"created_at" json:"createdAt"`
}

func (s *Store) Insert(ctx context.Context, entry Entry) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO dbo.AuditTrail (audit_trail_id, user_name, method, path, status_code) VALUES ($1, $2, $3, $4, $5)`,
		entry.AuditTrailId, entry.UserName, entry.Method, entry.Path, entry.StatusCode)
	return err
}

func (s *Store) Page(ctx context.Context, page, pageSize int) ([]Entry, int, error) {
	total := 0
	if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM dbo.AuditTrail`); err != nil {
		return nil, 0, err
	}
	rows := []Entry{}
	offset := (page - 1) * pageSize
	err := s.db.SelectContext(ctx, &rows, `SELECT audit_trail_id, user_name, method, path, status_code, created_at FROM dbo.AuditTrail ORDER BY created_at DESC, audit_trail_id DESC OFFSET $1 ROWS FETCH NEXT $2 ROWS ONLY`, offset, pageSize)
	return rows, total, err
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func Middleware(store *Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			method := r.Method
			if method != http.MethodPost && method != http.MethodPut && method != http.MethodDelete {
				next.ServeHTTP(w, r)
				return
			}
			if !strings.HasPrefix(strings.ToLower(r.URL.Path), "/api/") {
				next.ServeHTTP(w, r)
				return
			}
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)
			user := auth.CurrentUserName(r.Context())
			path := r.URL.Path
			if len(path) > 400 {
				path = path[:400]
			}
			store.Insert(r.Context(), Entry{
				AuditTrailId: domain.NewGuid().String(),
				UserName:     user,
				Method:       method,
				Path:         path,
				StatusCode:   recorder.status,
			})
		})
	}
}

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Mount(router chi.Router) {
	router.With(auth.Require).Get("/api/Audit", h.list)
}

// @Summary Page through the record of who changed what and when
// @Tags Audit
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Rows per page"
// @Success 200 {object} web.Envelope
// @Security BearerAuth
// @Router /api/Audit [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page, err := web.QueryInt(r, "page", 1)
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := web.QueryInt(r, "pageSize", 20)
	if err != nil || pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	rows, total, err := h.store.Page(r.Context(), page, pageSize)
	if err != nil {
		web.RespondMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	totalPages := (total + pageSize - 1) / pageSize
	web.Respond(w, http.StatusOK, web.SuccessWithMeta(rows, "Success", web.NewMeta(total, totalPages, page, pageSize)))
}
