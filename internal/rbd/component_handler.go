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

type ComponentHandler struct {
	service *ComponentService
}

func NewComponentHandler(service *ComponentService) *ComponentHandler {
	return &ComponentHandler{service: service}
}

func (h *ComponentHandler) Mount(router chi.Router) {
	router.Group(func(protected chi.Router) {
		protected.Use(auth.Require)
		protected.Get("/api/SystemComponentProperties/list", h.list)
		protected.Get("/api/SystemComponentProperties/{systemComponentId}", h.detail)
		protected.Post("/api/SystemComponentProperties", h.create)
		protected.Put("/api/SystemComponentProperties/{systemComponentId}", h.update)
		protected.Delete("/api/SystemComponentProperties/{systemComponentId}", h.remove)
		protected.Get("/api/SystemMonitoring/GetDataComponent", h.monitoring)
		protected.Get("/api/SystemMonitoring/GetRBDCalculation", h.calculation)
	})
}

type listQuery struct {
	page      *int
	pageSize  *int
	search    string
	sortBy    string
	sortOrder string
}

func readListQuery(w http.ResponseWriter, r *http.Request) (listQuery, bool) {
	page, err := web.QueryOptionalInt(r, "page")
	if err != nil {
		web.RespondFieldProblem(w, "page", err)
		return listQuery{}, false
	}
	pageSize, err := web.QueryOptionalInt(r, "pageSize")
	if err != nil {
		web.RespondFieldProblem(w, "pageSize", err)
		return listQuery{}, false
	}
	return listQuery{page: page, pageSize: pageSize, search: web.QueryString(r, "search"), sortBy: web.QueryString(r, "sortBy"), sortOrder: web.QueryStringOr(r, "sortOrder", "asc")}, true
}

// @Summary List component properties with optional search, sorting and paging
// @Tags SystemComponentProperties
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Rows per page"
// @Param search query string false "Search text"
// @Param sortBy query string false "Sort field"
// @Param sortOrder query string false "Sort direction, asc or desc"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} web.ValidationProblem
// @Failure 500 {string} string
// @Security BearerAuth
// @Router /api/SystemComponentProperties/list [get]
func (h *ComponentHandler) list(w http.ResponseWriter, r *http.Request) {
	query, ok := readListQuery(w, r)
	if !ok {
		return
	}
	rows, err := h.service.ScpList(r.Context(), query.search, query.sortBy, query.sortOrder)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	items, meta := web.PageOptional(rows, query.page, query.pageSize)
	web.Respond(w, http.StatusOK, web.SuccessWithMeta(items, "System component properties retrieved successfully", meta))
}

// @Summary Retrieve the properties of one component
// @Tags SystemComponentProperties
// @Produce json
// @Param systemComponentId path string true "Component id"
// @Success 200 {object} web.Envelope
// @Failure 404 {object} web.Envelope
// @Failure 500 {string} string
// @Security BearerAuth
// @Router /api/SystemComponentProperties/{systemComponentId} [get]
func (h *ComponentHandler) detail(w http.ResponseWriter, r *http.Request) {
	detail, err := h.service.Detail(r.Context(), chi.URLParam(r, "systemComponentId"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if detail == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("Data not found"))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(detail, "Success"))
}

// @Summary Add a component beneath a hierarchy level
// @Tags SystemComponentProperties
// @Accept json
// @Produce json
// @Param request body domain.ComponentSimpleCreate true "Parent hierarchy id and component name"
// @Success 201 {object} web.Envelope
// @Failure 400 {object} web.ValidationProblem
// @Failure 500 {object} web.Envelope
// @Security BearerAuth
// @Router /api/SystemComponentProperties [post]
func (h *ComponentHandler) create(w http.ResponseWriter, r *http.Request) {
	var request domain.ComponentSimpleCreate
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	problems := map[string][]string{}
	if strings.TrimSpace(domain.Deref(request.ParentId)) == "" {
		problems["ParentId"] = []string{web.RequiredMessage("ParentId")}
	}
	if strings.TrimSpace(domain.Deref(request.ComponentName)) == "" {
		problems["ComponentName"] = []string{web.RequiredMessage("ComponentName")}
	}
	if len(problems) > 0 {
		web.RespondValidation(w, problems)
		return
	}
	detail, err := h.service.CreateSimple(r.Context(), request, auth.CurrentUserName(r.Context()))
	if err != nil {
		var invalid domain.InvalidOperationError
		if errors.As(err, &invalid) {
			web.Respond(w, http.StatusBadRequest, web.BadRequest(err.Error()))
			return
		}
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Internal server error: "+err.Error()))
		return
	}
	w.Header().Set("Location", web.AbsoluteURL(r, "/api/SystemComponentProperties/"+detail.SystemComponentId))
	web.Respond(w, http.StatusCreated, web.Created(detail, "Component created successfully"))
}

// @Summary Update the properties of a component
// @Tags SystemComponentProperties
// @Accept json
// @Produce json
// @Param systemComponentId path string true "Component id"
// @Param request body domain.ComponentDetailUpdate true "Component properties to change"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} web.ValidationProblem
// @Failure 404 {object} web.Envelope
// @Failure 500 {string} string
// @Security BearerAuth
// @Router /api/SystemComponentProperties/{systemComponentId} [put]
func (h *ComponentHandler) update(w http.ResponseWriter, r *http.Request) {
	systemComponentId := chi.URLParam(r, "systemComponentId")
	var request domain.ComponentDetailUpdate
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	detail, err := h.service.UpdateDetail(r.Context(), systemComponentId, request, auth.CurrentUserName(r.Context()))
	if err != nil {
		var invalid domain.InvalidOperationError
		if errors.As(err, &invalid) {
			web.Respond(w, http.StatusBadRequest, web.BadRequest(err.Error()))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if detail == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("Data not found"))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(detail, "Data updated successfully"))
}

// @Summary Delete a component
// @Tags SystemComponentProperties
// @Produce json
// @Param systemComponentId path string true "Component id"
// @Success 200 {object} web.Envelope
// @Failure 404 {object} web.Envelope
// @Failure 500 {string} string
// @Security BearerAuth
// @Router /api/SystemComponentProperties/{systemComponentId} [delete]
func (h *ComponentHandler) remove(w http.ResponseWriter, r *http.Request) {
	systemComponentId := chi.URLParam(r, "systemComponentId")
	component, err := h.service.Find(r.Context(), systemComponentId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if component == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("Data not found"))
		return
	}
	if err := h.service.Delete(r.Context(), systemComponentId); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(nil, "Data deleted successfully"))
}

// @Summary List monitored components with their running hours and reliability
// @Tags SystemMonitoring
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Rows per page"
// @Param search query string false "Search text"
// @Param sortBy query string false "Sort field"
// @Param sortOrder query string false "Sort direction, asc or desc"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} web.ValidationProblem
// @Failure 500 {object} web.Envelope
// @Security BearerAuth
// @Router /api/SystemMonitoring/GetDataComponent [get]
func (h *ComponentHandler) monitoring(w http.ResponseWriter, r *http.Request) {
	query, ok := readListQuery(w, r)
	if !ok {
		return
	}
	rows, err := h.service.MonitoredComponents(r.Context(), query.search, query.sortBy, query.sortOrder)
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError(err.Error()))
		return
	}
	items, meta := web.PageOptional(rows, query.page, query.pageSize)
	web.Respond(w, http.StatusOK, web.SuccessWithMeta(items, "Success", meta))
}

// @Summary Retrieve the reliability calculation summary across all systems
// @Tags SystemMonitoring
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Rows per page"
// @Param search query string false "Search text"
// @Param sortBy query string false "Sort field"
// @Param sortOrder query string false "Sort direction, asc or desc"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} web.ValidationProblem
// @Failure 500 {object} web.Envelope
// @Security BearerAuth
// @Router /api/SystemMonitoring/GetRBDCalculation [get]
func (h *ComponentHandler) calculation(w http.ResponseWriter, r *http.Request) {
	if _, err := web.QueryInt(r, "page", 1); err != nil {
		web.RespondFieldProblem(w, "page", err)
		return
	}
	if _, err := web.QueryInt(r, "pageSize", 10); err != nil {
		web.RespondFieldProblem(w, "pageSize", err)
		return
	}
	summary, err := h.service.CalculationSummary(r.Context(), web.QueryString(r, "search"), web.QueryString(r, "sortBy"), web.QueryStringOr(r, "sortOrder", "asc"))
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError(err.Error()))
		return
	}
	web.Respond(w, http.StatusOK, web.SuccessWithMeta(summary, "Success", web.NewMeta(1, 1, 1, 1)))
}
