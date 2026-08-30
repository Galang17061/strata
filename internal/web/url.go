package web

import (
	"net/http"
	"strconv"
)

func AbsoluteURL(r *http.Request, path string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		scheme = forwarded
	}
	return scheme + "://" + r.Host + path
}

func Itoa(value int) string {
	return strconv.Itoa(value)
}
