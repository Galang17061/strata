package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestRouterMatchesPathsRegardlessOfCase(t *testing.T) {
	mux := NewRouter()
	mux.Get("/api/MasterProject/{projectId}", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(chi.URLParam(r, "projectId")))
	})
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/masterproject/PJ-00001/", nil))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "PJ-00001", recorder.Body.String())
}

func TestQueryReadsNamesRegardlessOfCase(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/x?RbdSystemId=RS-1&Page=abc", nil)
	assert.Equal(t, "RS-1", QueryString(request, "rbdSystemId"))
	_, err := QueryInt(request, "page", 1)
	assert.EqualError(t, err, "The value 'abc' is not valid.")
	value, err := QueryInt(request, "pageSize", 10)
	assert.NoError(t, err)
	assert.Equal(t, 10, value)
}

func TestMetaFlagsFollowCurrentPage(t *testing.T) {
	meta := NewMeta(25, 3, 2, 10)
	assert.True(t, meta.HasNextPage)
	assert.True(t, meta.HasPreviousPage)
	edge := NewMeta(0, 1, 1, 0)
	assert.False(t, edge.HasNextPage)
	assert.False(t, edge.HasPreviousPage)
}

func TestEnvelopeStatusFollowsCode(t *testing.T) {
	assert.Equal(t, "success", Created(nil, "x").Status)
	assert.Equal(t, "failed", NotFound("x").Status)
}
