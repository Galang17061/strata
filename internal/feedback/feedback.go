package feedback

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/Galang17061/strata-api/internal/auth"
	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/mail"
	"github.com/Galang17061/strata-api/internal/web"
)

type Store struct {
	db *sqlx.DB
}

func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

type Entry struct {
	FeedbackId string          `db:"feedback_id" json:"feedbackId"`
	UserName   string          `db:"user_name" json:"userName"`
	Category   string          `db:"category" json:"category"`
	Message    string          `db:"message" json:"message"`
	Page       *string         `db:"page" json:"page"`
	CreatedAt  domain.DateTime `db:"created_at" json:"createdAt"`
}

func (s *Store) Insert(ctx context.Context, entry Entry) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO dbo.Feedback (feedback_id, user_name, category, message, page) VALUES ($1, $2, $3, $4, $5)`,
		entry.FeedbackId, entry.UserName, entry.Category, entry.Message, entry.Page)
	return err
}

func (s *Store) Page(ctx context.Context, page, pageSize int) ([]Entry, int, error) {
	total := 0
	if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM dbo.Feedback`); err != nil {
		return nil, 0, err
	}
	rows := []Entry{}
	offset := (page - 1) * pageSize
	err := s.db.SelectContext(ctx, &rows, `SELECT feedback_id, user_name, category, message, page, created_at FROM dbo.Feedback ORDER BY created_at DESC, feedback_id DESC OFFSET $1 ROWS FETCH NEXT $2 ROWS ONLY`, offset, pageSize)
	return rows, total, err
}

type CreateRequest struct {
	Category string `json:"category"`
	Message  string `json:"message"`
	Page     string `json:"page"`
}

func ValidCategory(category string) bool {
	switch category {
	case "bug", "idea", "question":
		return true
	}
	return false
}

type Handler struct {
	store  *Store
	mailer *mail.Sender
	inbox  string
}

func NewHandler(store *Store, mailer *mail.Sender, inbox string) *Handler {
	return &Handler{store: store, mailer: mailer, inbox: inbox}
}

func (h *Handler) Mount(router chi.Router) {
	router.Group(func(protected chi.Router) {
		protected.Use(auth.Require)
		protected.Post("/api/Feedback", h.create)
		protected.Get("/api/Feedback", h.list)
	})
}

// @Summary Send a note to the people who build Strata
// @Tags Feedback
// @Accept json
// @Produce json
// @Param request body feedback.CreateRequest true "Category, message and the page it concerns"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} web.ValidationProblem
// @Security BearerAuth
// @Router /api/Feedback [post]
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var request CreateRequest
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	message := strings.TrimSpace(request.Message)
	problems := map[string][]string{}
	if !ValidCategory(request.Category) {
		problems["Category"] = []string{"Category must be bug, idea or question."}
	}
	if message == "" {
		problems["Message"] = []string{web.RequiredMessage("Message")}
	}
	if len(message) > 2000 {
		problems["Message"] = []string{"Message must stay under 2000 characters."}
	}
	if len(problems) > 0 {
		web.RespondValidation(w, problems)
		return
	}
	entry := Entry{
		FeedbackId: domain.NewGuid().String(),
		UserName:   auth.CurrentUserName(r.Context()),
		Category:   request.Category,
		Message:    message,
	}
	if page := strings.TrimSpace(request.Page); page != "" {
		if len(page) > 400 {
			page = page[:400]
		}
		entry.Page = domain.StringPtr(page)
	}
	if err := h.store.Insert(r.Context(), entry); err != nil {
		web.RespondMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if h.inbox != "" && h.mailer.Enabled() {
		body := "From: " + entry.UserName + "\nCategory: " + entry.Category + "\nPage: " + domain.Deref(entry.Page) + "\n\n" + message
		h.mailer.Send(h.inbox, "Strata feedback ("+entry.Category+") from "+entry.UserName, body)
	}
	web.Respond(w, http.StatusOK, web.Success(nil, "Thank you. Your note has reached the people who build Strata."))
}

// @Summary Page through the notes people have sent in
// @Tags Feedback
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Rows per page"
// @Success 200 {object} web.Envelope
// @Security BearerAuth
// @Router /api/Feedback [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page, err := web.QueryInt(r, "page", 1)
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := web.QueryInt(r, "pageSize", 12)
	if err != nil || pageSize < 1 || pageSize > 200 {
		pageSize = 12
	}
	rows, total, err := h.store.Page(r.Context(), page, pageSize)
	if err != nil {
		web.RespondMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	totalPages := (total + pageSize - 1) / pageSize
	web.Respond(w, http.StatusOK, web.SuccessWithMeta(rows, "Success", web.NewMeta(total, totalPages, page, pageSize)))
}
