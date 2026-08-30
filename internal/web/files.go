package web

import (
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const maxMultipartMemory = 32 << 20

func StaticFiles(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		relative := strings.TrimPrefix(r.URL.Path, "/files")
		cleaned := path.Clean("/" + relative)
		target := filepath.Join(root, filepath.FromSlash(cleaned))
		info, err := os.Stat(target)
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, target)
	}
}

type Form struct {
	values map[string][]string
	files  map[string][]*multipart.FileHeader
}

func ParseForm(r *http.Request) (*Form, error) {
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
			return nil, err
		}
		return &Form{values: r.MultipartForm.Value, files: r.MultipartForm.File}, nil
	}
	if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			return nil, err
		}
		return &Form{values: r.PostForm, files: map[string][]*multipart.FileHeader{}}, nil
	}
	return nil, errors.New("unsupported media type")
}

func (f *Form) Value(name string) (string, bool) {
	for key, values := range f.values {
		if strings.EqualFold(key, name) && len(values) > 0 {
			return values[0], true
		}
	}
	return "", false
}

func (f *Form) File(name string) *multipart.FileHeader {
	for key, files := range f.files {
		if strings.EqualFold(key, name) && len(files) > 0 {
			return files[0]
		}
	}
	return nil
}

type UnsupportedMediaType struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	Status  int    `json:"status"`
	TraceID string `json:"traceId"`
}

func RespondUnsupportedMediaType(w http.ResponseWriter) {
	body := UnsupportedMediaType{
		Type:    "https://tools.ietf.org/html/rfc7231#section-6.5.13",
		Title:   "Unsupported Media Type",
		Status:  http.StatusUnsupportedMediaType,
		TraceID: newTraceID(),
	}
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(http.StatusUnsupportedMediaType)
	_ = json.NewEncoder(w).Encode(body)
}

func AttachmentHeader(w http.ResponseWriter, contentType, fileName string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename="+fileName+"; filename*=UTF-8''"+fileName)
}
