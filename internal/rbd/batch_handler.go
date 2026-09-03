package rbd

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Galang17061/strata-api/internal/auth"
	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/web"
)

type BatchHandler struct {
	service *BatchService
}

func NewBatchHandler(service *BatchService) *BatchHandler {
	return &BatchHandler{service: service}
}

func (h *BatchHandler) Mount(router chi.Router) {
	router.With(auth.Require).Post("/api/ReliabilityTotal/system/{rbdSystemId}/recalculate", h.recalculate)
}

// @Summary Rescore every part and layer of a system in one call
// @Tags ReliabilityTotal
// @Accept json
// @Produce json
// @Param rbdSystemId path string true "System id"
// @Param request body domain.BatchRecalculateRequest true "Running hours for every part"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/ReliabilityTotal/system/{rbdSystemId}/recalculate [post]
func (h *BatchHandler) recalculate(w http.ResponseWriter, r *http.Request) {
	var request domain.BatchRecalculateRequest
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	if request.RunningHours == nil || *request.RunningHours < 0 {
		web.RespondMessage(w, http.StatusBadRequest, "RunningHours must be zero or more.")
		return
	}
	result, err := h.service.RecalculateSystem(r.Context(), chi.URLParam(r, "rbdSystemId"), *request.RunningHours, auth.CurrentUserName(r.Context()))
	if err != nil {
		web.RespondMessage(w, http.StatusBadRequest, err.Error())
		return
	}
	web.Respond(w, http.StatusOK, web.Success(result, "Every part and layer has been scored again."))
}
