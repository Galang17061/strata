package rbd

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Galang17061/strata-api/internal/auth"
	"github.com/Galang17061/strata-api/internal/web"
)

type PoissonHandler struct {
	parameters *ParameterService
}

func NewPoissonHandler(parameters *ParameterService) *PoissonHandler {
	return &PoissonHandler{parameters: parameters}
}

func (h *PoissonHandler) Mount(router chi.Router) {
	router.Group(func(protected chi.Router) {
		protected.Use(auth.Require)
		protected.Post("/api/SystemComponentProperties/{systemComponentId}/poisson-parameter", h.seed)
	})
}

// @Summary Create the initial Poisson rate entries for a component from its failure history
// @Tags SystemComponentProperties
// @Produce json
// @Param systemComponentId path string true "System component id"
// @Success 201 {object} web.Envelope
// @Failure 400 {object} web.Envelope
// @Failure 500 {object} web.Envelope
// @Security BearerAuth
// @Router /api/SystemComponentProperties/{systemComponentId}/poisson-parameter [post]
func (h *PoissonHandler) seed(w http.ResponseWriter, r *http.Request) {
	systemComponentId := chi.URLParam(r, "systemComponentId")
	if strings.TrimSpace(systemComponentId) == "" {
		web.Respond(w, http.StatusBadRequest, web.BadRequest("SystemComponentId is required"))
		return
	}
	envelope, err := h.parameters.SeedPoisson(r.Context(), systemComponentId, auth.CurrentUserName(r.Context()))
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
