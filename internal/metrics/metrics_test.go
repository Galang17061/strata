package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestThePulseCountsEveryVisitByMethodAndOutcome(t *testing.T) {
	collector := NewCollector()
	handler := collector.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/thing", nil))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/thing", nil))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/missing", nil))

	rendered := collector.Render()
	assert.Contains(t, rendered, `strata_requests_total{method="GET",status="2xx"} 2`)
	assert.Contains(t, rendered, `strata_requests_total{method="POST",status="4xx"} 1`)
	assert.Contains(t, rendered, "strata_request_seconds_count 3")
	assert.Contains(t, rendered, "strata_uptime_seconds")
	assert.Contains(t, rendered, "strata_goroutines")
}

func TestReadingThePulseLeavesNoFingerprint(t *testing.T) {
	collector := NewCollector()
	inner := collector.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		collector.Handler(w, r)
	}))
	inner.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/metrics", nil))
	assert.Contains(t, collector.Render(), "strata_request_seconds_count 0")
}
