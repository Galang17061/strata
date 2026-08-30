package rbd

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Galang17061/strata-api/internal/auth"
	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/web"
)

type SystemHandler struct {
	service *SystemService
}

func NewSystemHandler(service *SystemService) *SystemHandler {
	return &SystemHandler{service: service}
}

func (h *SystemHandler) Mount(router chi.Router) {
	router.Group(func(protected chi.Router) {
		protected.Use(auth.Require)
		protected.Get("/api/MasterSystem/systemList", h.systemList)
		protected.Get("/api/MasterSystem/systemByProject", h.systemByProject)
		protected.Get("/api/MasterSystem/getRbdTreeView/{rbdSystemId}", h.tree)
		protected.Put("/api/MasterSystem/updateRbdTreeView/{rbdSystemId}", h.updateTree)
		protected.Post("/api/MasterSystem/createRbdSystem", h.create)
		protected.Put("/api/MasterSystem/{rbdSystemId}", h.updateSystem)
		protected.Get("/api/MasterSystem/hierarchy/{hierarchyId}/input-parameters", h.inputParameters)
		protected.Get("/api/MasterSystem/hierarchy/{hierarchyId}/plot-graphic", h.plotGraphic)
		protected.Delete("/api/MasterSystem/{rbdSystemId}", h.deleteSystem)
	})
}

// @Summary List every reliability block diagram system in Strata
// @Tags MasterSystem
// @Produce json
// @Success 200 {object} web.Envelope
// @Failure 500 {string} string
// @Security BearerAuth
// @Router /api/MasterSystem/systemList [get]
func (h *SystemHandler) systemList(w http.ResponseWriter, r *http.Request) {
	rows, err := h.service.ListSystems(r.Context(), nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(rows, "Success"))
}

// @Summary List the systems that belong to one project
// @Tags MasterSystem
// @Produce json
// @Param projectId query string false "Project id"
// @Success 200 {object} web.Envelope
// @Failure 500 {string} string
// @Security BearerAuth
// @Router /api/MasterSystem/systemByProject [get]
func (h *SystemHandler) systemByProject(w http.ResponseWriter, r *http.Request) {
	var projectId *string
	if value, ok := web.Query(r, "projectId"); ok {
		projectId = &value
	}
	rows, err := h.service.ListSystems(r.Context(), projectId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(rows, "Success"))
}

// @Summary Retrieve a system together with its full hierarchy tree
// @Tags MasterSystem
// @Produce json
// @Param rbdSystemId path string true "System id"
// @Param systemName query string false "System name"
// @Success 200 {object} web.Envelope
// @Failure 404 {object} web.Envelope
// @Failure 500 {object} web.Envelope
// @Security BearerAuth
// @Router /api/MasterSystem/getRbdTreeView/{rbdSystemId} [get]
func (h *SystemHandler) tree(w http.ResponseWriter, r *http.Request) {
	var systemName *string
	if value, ok := web.Query(r, "systemName"); ok {
		systemName = &value
	}
	tree, err := h.service.Tree(r.Context(), chi.URLParam(r, "rbdSystemId"), systemName)
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Error retrieving RBD System: "+err.Error()))
		return
	}
	if tree == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("RBD System not found for the specified project and system name"))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(tree, "RBD System with complete hierarchy retrieved successfully"))
}

// @Summary Replace the hierarchy tree of a system
// @Tags MasterSystem
// @Accept json
// @Produce json
// @Param rbdSystemId path string true "System id"
// @Param request body domain.SystemUpdate true "System name, project and hierarchy tree"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} web.ValidationProblem
// @Failure 404 {object} web.Envelope
// @Failure 500 {object} web.Envelope
// @Security BearerAuth
// @Router /api/MasterSystem/updateRbdTreeView/{rbdSystemId} [put]
func (h *SystemHandler) updateTree(w http.ResponseWriter, r *http.Request) {
	var request domain.SystemUpdate
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	h.applyUpdate(w, r, request)
}

// @Summary Update a system and its hierarchy tree
// @Tags MasterSystem
// @Accept json
// @Produce json
// @Param rbdSystemId path string true "System id"
// @Param request body domain.SystemCreate true "System name, project and hierarchy tree"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} web.ValidationProblem
// @Failure 404 {object} web.Envelope
// @Failure 500 {object} web.Envelope
// @Security BearerAuth
// @Router /api/MasterSystem/{rbdSystemId} [put]
func (h *SystemHandler) updateSystem(w http.ResponseWriter, r *http.Request) {
	var request domain.SystemCreate
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	h.applyUpdate(w, r, domain.SystemUpdate{ProjectId: request.ProjectId, SystemName: request.SystemName, Hierarchy: request.Hierarchy})
}

func (h *SystemHandler) applyUpdate(w http.ResponseWriter, r *http.Request, request domain.SystemUpdate) {
	rbdSystemId := chi.URLParam(r, "rbdSystemId")
	if rbdSystemId == "" {
		web.Respond(w, http.StatusBadRequest, web.Failed(http.StatusBadRequest, "RBD System ID is required", nil))
		return
	}
	tree, err := h.service.Update(r.Context(), rbdSystemId, request, auth.CurrentUserName(r.Context()))
	if err != nil {
		var invalid domain.InvalidOperationError
		if errors.As(err, &invalid) {
			web.Respond(w, http.StatusNotFound, web.NotFound(err.Error()))
			return
		}
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Error updating RBD System: "+err.Error()))
		return
	}
	if tree == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("RBD System not found"))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(tree, "RBD System with complete hierarchy updated successfully"))
}

// @Summary Create a system with its hierarchy levels and components
// @Tags MasterSystem
// @Accept json
// @Produce json
// @Param request body domain.SystemCreate true "System name, project and hierarchy tree"
// @Success 201 {object} web.Envelope
// @Failure 400 {object} web.ValidationProblem
// @Failure 500 {object} web.Envelope
// @Security BearerAuth
// @Router /api/MasterSystem/createRbdSystem [post]
func (h *SystemHandler) create(w http.ResponseWriter, r *http.Request) {
	var request domain.SystemCreate
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	rbdSystemId, err := h.service.Create(r.Context(), request, auth.CurrentUserName(r.Context()))
	if err != nil {
		var invalid domain.InvalidOperationError
		if errors.As(err, &invalid) {
			web.Respond(w, http.StatusBadRequest, web.BadRequest(err.Error()))
			return
		}
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Error creating RBD System: "+err.Error()))
		return
	}
	w.Header().Set("Location", web.AbsoluteURL(r, "/api/MasterSystem/getRbdTreeView/"+rbdSystemId))
	web.Respond(w, http.StatusCreated, web.Created(map[string]string{"rbdSystemId": rbdSystemId}, "RBD System with hierarchies and components created successfully"))
}

// @Summary Retrieve the component input parameters of a hierarchy level
// @Tags MasterSystem
// @Produce json
// @Param hierarchyId path string true "Hierarchy id"
// @Success 200 {object} web.Envelope
// @Failure 500 {object} web.Envelope
// @Security BearerAuth
// @Router /api/MasterSystem/hierarchy/{hierarchyId}/input-parameters [get]
func (h *SystemHandler) inputParameters(w http.ResponseWriter, r *http.Request) {
	rows, err := h.service.InputParameters(r.Context(), chi.URLParam(r, "hierarchyId"))
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Error retrieving component input parameters: "+err.Error()))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(rows, "Component input parameters retrieved successfully"))
}

// @Summary Retrieve the component input and output values used to plot a hierarchy level
// @Tags MasterSystem
// @Produce json
// @Param hierarchyId path string true "Hierarchy id"
// @Success 200 {object} web.Envelope
// @Failure 500 {object} web.Envelope
// @Security BearerAuth
// @Router /api/MasterSystem/hierarchy/{hierarchyId}/plot-graphic [get]
func (h *SystemHandler) plotGraphic(w http.ResponseWriter, r *http.Request) {
	rows, err := h.service.InputOutputParameters(r.Context(), chi.URLParam(r, "hierarchyId"))
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Error retrieving component input/output parameters: "+err.Error()))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(rows, "Component input/output parameters retrieved successfully"))
}

// @Summary Delete a system and everything beneath it
// @Tags MasterSystem
// @Produce json
// @Param rbdSystemId path string true "System id"
// @Success 200 {object} web.Envelope
// @Failure 404 {object} web.Envelope
// @Failure 500 {string} string
// @Security BearerAuth
// @Router /api/MasterSystem/{rbdSystemId} [delete]
func (h *SystemHandler) deleteSystem(w http.ResponseWriter, r *http.Request) {
	first, err := h.service.FirstSystem(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if first == nil {
		web.Respond(w, http.StatusNotFound, web.NotFound("Data not found"))
		return
	}
	if err := h.service.Delete(r.Context(), chi.URLParam(r, "rbdSystemId")); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(nil, "Data deleted successfully"))
}
