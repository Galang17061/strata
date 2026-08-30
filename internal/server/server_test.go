package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Galang17061/strata-api/internal/config"
)

func TestSwaggerDescriptionAndPageAreServed(t *testing.T) {
	handler := New(config.Config{JWTKey: "k", UploadDir: t.TempDir()}, nil)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"title": "Strata API"`)
	assert.Contains(t, recorder.Body.String(), `"/api/Auth/Login"`)
	assert.Contains(t, recorder.Body.String(), `"/api/ReliabilityTotal/reliabilityPlot"`)

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Header().Get("Content-Type"), "text/html")

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	assert.Equal(t, http.StatusOK, recorder.Code)
}
