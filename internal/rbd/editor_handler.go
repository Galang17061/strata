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

type EditorHandler struct {
	drawing       *DrawingService
	afterEdgeSave func(r *http.Request, hierarchyId string)
}

func NewEditorHandler(drawing *DrawingService) *EditorHandler {
	return &EditorHandler{drawing: drawing}
}

func (h *EditorHandler) WithAfterEdgeSave(hook func(r *http.Request, hierarchyId string)) *EditorHandler {
	h.afterEdgeSave = hook
	return h
}

func (h *EditorHandler) Mount(router chi.Router) {
	router.Group(func(protected chi.Router) {
		protected.Use(auth.Require)
		protected.Put("/api/ReliabilityEditor/rbdDrawing/saveNodesByHierarchy", h.saveNodesByHierarchy)
		protected.Put("/api/ReliabilityEditor/rbdDrawing/saveNodesByRbdSystem", h.saveNodesBySystem)
		protected.Put("/api/ReliabilityEditor/rbdDrawing/saveEdgesByHierarchy", h.saveEdgesByHierarchy)
		protected.Put("/api/ReliabilityEditor/rbdDrawing/saveEdgesByRbdSystem", h.saveEdgesBySystem)
		protected.Get("/api/ReliabilityEditor/rbdDrawing/getNodesByHierarchy", h.nodesByHierarchy)
		protected.Get("/api/ReliabilityEditor/rbdDrawing/getNodesByRbdSystem", h.nodesBySystem)
		protected.Get("/api/ReliabilityEditor/rbdDrawing/getEdgesByHierarchy", h.edgesByHierarchy)
		protected.Get("/api/ReliabilityEditor/rbdDrawing/getEdgesByRbdSystem", h.edgesBySystem)
	})
}

func decodeNodes(w http.ResponseWriter, r *http.Request) ([]domain.DrawingNodeInput, bool) {
	var inputs []domain.DrawingNodeInput
	if err := web.DecodeBody(r, &inputs); err != nil {
		web.RespondBodyProblem(w, err)
		return nil, false
	}
	if len(inputs) == 0 {
		web.Respond(w, http.StatusBadRequest, web.BadRequest("No data provided for update."))
		return nil, false
	}
	return inputs, true
}

func decodeEdges(w http.ResponseWriter, r *http.Request) ([]domain.EdgeInput, bool) {
	var inputs []domain.EdgeInput
	if err := web.DecodeBody(r, &inputs); err != nil {
		web.RespondBodyProblem(w, err)
		return nil, false
	}
	problems := map[string][]string{}
	for index, input := range inputs {
		if strings.TrimSpace(domain.Deref(input.IdEdge)) == "" {
			problems["["+web.Itoa(index)+"].IdEdge"] = []string{web.RequiredMessage("IdEdge")}
		}
	}
	if len(problems) > 0 {
		web.RespondValidation(w, problems)
		return nil, false
	}
	if len(inputs) == 0 {
		web.Respond(w, http.StatusBadRequest, web.BadRequest("No data provided for update."))
		return nil, false
	}
	return inputs, true
}

func respondDrawingError(w http.ResponseWriter, err error) {
	var notFound domain.KeyNotFoundError
	var invalid domain.InvalidOperationError
	switch {
	case errors.As(err, &notFound):
		web.Respond(w, http.StatusNotFound, web.NotFound(err.Error()))
	case errors.As(err, &invalid):
		web.Respond(w, http.StatusBadRequest, web.BadRequest(err.Error()))
	default:
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Internal server error: "+err.Error()))
	}
}

func respondEdgeError(w http.ResponseWriter, err error) {
	var notFound domain.KeyNotFoundError
	if errors.As(err, &notFound) {
		web.Respond(w, http.StatusNotFound, web.NotFound(err.Error()+" | StackTrace: "))
		return
	}
	web.Respond(w, http.StatusInternalServerError, web.ServerError("Internal server error: "+err.Error()+" | Inner:  | StackTrace: "))
}

func (h *EditorHandler) saveNodesByHierarchy(w http.ResponseWriter, r *http.Request) {
	inputs, ok := decodeNodes(w, r)
	if !ok {
		return
	}
	if err := h.drawing.SaveNodesByHierarchy(r.Context(), web.QueryString(r, "hierarchyId"), inputs, auth.CurrentUserName(r.Context())); err != nil {
		respondDrawingError(w, err)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(inputs, "Drawings updated successfully."))
}

func (h *EditorHandler) saveNodesBySystem(w http.ResponseWriter, r *http.Request) {
	inputs, ok := decodeNodes(w, r)
	if !ok {
		return
	}
	if err := h.drawing.SaveNodesBySystem(r.Context(), web.QueryString(r, "rbdSystemId"), inputs); err != nil {
		respondDrawingError(w, err)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(inputs, "System level nodes updated successfully."))
}

func (h *EditorHandler) saveEdgesByHierarchy(w http.ResponseWriter, r *http.Request) {
	inputs, ok := decodeEdges(w, r)
	if !ok {
		return
	}
	hierarchyId := web.QueryString(r, "hierarchyId")
	if err := h.drawing.SaveEdgesByHierarchy(r.Context(), hierarchyId, inputs); err != nil {
		respondEdgeError(w, err)
		return
	}
	if h.afterEdgeSave != nil {
		h.afterEdgeSave(r, hierarchyId)
	}
	web.Respond(w, http.StatusOK, web.Success(inputs, "Drawings updated successfully."))
}

func (h *EditorHandler) saveEdgesBySystem(w http.ResponseWriter, r *http.Request) {
	inputs, ok := decodeEdges(w, r)
	if !ok {
		return
	}
	if err := h.drawing.SaveEdgesBySystem(r.Context(), web.QueryString(r, "rbdSystemId"), inputs, auth.CurrentUserName(r.Context())); err != nil {
		respondEdgeError(w, err)
		return
	}
	web.Respond(w, http.StatusOK, web.Success(inputs, "System level edges updated successfully."))
}

func (h *EditorHandler) nodesByHierarchy(w http.ResponseWriter, r *http.Request) {
	hierarchyId := web.QueryString(r, "hierarchyId")
	if strings.TrimSpace(hierarchyId) == "" {
		web.Respond(w, http.StatusBadRequest, web.BadRequest("HierarchyId is required."))
		return
	}
	nodes, err := h.drawing.NodesByHierarchy(r.Context(), hierarchyId)
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Internal server error: "+err.Error()))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(nodes, "Nodes retrieved successfully."))
}

func (h *EditorHandler) nodesBySystem(w http.ResponseWriter, r *http.Request) {
	rbdSystemId := web.QueryString(r, "rbdSystemId")
	if strings.TrimSpace(rbdSystemId) == "" {
		web.Respond(w, http.StatusBadRequest, web.BadRequest("RbdSystemId is required."))
		return
	}
	nodes, err := h.drawing.NodesBySystem(r.Context(), rbdSystemId)
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Internal server error: "+err.Error()))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(nodes, "Nodes retrieved successfully."))
}

func (h *EditorHandler) edgesByHierarchy(w http.ResponseWriter, r *http.Request) {
	hierarchyId := web.QueryString(r, "hierarchyId")
	if strings.TrimSpace(hierarchyId) == "" {
		web.Respond(w, http.StatusBadRequest, web.BadRequest("HierarchyId is required."))
		return
	}
	edges, err := h.drawing.EdgesByHierarchy(r.Context(), hierarchyId)
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Internal server error: "+err.Error()))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(edges, "Edges retrieved successfully."))
}

func (h *EditorHandler) edgesBySystem(w http.ResponseWriter, r *http.Request) {
	rbdSystemId := web.QueryString(r, "rbdSystemId")
	if strings.TrimSpace(rbdSystemId) == "" {
		web.Respond(w, http.StatusBadRequest, web.BadRequest("RbdSystemId is required."))
		return
	}
	edges, err := h.drawing.EdgesBySystem(r.Context(), rbdSystemId)
	if err != nil {
		web.Respond(w, http.StatusInternalServerError, web.ServerError("Internal server error: "+err.Error()))
		return
	}
	web.Respond(w, http.StatusOK, web.Success(edges, "Edges retrieved successfully."))
}
