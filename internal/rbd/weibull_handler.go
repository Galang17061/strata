package rbd

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Galang17061/strata-api/internal/auth"
	"github.com/Galang17061/strata-api/internal/web"
)

type WeibullHandler struct {
	parameters *ParameterService
}

func NewWeibullHandler(parameters *ParameterService) *WeibullHandler {
	return &WeibullHandler{parameters: parameters}
}

func (h *WeibullHandler) Mount(router chi.Router) {
	router.Group(func(protected chi.Router) {
		protected.Use(auth.Require)
		protected.Get("/api/SystemComponentProperties/{systemComponentId}/weibull-parameters", h.list)
		protected.Put("/api/SystemComponentProperties/{systemComponentId}/weibull", h.update)
		protected.Post("/api/SystemComponentProperties/{systemComponentId}/weibull-parameter", h.seed)
	})
}

// @Summary List the Weibull shape and scale values of a component
// @Tags SystemComponentProperties
// @Produce json
// @Param systemComponentId path string true "System component id"
// @Success 200 {object} web.Envelope
// @Failure 500 {string} string
// @Security BearerAuth
// @Router /api/SystemComponentProperties/{systemComponentId}/weibull-parameters [get]
func (h *WeibullHandler) list(w http.ResponseWriter, r *http.Request) {
	rows, err := h.parameters.WeibullViews(r.Context(), chi.URLParam(r, "systemComponentId"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(rows, "Weibull parameters retrieved successfully"))
}

// @Summary Recalculate the Weibull shape and scale of a component from its failure history
// @Tags SystemComponentProperties
// @Produce json
// @Param systemComponentId path string true "System component id"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} web.Envelope
// @Failure 500 {object} web.Envelope
// @Security BearerAuth
// @Router /api/SystemComponentProperties/{systemComponentId}/weibull [put]
func (h *WeibullHandler) update(w http.ResponseWriter, r *http.Request) {
	systemComponentId := chi.URLParam(r, "systemComponentId")
	if strings.TrimSpace(systemComponentId) == "" {
		web.Respond(w, http.StatusBadRequest, web.Failed(http.StatusBadRequest, "SystemComponentId is required", nil))
		return
	}
	envelope, err := h.parameters.UpdateWeibull(r.Context(), systemComponentId, auth.CurrentUserName(r.Context()))
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.Failed(http.StatusInternalServerError, "Internal server error: "+err.Error(), nil))
		return
	}
	if envelope.StatusCode == http.StatusBadRequest || envelope.StatusCode == http.StatusNotFound {
		web.Respond(w, http.StatusBadRequest, envelope)
		return
	}
	web.Respond(w, http.StatusOK, envelope)
}

// @Summary Create the initial Weibull shape and scale entry for a component
// @Tags SystemComponentProperties
// @Produce json
// @Param systemComponentId path string true "System component id"
// @Success 201 {object} web.Envelope
// @Failure 400 {object} web.Envelope
// @Failure 500 {object} web.Envelope
// @Security BearerAuth
// @Router /api/SystemComponentProperties/{systemComponentId}/weibull-parameter [post]
func (h *WeibullHandler) seed(w http.ResponseWriter, r *http.Request) {
	systemComponentId := chi.URLParam(r, "systemComponentId")
	if strings.TrimSpace(systemComponentId) == "" {
		web.Respond(w, http.StatusBadRequest, web.BadRequest("SystemComponentId is required"))
		return
	}
	envelope, err := h.parameters.SeedWeibull(r.Context(), systemComponentId, auth.CurrentUserName(r.Context()))
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Internal server error: "+err.Error()))
		return
	}
	if envelope.StatusCode == http.StatusBadRequest {
		web.Respond(w, http.StatusBadRequest, envelope)
		return
	}
	w.Header().Set("Location", web.AbsoluteURL(r, "/api/SystemComponentProperties/"+systemComponentId))
	web.Respond(w, http.StatusCreated, envelope)
}
