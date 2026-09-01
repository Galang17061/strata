package rbd

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Galang17061/strata-api/internal/auth"
	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/web"
)

type OptimizationHandler struct {
	service *OptimizationService
}

func NewOptimizationHandler(service *OptimizationService) *OptimizationHandler {
	return &OptimizationHandler{service: service}
}

func (h *OptimizationHandler) Mount(router chi.Router) {
	router.Group(func(protected chi.Router) {
		protected.Use(auth.Require)
		protected.Post("/api/Optimization/preview", h.preview)
		protected.Post("/api/Optimization/score", h.score)
		protected.Post("/api/Optimization/apply", h.apply)
	})
}

func respondOptimizationError(w http.ResponseWriter, err error) {
	var argument domain.ArgumentError
	var invalid domain.InvalidOperationError
	var missing domain.KeyNotFoundError
	switch {
	case errors.As(err, &argument):
		web.Respond(w, http.StatusBadRequest, web.BadRequest(err.Error()))
	case errors.As(err, &invalid):
		web.Respond(w, http.StatusBadRequest, web.BadRequest(err.Error()))
	case errors.As(err, &missing):
		web.Respond(w, http.StatusNotFound, web.NotFound(err.Error()))
	default:
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Internal server error: "+err.Error()))
	}
}

// @Summary Search vendor line-ups for a system with a genetic algorithm and preview the best one
// @Tags Optimization
// @Accept json
// @Produce json
// @Param request body domain.OptimizationPreviewRequest true "Optimization request"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} web.Envelope
// @Failure 404 {object} web.Envelope
// @Failure 500 {object} web.Envelope
// @Security BearerAuth
// @Router /api/Optimization/preview [post]
func (h *OptimizationHandler) preview(w http.ResponseWriter, r *http.Request) {
	var request domain.OptimizationPreviewRequest
	if err := web.DecodeBody(r, &request); err != nil {
		web.Respond(w, http.StatusBadRequest, web.BadRequest(err.Error()))
		return
	}
	result, err := h.service.Preview(r.Context(), request)
	if err != nil {
		respondOptimizationError(w, err)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(result, "Optimization preview calculated successfully"))
}

// @Summary Turn a previewed vendor line-up into a new project holding a full copy of the system
// @Tags Optimization
// @Accept json
// @Produce json
// @Param request body domain.OptimizationApplyRequest true "Apply request"
// @Success 201 {object} web.Envelope
// @Failure 400 {object} web.Envelope
// @Failure 404 {object} web.Envelope
// @Failure 500 {object} web.Envelope
// @Security BearerAuth
// @Router /api/Optimization/apply [post]
func (h *OptimizationHandler) apply(w http.ResponseWriter, r *http.Request) {
	var request domain.OptimizationApplyRequest
	if err := web.DecodeBody(r, &request); err != nil {
		web.Respond(w, http.StatusBadRequest, web.BadRequest(err.Error()))
		return
	}
	result, err := h.service.Apply(r.Context(), request, auth.CurrentUserName(r.Context()))
	if err != nil {
		respondOptimizationError(w, err)
		return
	}
	web.Respond(w, http.StatusCreated, web.Created(result, "Optimized system created successfully"))
}

// @Summary Price and score one explicit vendor line-up without searching
// @Tags Optimization
// @Accept json
// @Produce json
// @Param request body domain.OptimizationScoreRequest true "Score request"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} web.Envelope
// @Failure 404 {object} web.Envelope
// @Failure 500 {object} web.Envelope
// @Security BearerAuth
// @Router /api/Optimization/score [post]
func (h *OptimizationHandler) score(w http.ResponseWriter, r *http.Request) {
	var request domain.OptimizationScoreRequest
	if err := web.DecodeBody(r, &request); err != nil {
		web.Respond(w, http.StatusBadRequest, web.BadRequest(err.Error()))
		return
	}
	result, err := h.service.Score(r.Context(), request)
	if err != nil {
		respondOptimizationError(w, err)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(result, "Line-up scored successfully"))
}
