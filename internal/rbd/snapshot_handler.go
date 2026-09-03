package rbd

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Galang17061/strata-api/internal/auth"
	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/web"
)

type SnapshotHandler struct {
	service *SnapshotService
}

func NewSnapshotHandler(service *SnapshotService) *SnapshotHandler {
	return &SnapshotHandler{service: service}
}

func (h *SnapshotHandler) Mount(router chi.Router) {
	router.Group(func(protected chi.Router) {
		protected.Use(auth.Require)
		protected.Get("/api/Snapshot/system/{RbdSystemId}", h.list)
		protected.Post("/api/Snapshot/system/{RbdSystemId}", h.create)
		protected.Delete("/api/Snapshot/{SnapshotId}", h.remove)
	})
}

// @Summary List the saved versions of a system
// @Tags Snapshot
// @Produce json
// @Param RbdSystemId path string true "System id"
// @Success 200 {object} web.Envelope
// @Security BearerAuth
// @Router /api/Snapshot/system/{RbdSystemId} [get]
func (h *SnapshotHandler) list(w http.ResponseWriter, r *http.Request) {
	rows, err := h.service.List(r.Context(), chi.URLParam(r, "RbdSystemId"))
	if err != nil {
		web.RespondMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	web.Respond(w, http.StatusOK, web.Success(rows, "Success"))
}

// @Summary Save a named version of a system as it stands right now
// @Tags Snapshot
// @Accept json
// @Produce json
// @Param RbdSystemId path string true "System id"
// @Param request body domain.SnapshotCreateRequest true "Version label"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/Snapshot/system/{RbdSystemId} [post]
func (h *SnapshotHandler) create(w http.ResponseWriter, r *http.Request) {
	rbdSystemId := strings.TrimSpace(chi.URLParam(r, "RbdSystemId"))
	if rbdSystemId == "" {
		web.RespondMessage(w, http.StatusBadRequest, "RbdSystemId is required")
		return
	}
	var request domain.SnapshotCreateRequest
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	row, err := h.service.Create(r.Context(), rbdSystemId, domain.Deref(request.Label), "manual", auth.CurrentUserName(r.Context()))
	if err != nil {
		web.RespondMessage(w, http.StatusBadRequest, err.Error())
		return
	}
	web.Respond(w, http.StatusOK, web.Success(row, "The version has been saved."))
}

// @Summary Throw away one saved version
// @Tags Snapshot
// @Produce json
// @Param SnapshotId path string true "Snapshot id"
// @Success 200 {object} web.Envelope
// @Security BearerAuth
// @Router /api/Snapshot/{SnapshotId} [delete]
func (h *SnapshotHandler) remove(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), chi.URLParam(r, "SnapshotId")); err != nil {
		web.RespondMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	web.Respond(w, http.StatusOK, web.Success(nil, "The version has been removed."))
}
