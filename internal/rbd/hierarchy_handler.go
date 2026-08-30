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

type HierarchyHandler struct {
	service *HierarchyService
}

func NewHierarchyHandler(service *HierarchyService) *HierarchyHandler {
	return &HierarchyHandler{service: service}
}

func (h *HierarchyHandler) Mount(router chi.Router) {
	router.Group(func(protected chi.Router) {
		protected.Use(auth.Require)
		protected.Get("/api/Hierarchy", h.list)
		protected.Get("/api/Hierarchy/{id}", h.byId)
		protected.Get("/api/Hierarchy/parent/{parentId}", h.byParent)
		protected.Post("/api/Hierarchy", h.create)
		protected.Put("/api/Hierarchy/{id}", h.update)
		protected.Delete("/api/Hierarchy/{id}", h.remove)
		protected.Get("/api/Hierarchy/{hierarchyId}/can-add-component", h.canAddComponent)
	})
}

func (h *HierarchyHandler) list(w http.ResponseWriter, r *http.Request) {
	page, err := web.QueryOptionalInt(r, "page")
	if err != nil {
		web.RespondFieldProblem(w, "page", err)
		return
	}
	pageSize, err := web.QueryOptionalInt(r, "pageSize")
	if err != nil {
		web.RespondFieldProblem(w, "pageSize", err)
		return
	}
	rows, err := h.service.List(r.Context(), web.QueryString(r, "search"), web.QueryString(r, "sortBy"), web.QueryStringOr(r, "sortOrder", "asc"))
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Error retrieving hierarchies: "+err.Error()))
		return
	}
	items, meta := web.PageOptional(rows, page, pageSize)
	web.Respond(w, http.StatusOK, web.SuccessWithMeta(items, "Hierarchies retrieved successfully", meta))
}

func (h *HierarchyHandler) byId(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	view, err := h.service.Find(r.Context(), id)
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Error retrieving hierarchy: "+err.Error()))
		return
	}
	if view == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("Hierarchy with ID "+id+" not found"))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(view, "Hierarchy retrieved successfully"))
}

func (h *HierarchyHandler) byParent(w http.ResponseWriter, r *http.Request) {
	rows, err := h.service.Children(r.Context(), chi.URLParam(r, "parentId"))
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Error retrieving child hierarchies: "+err.Error()))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(rows, "Child hierarchies retrieved successfully"))
}

func lengthProblems(problems map[string][]string, name string, value *string, limit int) {
	if value != nil && len([]rune(*value)) > limit {
		problems[name] = append(problems[name], name+" cannot exceed "+web.Itoa(limit)+" characters")
	}
}

func (h *HierarchyHandler) create(w http.ResponseWriter, r *http.Request) {
	var request domain.HierarchyCreate
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	problems := map[string][]string{}
	if strings.TrimSpace(domain.Deref(request.RbdSystemId)) == "" {
		problems["RbdSystemId"] = []string{"RbdSystemId is required"}
	}
	if strings.TrimSpace(domain.Deref(request.ParentId)) == "" {
		problems["ParentId"] = []string{"ParentId is required"}
	}
	if level := domain.DerefInt(request.Level, 0); level < 1 || level > 3 {
		problems["Level"] = []string{"Level must be between 1 and 3"}
	}
	if strings.TrimSpace(domain.Deref(request.SubSystemName)) == "" {
		problems["SubSystemName"] = []string{"SubSystemName is required"}
	}
	lengthProblems(problems, "SubSystemName", request.SubSystemName, 255)
	lengthProblems(problems, "Formula", request.Formula, 255)
	lengthProblems(problems, "FormulaCode", request.FormulaCode, 20)
	lengthProblems(problems, "ConnectionType", request.ConnectionType, 255)
	if len(problems) > 0 {
		web.RespondValidation(w, problems)
		return
	}
	view, err := h.service.Create(r.Context(), request)
	if err != nil {
		var invalid domain.InvalidOperationError
		if errors.As(err, &invalid) {
			web.Respond(w, http.StatusBadRequest, web.Failed(http.StatusBadRequest, err.Error(), nil))
			return
		}
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Error creating hierarchy: "+err.Error()))
		return
	}
	w.Header().Set("Location", web.AbsoluteURL(r, "/api/Hierarchy/"+view.HierarchyId))
	web.Respond(w, http.StatusCreated, web.Success(view, "Hierarchy created successfully"))
}

func (h *HierarchyHandler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var request domain.HierarchyUpdate
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	problems := map[string][]string{}
	lengthProblems(problems, "SubSystemName", request.SubSystemName, 255)
	lengthProblems(problems, "Formula", request.Formula, 255)
	lengthProblems(problems, "FormulaCode", request.FormulaCode, 20)
	lengthProblems(problems, "ConnectionType", request.ConnectionType, 255)
	if len(problems) > 0 {
		web.RespondValidation(w, problems)
		return
	}
	existing, err := h.service.Find(r.Context(), id)
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Error updating hierarchy: "+err.Error()))
		return
	}
	if existing == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("Hierarchy with ID "+id+" not found"))
		return
	}
	view, err := h.service.Update(r.Context(), id, request)
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Error updating hierarchy: "+err.Error()))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(view, "Hierarchy updated successfully"))
}

func (h *HierarchyHandler) remove(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		var invalid domain.InvalidOperationError
		if errors.As(err, &invalid) {
			web.Respond(w, http.StatusBadRequest, web.Failed(http.StatusBadRequest, err.Error(), nil))
			return
		}
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Error deleting hierarchy: "+err.Error()))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(nil, "Hierarchy deleted successfully"))
}

func (h *HierarchyHandler) canAddComponent(w http.ResponseWriter, r *http.Request) {
	canAdd, err := h.service.CanAddComponent(r.Context(), chi.URLParam(r, "hierarchyId"))
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Error checking component eligibility: "+err.Error()))
		return
	}
	message := "Components can only be added to the highest level"
	if canAdd {
		message = "Components can be added to this hierarchy"
	}
	web.Respond(w, http.StatusOK, web.Success(canAdd, message))
}
