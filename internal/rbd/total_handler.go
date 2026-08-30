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

func (h *TotalHandler) hierarchyReliability(w http.ResponseWriter, r *http.Request) {
	hierarchyId := chi.URLParam(r, "hierarchyId")
	if strings.TrimSpace(hierarchyId) == "" {
		web.Respond(w, http.StatusBadRequest, web.BadRequest("Hierarchy ID is required"))
		return
	}
	respondByCode(w, h.service.CalculateHierarchy(WithUser(r.Context(), auth.CurrentUserName(r.Context())), hierarchyId))
}

func (h *TotalHandler) systemTotal(w http.ResponseWriter, r *http.Request) {
	rbdSystemId := chi.URLParam(r, "rbdSystemId")
	if strings.TrimSpace(rbdSystemId) == "" {
		web.Respond(w, http.StatusBadRequest, web.BadRequest("RBD System ID is required"))
		return
	}
	respondByCode(w, h.service.SystemTotal(r.Context(), rbdSystemId))
}

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
