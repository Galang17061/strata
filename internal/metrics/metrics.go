package metrics

import (
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Collector struct {
	mu       sync.Mutex
	started  time.Time
	requests map[string]uint64
	seconds  float64
	count    uint64
}

func NewCollector() *Collector {
	return &Collector{started: time.Now(), requests: map[string]uint64{}}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (c *Collector) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		key := r.Method + "|" + strconv.Itoa(recorder.status/100) + "xx"
		elapsed := time.Since(start).Seconds()
		c.mu.Lock()
		c.requests[key]++
		c.seconds += elapsed
		c.count++
		c.mu.Unlock()
	})
}

func (c *Collector) Render() string {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	c.mu.Lock()
	keys := make([]string, 0, len(c.requests))
	for key := range c.requests {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out strings.Builder
	out.WriteString("# TYPE strata_uptime_seconds gauge\n")
	out.WriteString("strata_uptime_seconds " + strconv.FormatFloat(time.Since(c.started).Seconds(), 'f', 0, 64) + "\n")
	out.WriteString("# TYPE strata_requests_total counter\n")
	for _, key := range keys {
		parts := strings.SplitN(key, "|", 2)
		out.WriteString("strata_requests_total{method=\"" + parts[0] + "\",status=\"" + parts[1] + "\"} " + strconv.FormatUint(c.requests[key], 10) + "\n")
	}
	out.WriteString("# TYPE strata_request_seconds_sum counter\n")
	out.WriteString("strata_request_seconds_sum " + strconv.FormatFloat(c.seconds, 'f', 6, 64) + "\n")
	out.WriteString("# TYPE strata_request_seconds_count counter\n")
	out.WriteString("strata_request_seconds_count " + strconv.FormatUint(c.count, 10) + "\n")
	c.mu.Unlock()
	out.WriteString("# TYPE strata_goroutines gauge\n")
	out.WriteString("strata_goroutines " + strconv.Itoa(runtime.NumGoroutine()) + "\n")
	out.WriteString("# TYPE strata_heap_alloc_bytes gauge\n")
	out.WriteString("strata_heap_alloc_bytes " + strconv.FormatUint(memory.HeapAlloc, 10) + "\n")
	return out.String()
}

func (c *Collector) Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Write([]byte(c.Render()))
}
