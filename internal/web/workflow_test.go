package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type hierarchySnapshot struct {
	HierarchyId   string `json:"hierarchyId"`
	RbdSystemId   string `json:"rbdSystemId"`
	Level         int    `json:"level"`
	SubSystemName string `json:"subSystemName"`
	Formula       string `json:"formula"`
	FormulaCode   string `json:"formulaCode"`
	SourceId      string `json:"sourceId"`
	TargetId      string `json:"targetId"`
}

func workflowServer(t *testing.T, calls *int32) *httptest.Server {
	levels := map[string]hierarchySnapshot{
		"H-L3-COMPONENTS": {"H-L3-COMPONENTS", "RBD-WF-TEST-001", 3, "Component Group", "1-((1-C-1)*(1-C-2))", "HC-1", "C-1", "C-2"},
		"H-L2-SUBSYSTEM":  {"H-L2-SUBSYSTEM", "RBD-WF-TEST-001", 2, "Brake Assembly", "1-((1-C-1)*(1-C-2))", "HS-2", "HC-1", "HC-1"},
	}
	mux := NewRouter()
	mux.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(calls, 1)
			next.ServeHTTP(w, r)
		})
	})
	mux.Post("/api/MasterSystem/createRbdSystem", func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		Respond(w, http.StatusCreated, Created(map[string]any{
			"rbdSystemId": "RBD-WF-TEST-001",
			"projectId":   payload["projectId"],
			"systemName":  payload["systemName"],
			"drawingName": "Brake RBD",
		}, "RBD System created successfully"))
	})
	mux.Put("/api/ReliabilityEditor/rbdDrawing/saveEdges", func(w http.ResponseWriter, r *http.Request) {
		var edges []map[string]string
		require.NoError(t, json.NewDecoder(r.Body).Decode(&edges))
		assert.Len(t, edges, 4)
		assert.Equal(t, "RBD-WF-TEST-001", QueryString(r, "rbdSystemId"))
		Respond(w, http.StatusOK, Success([]any{}, "Edges updated successfully"))
	})
	mux.Get("/api/Hierarchy/{id}", func(w http.ResponseWriter, r *http.Request) {
		Respond(w, http.StatusOK, Success(levels[chi.URLParam(r, "id")], "Hierarchy retrieved successfully"))
	})
	return httptest.NewServer(mux)
}

func send(t *testing.T, client *http.Client, method, url string, body any) (*http.Response, Envelope) {
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		encoded, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, url, reader)
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	var envelope Envelope
	require.NoError(t, json.NewDecoder(response.Body).Decode(&envelope))
	return response, envelope
}

func snapshotOf(t *testing.T, data any) hierarchySnapshot {
	encoded, err := json.Marshal(data)
	require.NoError(t, err)
	var snapshot hierarchySnapshot
	require.NoError(t, json.Unmarshal(encoded, &snapshot))
	return snapshot
}

func TestSavedParallelWiringShowsUpAtEveryLevelAbove(t *testing.T) {
	var calls int32
	server := workflowServer(t, &calls)
	defer server.Close()
	client := server.Client()

	createResponse, created := send(t, client, http.MethodPost, server.URL+"/api/MasterSystem/createRbdSystem", map[string]any{
		"projectId":  "PJ-00001",
		"systemName": "Brake System",
		"hierarchy":  []any{},
	})
	assert.Equal(t, http.StatusCreated, createResponse.StatusCode)
	assert.Equal(t, "success", created.Status)
	assert.Equal(t, 201, created.StatusCode)
	assert.Equal(t, "RBD-WF-TEST-001", created.Data.(map[string]any)["rbdSystemId"])

	edgesResponse, saved := send(t, client, http.MethodPut, server.URL+"/api/ReliabilityEditor/rbdDrawing/saveEdges?rbdSystemId=RBD-WF-TEST-001", []map[string]string{
		{"idEdge": "edge-1", "sourceId": "IN", "targetId": "00000711"},
		{"idEdge": "edge-2", "sourceId": "IN", "targetId": "00000712"},
		{"idEdge": "edge-3", "sourceId": "00000711", "targetId": "OUT"},
		{"idEdge": "edge-4", "sourceId": "00000712", "targetId": "OUT"},
	})
	assert.Equal(t, http.StatusOK, edgesResponse.StatusCode)
	assert.Equal(t, "Edges updated successfully", saved.Message)
	assert.Equal(t, "application/json; charset=utf-8", edgesResponse.Header.Get("Content-Type"))

	levelThreeResponse, levelThree := send(t, client, http.MethodGet, server.URL+"/api/Hierarchy/H-L3-COMPONENTS", nil)
	assert.Equal(t, http.StatusOK, levelThreeResponse.StatusCode)
	components := snapshotOf(t, levelThree.Data)
	assert.True(t, strings.Contains(components.Formula, "1-((1-"), components.Formula)
	assert.Contains(t, components.Formula, "C-1")
	assert.Contains(t, components.Formula, "C-2")
	assert.Equal(t, "C-1", components.SourceId)
	assert.Equal(t, "C-2", components.TargetId)

	levelTwoResponse, levelTwo := send(t, client, http.MethodGet, server.URL+"/api/hierarchy/H-L2-SUBSYSTEM", nil)
	assert.Equal(t, http.StatusOK, levelTwoResponse.StatusCode)
	subsystem := snapshotOf(t, levelTwo.Data)
	assert.True(t, strings.Contains(subsystem.Formula, "1-((1-"), subsystem.Formula)
	assert.Contains(t, subsystem.Formula, "C-1")
	assert.Contains(t, subsystem.Formula, "C-2")
	assert.NotEmpty(t, subsystem.SourceId)
	assert.NotEmpty(t, subsystem.TargetId)
	assert.Equal(t, components.Formula, subsystem.Formula)

	assert.Equal(t, int32(4), atomic.LoadInt32(&calls))
}
