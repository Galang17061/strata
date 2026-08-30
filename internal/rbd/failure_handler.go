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

type FailureHandler struct {
	failures   *FailureService
	parameters *ParameterService
}

func NewFailureHandler(failures *FailureService, parameters *ParameterService) *FailureHandler {
	return &FailureHandler{failures: failures, parameters: parameters}
}

func (h *FailureHandler) Mount(router chi.Router) {
	router.Group(func(protected chi.Router) {
		protected.Use(auth.Require)
		protected.Post("/api/ReliabilityEditor/failureEvent", h.create)
		protected.Get("/api/ReliabilityEditor/failureEvent", h.list)
		protected.Delete("/api/ReliabilityEditor/failureEvent/{failureEventHistoryId}", h.remove)
		protected.Delete("/api/ReliabilityEditor/failureEvent/bySystemComponent/{systemComponentId}", h.removeAll)
		protected.Put("/api/SystemComponentProperties/{systemComponentId}/exponential", h.exponential)
		protected.Post("/api/SystemComponentProperties/{systemComponentId}/exponential-parameter", h.exponentialSeed)
	})
}

// @Summary Record failure events in the history of one or more components
// @Tags ReliabilityEditor
// @Accept json
// @Produce json
// @Param request body []domain.FailureEventCreate true "Failure events to record"
// @Success 201 {object} web.Envelope
// @Failure 400 {object} web.ValidationProblem
// @Failure 500 {string} string
// @Security BearerAuth
// @Router /api/ReliabilityEditor/failureEvent [post]
func (h *FailureHandler) create(w http.ResponseWriter, r *http.Request) {
	var inputs []domain.FailureEventCreate
	if err := web.DecodeBody(r, &inputs); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	problems := map[string][]string{}
	for index, input := range inputs {
		if strings.TrimSpace(domain.Deref(input.SystemComponentId)) == "" {
			problems["["+web.Itoa(index)+"].SystemComponentId"] = []string{web.RequiredMessage("SystemComponentId")}
		}
	}
	if len(problems) > 0 {
		web.RespondValidation(w, problems)
		return
	}
	if err := h.failures.Add(r.Context(), inputs, auth.CurrentUserName(r.Context())); err != nil {
		var invalid domain.InvalidOperationError
		if errors.As(err, &invalid) {
			web.Respond(w, http.StatusBadRequest, web.BadRequest(err.Error()))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Location", web.AbsoluteURL(r, "/api/ReliabilityEditor/failureEvent"))
	web.Respond(w, http.StatusCreated, web.Created(web.NonNil(inputs), "Data created successfully"))
}

// @Summary List failure event history one page at a time
// @Tags ReliabilityEditor
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Items per page"
// @Param systemComponentId query string false "System component id"
// @Param search query string false "Search text"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} web.ValidationProblem
// @Failure 500 {string} string
// @Security BearerAuth
// @Router /api/ReliabilityEditor/failureEvent [get]
func (h *FailureHandler) list(w http.ResponseWriter, r *http.Request) {
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
	rows, err := h.failures.List(r.Context(), web.QueryString(r, "systemComponentId"), web.QueryString(r, "search"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	items, meta := web.Page(rows, page, pageSize)
	web.Respond(w, http.StatusOK, web.SuccessWithMeta(items, "Success", meta))
}

// @Summary Delete a single failure event from the history
// @Tags ReliabilityEditor
// @Produce json
// @Param failureEventHistoryId path string true "Failure event history id"
// @Success 200 {object} web.Envelope
// @Failure 404 {object} web.Envelope
// @Failure 500 {string} string
// @Security BearerAuth
// @Router /api/ReliabilityEditor/failureEvent/{failureEventHistoryId} [delete]
func (h *FailureHandler) remove(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "failureEventHistoryId")
	event, err := h.failures.Find(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if event == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("Data not found"))
		return
	}
	if err := h.failures.Delete(r.Context(), id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(nil, "Data deleted successfully"))
}

// @Summary Delete every failure event recorded for a system component
// @Tags ReliabilityEditor
// @Produce json
// @Param systemComponentId path string true "System component id"
// @Success 200 {object} web.Envelope
// @Failure 404 {object} web.Envelope
// @Failure 500 {string} string
// @Security BearerAuth
// @Router /api/ReliabilityEditor/failureEvent/bySystemComponent/{systemComponentId} [delete]
func (h *FailureHandler) removeAll(w http.ResponseWriter, r *http.Request) {
	systemComponentId := chi.URLParam(r, "systemComponentId")
	rows, err := h.failures.List(r.Context(), systemComponentId, "")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(rows) == 0 {
		web.Respond(w, http.StatusNotFound, web.NotFound("No failure events found for this system component"))
		return
	}
	if err := h.failures.DeleteAllOfComponent(r.Context(), systemComponentId); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(nil, "All failure events deleted successfully"))
}

// @Summary Recalculate the exponential failure rate of a component from its failure history
// @Tags SystemComponentProperties
// @Produce json
// @Param systemComponentId path string true "System component id"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} web.Envelope
// @Failure 500 {object} web.Envelope
// @Security BearerAuth
// @Router /api/SystemComponentProperties/{systemComponentId}/exponential [put]
func (h *FailureHandler) exponential(w http.ResponseWriter, r *http.Request) {
	systemComponentId := chi.URLParam(r, "systemComponentId")
	if strings.TrimSpace(systemComponentId) == "" {
		web.Respond(w, http.StatusBadRequest, web.Failed(http.StatusBadRequest, "SystemComponentId is required", nil))
		return
	}
	envelope, err := h.parameters.UpdateExponential(r.Context(), systemComponentId, auth.CurrentUserName(r.Context()))
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

// @Summary Create the initial exponential distribution entry for a component
// @Tags SystemComponentProperties
// @Produce json
// @Param systemComponentId path string true "System component id"
// @Success 201 {object} web.Envelope
// @Failure 400 {object} web.Envelope
// @Failure 500 {object} web.Envelope
// @Security BearerAuth
// @Router /api/SystemComponentProperties/{systemComponentId}/exponential-parameter [post]
func (h *FailureHandler) exponentialSeed(w http.ResponseWriter, r *http.Request) {
	systemComponentId := chi.URLParam(r, "systemComponentId")
	envelope, err := h.parameters.SeedExponential(r.Context(), systemComponentId, auth.CurrentUserName(r.Context()))
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
