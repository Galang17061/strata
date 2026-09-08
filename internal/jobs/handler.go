package jobs

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Galang17061/strata-api/internal/auth"
	"github.com/Galang17061/strata-api/internal/web"
)

type Handler struct {
	store    *Store
	listener *Listener
}

func NewHandler(store *Store, listener *Listener) *Handler {
	return &Handler{store: store, listener: listener}
}

func (h *Handler) Mount(router chi.Router) {
	router.Group(func(protected chi.Router) {
		protected.Use(auth.Require)
		protected.Get("/api/Job", h.list)
		protected.Get("/api/Job/stream", h.stream)
		protected.Get("/api/Job/{jobId}", h.detail)
		protected.Post("/api/Job/{jobId}/cancel", h.cancel)
	})
}

func Keeper(ctx context.Context) bool {
	identity, ok := auth.IdentityFrom(ctx)
	return ok && strings.EqualFold(identity.Role, "admin")
}

func OwnerId(ctx context.Context) string {
	identity, _ := auth.IdentityFrom(ctx)
	return identity.Id
}

func ownScope(ctx context.Context) string {
	if Keeper(ctx) {
		return ""
	}
	return OwnerId(ctx)
}

func mine(ctx context.Context, job *Job) bool {
	if Keeper(ctx) {
		return true
	}
	owner := OwnerId(ctx)
	return owner != "" && job.CreatedById != nil && *job.CreatedById == owner
}

// @Summary Page through the jobs the workers have taken on
// @Tags Job
// @Produce json
// @Param rbdSystemId query string false "Only jobs for this system"
// @Param page query int false "Page number"
// @Param pageSize query int false "Rows per page"
// @Success 200 {object} web.Envelope
// @Security BearerAuth
// @Router /api/Job [get]
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page, err := web.QueryInt(r, "page", 1)
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := web.QueryInt(r, "pageSize", 12)
	if err != nil || pageSize < 1 || pageSize > 200 {
		pageSize = 12
	}
	rows, total, err := h.store.Page(r.Context(), r.URL.Query().Get("rbdSystemId"), ownScope(r.Context()), page, pageSize)
	if err != nil {
		web.RespondMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	views := make([]View, 0, len(rows))
	for _, row := range rows {
		views = append(views, row.View())
	}
	totalPages := (total + pageSize - 1) / pageSize
	web.Respond(w, http.StatusOK, web.SuccessWithMeta(views, "Success", web.NewMeta(total, totalPages, page, pageSize)))
}

// @Summary Look at one job and its result
// @Tags Job
// @Produce json
// @Param jobId path string true "Job id"
// @Success 200 {object} web.Envelope
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /api/Job/{jobId} [get]
func (h *Handler) detail(w http.ResponseWriter, r *http.Request) {
	job, err := h.store.Find(r.Context(), chi.URLParam(r, "jobId"))
	if err != nil {
		web.RespondMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if job == nil || !mine(r.Context(), job) {
		web.RespondMessage(w, http.StatusNotFound, "Job not found")
		return
	}
	web.Respond(w, http.StatusOK, web.Success(job.View(), "Job"))
}

// @Summary Call off a job that is still waiting its turn
// @Tags Job
// @Produce json
// @Param jobId path string true "Job id"
// @Success 200 {object} web.Envelope
// @Failure 404 {object} map[string]string
// @Security BearerAuth
// @Router /api/Job/{jobId}/cancel [post]
func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	job, err := h.store.Find(r.Context(), chi.URLParam(r, "jobId"))
	if err != nil {
		web.RespondMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if job == nil || !mine(r.Context(), job) {
		web.RespondMessage(w, http.StatusNotFound, "Job not found")
		return
	}
	called, err := h.store.Cancel(r.Context(), job.JobId)
	if err != nil {
		web.RespondMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !called {
		web.Respond(w, http.StatusConflict, web.Failed(http.StatusConflict, "This job has already left the queue, so it cannot be called off.", nil))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(nil, "The job was called off."))
}

// @Summary Hold a line open and hear about every job as it moves
// @Tags Job
// @Produce plain
// @Success 200 {string} string "event stream"
// @Security BearerAuth
// @Router /api/Job/stream [get]
func (h *Handler) stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok || h.listener == nil {
		web.RespondMessage(w, http.StatusInternalServerError, "This server cannot hold a stream open.")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, ": listening\n\n")
	flusher.Flush()
	updates, stop := h.listener.Watch()
	defer stop()
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case payload, open := <-updates:
			if !open {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprint(w, ": still here\n\n")
			flusher.Flush()
		}
	}
}
