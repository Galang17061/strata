package rbd

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Galang17061/strata-api/internal/auth"
	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/web"
)

type TotalHandler struct {
	service *TotalService
}

func NewTotalHandler(service *TotalService) *TotalHandler {
	return &TotalHandler{service: service}
}

func (h *TotalHandler) Mount(router chi.Router) {
	router.Group(func(protected chi.Router) {
		protected.Use(auth.Require)
		protected.Put("/api/ReliabilityTotal/updateFormula", h.updateFormula)
		protected.Get("/api/ReliabilityTotal/hierarchy/{hierarchyId}/reliability", h.hierarchyReliability)
		protected.Get("/api/ReliabilityTotal/rbdSystem/{rbdSystemId}/reliability-total", h.systemTotal)
		protected.Get("/api/ReliabilityTotal/hierarchy/{hierarchyId}/history", h.history)
		protected.Delete("/api/ReliabilityTotal/reliability-history/{historyId}", h.deleteHistory)
	})
}

func (h *TotalHandler) AfterEdgeSave(r *http.Request, hierarchyId string) {
	h.service.SaveHistoryAfterEdgeSave(r.Context(), hierarchyId, auth.CurrentUserName(r.Context()))
}

// @Summary Store the reliability formula of a system
// @Tags ReliabilityTotal
// @Produce json
// @Param rbdSystemId query string true "System id"
// @Param formula query string true "Reliability formula"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} web.Envelope
// @Failure 404 {object} web.Envelope
// @Failure 500 {object} web.Envelope
// @Security BearerAuth
// @Router /api/ReliabilityTotal/updateFormula [put]
func (h *TotalHandler) updateFormula(w http.ResponseWriter, r *http.Request) {
	rbdSystemId := web.QueryString(r, "rbdSystemId")
	formula := web.QueryString(r, "formula")
	if strings.TrimSpace(rbdSystemId) == "" {
		web.Respond(w, http.StatusBadRequest, web.BadRequest("RBD System ID is required"))
		return
	}
	if strings.TrimSpace(formula) == "" {
		web.Respond(w, http.StatusBadRequest, web.BadRequest("Formula is required"))
		return
	}
	result, err := h.service.UpdateFormula(r.Context(), rbdSystemId, formula)
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Internal server error: "+err.Error()))
		return
	}
	if strings.TrimSpace(result) == "" {
		web.Respond(w, http.StatusNotFound, web.NotFound("RBD System not found"))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(result, "Formula updated successfully"))
}

func respondByCode(w http.ResponseWriter, envelope web.Envelope) {
	switch envelope.StatusCode {
	case http.StatusNotFound:
		web.Respond(w, http.StatusNotFound, envelope)
	case http.StatusBadRequest:
		web.Respond(w, http.StatusBadRequest, envelope)
	default:
		web.Respond(w, http.StatusOK, envelope)
	}
}

// @Summary Calculate the reliability of a hierarchy level from its block diagram
// @Tags ReliabilityTotal
// @Produce json
// @Param hierarchyId path string true "Hierarchy id"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} web.Envelope
// @Failure 404 {object} web.Envelope
// @Security BearerAuth
// @Router /api/ReliabilityTotal/hierarchy/{hierarchyId}/reliability [get]
func (h *TotalHandler) hierarchyReliability(w http.ResponseWriter, r *http.Request) {
	hierarchyId := chi.URLParam(r, "hierarchyId")
	if strings.TrimSpace(hierarchyId) == "" {
		web.Respond(w, http.StatusBadRequest, web.BadRequest("Hierarchy ID is required"))
		return
	}
	respondByCode(w, h.service.CalculateHierarchy(WithUser(r.Context(), auth.CurrentUserName(r.Context())), hierarchyId))
}

// @Summary Calculate the total reliability of a system across its hierarchy levels
// @Tags ReliabilityTotal
// @Produce json
// @Param rbdSystemId path string true "System id"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} web.Envelope
// @Failure 404 {object} web.Envelope
// @Security BearerAuth
// @Router /api/ReliabilityTotal/rbdSystem/{rbdSystemId}/reliability-total [get]
func (h *TotalHandler) systemTotal(w http.ResponseWriter, r *http.Request) {
	rbdSystemId := chi.URLParam(r, "rbdSystemId")
	if strings.TrimSpace(rbdSystemId) == "" {
		web.Respond(w, http.StatusBadRequest, web.BadRequest("RBD System ID is required"))
		return
	}
	respondByCode(w, h.service.SystemTotal(r.Context(), rbdSystemId))
}

// @Summary List the reliability calculation history of a hierarchy level one page at a time
// @Tags ReliabilityTotal
// @Produce json
// @Param hierarchyId path string true "Hierarchy id"
// @Param page query int false "Page number"
// @Param pageSize query int false "Items per page"
// @Param sortBy query string false "Field to sort by"
// @Param sortOrder query string false "Sort direction, asc or desc"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} web.ValidationProblem
// @Failure 404 {object} web.Envelope
// @Failure 500 {object} web.Envelope
// @Security BearerAuth
// @Router /api/ReliabilityTotal/hierarchy/{hierarchyId}/history [get]
func (h *TotalHandler) history(w http.ResponseWriter, r *http.Request) {
	hierarchyId := chi.URLParam(r, "hierarchyId")
	page, err := web.QueryInt(r, "page", 1)
	if err != nil {
		web.RespondFieldProblem(w, "page", err)
		return
	}
	pageSize, err := web.QueryInt(r, "pageSize", 10)
	if err != nil {
		web.RespondFieldProblem(w, "pageSize", err)
		return
	}
	if strings.TrimSpace(hierarchyId) == "" {
		web.Respond(w, http.StatusBadRequest, web.BadRequest("Hierarchy ID is required"))
		return
	}
	envelope, err := h.service.HistoryPage(r.Context(), hierarchyId, page, pageSize, web.QueryString(r, "sortBy"), web.QueryStringOr(r, "sortOrder", "desc"))
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Internal server error: "+err.Error()))
		return
	}
	web.RespondEnvelope(w, envelope)
}

// @Summary Delete one entry from the reliability calculation history
// @Tags ReliabilityTotal
// @Produce json
// @Param historyId path string true "Calculation history id"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} web.Envelope
// @Failure 404 {object} web.Envelope
// @Failure 500 {object} web.Envelope
// @Security BearerAuth
// @Router /api/ReliabilityTotal/reliability-history/{historyId} [delete]
func (h *TotalHandler) deleteHistory(w http.ResponseWriter, r *http.Request) {
	historyId := chi.URLParam(r, "historyId")
	if strings.TrimSpace(historyId) == "" {
		web.Respond(w, http.StatusBadRequest, web.BadRequest("History ID is required"))
		return
	}
	if err := h.service.DeleteHistory(r.Context(), historyId); err != nil {
		var notFound domain.KeyNotFoundError
		if errors.As(err, &notFound) {
			web.Respond(w, http.StatusNotFound, web.NotFound(err.Error()))
			return
		}
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Internal server error: "+err.Error()))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(nil, "Reliability history deleted successfully"))
}
