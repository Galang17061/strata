package web

import (
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func NewRouter() *chi.Mux {
	mux := chi.NewRouter()
	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowedHeaders: []string{"*"},
		ExposedHeaders: []string{"Content-Disposition"},
	}))
	mux.Use(CanonicalPaths(mux))
	return mux
}

func CanonicalPaths(routes chi.Routes) func(http.Handler) http.Handler {
	var once sync.Once
	var patterns [][]string
	load := func() {
		_ = chi.Walk(routes, func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
			patterns = append(patterns, splitPath(route))
			return nil
		})
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			once.Do(load)
			if canonical, ok := canonicalize(r.URL.Path, patterns); ok {
				r.URL.Path = canonical
				r.URL.RawPath = ""
			}
			next.ServeHTTP(w, r)
		})
	}
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "/")
}

func canonicalize(path string, patterns [][]string) (string, bool) {
	segments := splitPath(path)
	for _, pattern := range patterns {
		if len(pattern) != len(segments) {
			continue
		}
		matched := true
		rewritten := make([]string, len(segments))
		for index, part := range pattern {
			if strings.HasPrefix(part, "{") {
				rewritten[index] = segments[index]
				continue
			}
			if !strings.EqualFold(part, segments[index]) {
				matched = false
				break
			}
			rewritten[index] = part
		}
		if matched {
			return "/" + strings.Join(rewritten, "/"), true
		}
	}
	return path, false
}
