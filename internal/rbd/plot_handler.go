package rbd

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/Galang17061/strata-api/internal/auth"
	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/web"
)

type PlotHandler struct {
	service *PlotService
}

func NewPlotHandler(service *PlotService) *PlotHandler {
	return &PlotHandler{service: service}
}

func (h *PlotHandler) Mount(router chi.Router) {
	router.Group(func(protected chi.Router) {
		protected.Use(auth.Require)
		protected.Put("/api/ReliabilityTotal/update-running-hours", h.updateRunningHours)
		protected.Get("/api/ReliabilityTotal/reliabilityPlot", h.systemPlot)
		protected.Get("/api/ReliabilityPlotComponent", h.list)
		protected.Get("/api/ReliabilityPlotComponent/{rbdSystemId}", h.byId)
		protected.Post("/api/ReliabilityPlotComponent", h.create)
		protected.Put("/api/ReliabilityPlotComponent/{reliabilityPlotComponentId}", h.update)
		protected.Delete("/api/ReliabilityPlotComponent/{reliabilityPlotComponentId}", h.remove)
	})
}

// @Summary Update the running hours used to evaluate the reliability of a system
// @Tags ReliabilityTotal
// @Produce json
// @Param rbdSystemId query string true "System id"
// @Param runningHours query number false "Running hours, zero or more"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} web.ValidationProblem
// @Failure 404 {object} web.Envelope
// @Failure 500 {object} web.Envelope
// @Security BearerAuth
// @Router /api/ReliabilityTotal/update-running-hours [put]
func (h *PlotHandler) updateRunningHours(w http.ResponseWriter, r *http.Request) {
	rbdSystemId := web.QueryString(r, "rbdSystemId")
	runningHours := decimal.Zero
	if raw, ok := web.Query(r, "runningHours"); ok && strings.TrimSpace(raw) != "" {
		parsed, err := decimal.NewFromString(strings.TrimSpace(raw))
		if err != nil {
			web.RespondFieldProblem(w, "runningHours", web.InvalidValue(raw))
			return
		}
		runningHours = parsed
	}
	if strings.TrimSpace(rbdSystemId) == "" {
		web.Respond(w, http.StatusBadRequest, web.BadRequest("RBD System ID is required"))
		return
	}
	if runningHours.Sign() < 0 {
		web.Respond(w, http.StatusBadRequest, web.BadRequest("Running hours must be a non-negative value"))
		return
	}
	if err := h.service.UpdateRunningHours(r.Context(), rbdSystemId, runningHours, auth.CurrentUserName(r.Context())); err != nil {
		var notFound domain.KeyNotFoundError
		if errors.As(err, &notFound) {
			web.Respond(w, http.StatusNotFound, web.NotFound(err.Error()))
			return
		}
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Internal server error: "+err.Error()))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(nil, "Running hours updated successfully"))
}

// @Summary Get the reliability plot time series of a system
// @Tags ReliabilityTotal
// @Produce json
// @Param rbdSystemId query string true "System id"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} web.Envelope
// @Failure 500 {object} web.Envelope
// @Security BearerAuth
// @Router /api/ReliabilityTotal/reliabilityPlot [get]
func (h *PlotHandler) systemPlot(w http.ResponseWriter, r *http.Request) {
	rbdSystemId := web.QueryString(r, "rbdSystemId")
	if strings.TrimSpace(rbdSystemId) == "" {
		web.Respond(w, http.StatusBadRequest, web.BadRequest("RBD System ID is required"))
		return
	}
	points, err := h.service.SystemPlot(r.Context(), rbdSystemId)
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Internal server error: "+err.Error()))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(points, "Reliability plot data retrieved successfully"))
}

// @Summary List reliability plot components one page at a time
// @Tags ReliabilityPlotComponent
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Items per page"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} web.ValidationProblem
// @Failure 500 {string} string
// @Security BearerAuth
// @Router /api/ReliabilityPlotComponent [get]
func (h *PlotHandler) list(w http.ResponseWriter, r *http.Request) {
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
	rows, err := h.service.All(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	items, meta := web.Page(rows, page, pageSize)
	web.Respond(w, http.StatusOK, web.SuccessWithMeta(items, "Success", meta))
}

// @Summary Show a single reliability plot component
// @Tags ReliabilityPlotComponent
// @Produce json
// @Param rbdSystemId path string true "System id"
// @Param reliabilityPlotComponentId query string true "Reliability plot component id"
// @Success 200 {object} web.Envelope
// @Failure 404 {object} web.Envelope
// @Failure 500 {string} string
// @Security BearerAuth
// @Router /api/ReliabilityPlotComponent/{rbdSystemId} [get]
func (h *PlotHandler) byId(w http.ResponseWriter, r *http.Request) {
	row, err := h.service.Find(r.Context(), web.QueryString(r, "reliabilityPlotComponentId"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if row == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("Data not found"))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(row, "Success"))
}

// @Summary Add a component to a reliability plot
// @Tags ReliabilityPlotComponent
// @Accept json
// @Produce json
// @Param request body domain.PlotCreate true "Reliability plot component to create"
// @Success 201 {object} web.Envelope
// @Failure 400 {object} web.ValidationProblem
// @Failure 500 {string} string
// @Security BearerAuth
// @Router /api/ReliabilityPlotComponent [post]
func (h *PlotHandler) create(w http.ResponseWriter, r *http.Request) {
	var request domain.PlotCreate
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	problems := map[string][]string{}
	if strings.TrimSpace(domain.Deref(request.ReliabilityPlotId)) == "" {
		problems["ReliabilityPlotId"] = []string{web.RequiredMessage("ReliabilityPlotId")}
	}
	if strings.TrimSpace(domain.Deref(request.SystemComponentId)) == "" {
		problems["SystemComponentId"] = []string{web.RequiredMessage("SystemComponentId")}
	}
	if len(problems) > 0 {
		web.RespondValidation(w, problems)
		return
	}
	if err := h.service.Add(r.Context(), request, auth.CurrentUserName(r.Context())); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Location", web.AbsoluteURL(r, "/api/ReliabilityPlotComponent/"+domain.Deref(request.ReliabilityPlotId)))
	web.Respond(w, http.StatusCreated, web.Created(request, "Data created successfully"))
}

// @Summary Change the component linked to a reliability plot entry
// @Tags ReliabilityPlotComponent
// @Accept json
// @Produce json
// @Param reliabilityPlotComponentId path string true "Reliability plot component id"
// @Param request body domain.PlotUpdate true "New values for the reliability plot component"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} web.ValidationProblem
// @Failure 404 {object} web.Envelope
// @Failure 500 {string} string
// @Security BearerAuth
// @Router /api/ReliabilityPlotComponent/{reliabilityPlotComponentId} [put]
func (h *PlotHandler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "reliabilityPlotComponentId")
	var request domain.PlotUpdate
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	if strings.TrimSpace(domain.Deref(request.SystemComponentId)) == "" {
		web.RespondValidation(w, map[string][]string{"SystemComponentId": {web.RequiredMessage("SystemComponentId")}})
		return
	}
	existing, err := h.service.Find(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if existing == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("Data not found"))
		return
	}
	if err := h.service.Update(r.Context(), id, request, auth.CurrentUserName(r.Context())); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(request, "Data updated successfully"))
}

// @Summary Remove a component from a reliability plot
// @Tags ReliabilityPlotComponent
// @Produce json
// @Param reliabilityPlotComponentId path string true "Reliability plot component id"
// @Success 200 {object} web.Envelope
// @Failure 404 {object} web.Envelope
// @Failure 500 {string} string
// @Security BearerAuth
// @Router /api/ReliabilityPlotComponent/{reliabilityPlotComponentId} [delete]
func (h *PlotHandler) remove(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "reliabilityPlotComponentId")
	existing, err := h.service.Find(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if existing == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("Data not found"))
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(nil, "Data deleted successfully"))
}
