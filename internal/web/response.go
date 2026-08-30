package web

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
)

type Meta struct {
	TotalData       int  `json:"totalData"`
	TotalPage       int  `json:"totalPage"`
	CurrentPage     int  `json:"currentPage"`
	PageSize        int  `json:"pageSize"`
	HasNextPage     bool `json:"hasNextPage"`
	HasPreviousPage bool `json:"hasPreviousPage"`
}

func NewMeta(totalData, totalPage, currentPage, pageSize int) *Meta {
	return &Meta{
		TotalData:       totalData,
		TotalPage:       totalPage,
		CurrentPage:     currentPage,
		PageSize:        pageSize,
		HasNextPage:     currentPage < totalPage,
		HasPreviousPage: currentPage > 1,
	}
}

type Envelope struct {
	Status     string `json:"status"`
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Data       any    `json:"data"`
	Meta       *Meta  `json:"meta"`
}

func NewEnvelope(statusCode int, message string, data any, meta *Meta) Envelope {
	status := "failed"
	if statusCode >= 200 && statusCode < 300 {
		status = "success"
	}
	return Envelope{Status: status, StatusCode: statusCode, Message: message, Data: data, Meta: meta}
}

func Success(data any, message string) Envelope {
	return NewEnvelope(http.StatusOK, message, data, nil)
}

func SuccessWithMeta(data any, message string, meta *Meta) Envelope {
	return NewEnvelope(http.StatusOK, message, data, meta)
}

func Created(data any, message string) Envelope {
	return NewEnvelope(http.StatusCreated, message, data, nil)
}

func NotFound(message string) Envelope {
	return NewEnvelope(http.StatusNotFound, message, nil, nil)
}

func BadRequest(message string) Envelope {
	return NewEnvelope(http.StatusBadRequest, message, nil, nil)
}

func ServerError(message string) Envelope {
	return NewEnvelope(http.StatusInternalServerError, message, nil, nil)
}

func Failed(statusCode int, message string, data any) Envelope {
	return NewEnvelope(statusCode, message, data, nil)
}

type ExceptionBody struct {
	StatusCode            int    `json:"statusCode"`
	IsModelValidatonError bool   `json:"isModelValidatonError"`
	Errors                any    `json:"errors"`
	CustomError           any    `json:"customError"`
	ReferenceErrorCode    any    `json:"referenceErrorCode"`
	ReferenceDocumentLink any    `json:"referenceDocumentLink"`
	Message               string `json:"message"`
}

func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func Respond(w http.ResponseWriter, status int, env Envelope) {
	WriteJSON(w, status, env)
}

func RespondEnvelope(w http.ResponseWriter, env Envelope) {
	WriteJSON(w, env.StatusCode, env)
}

func RespondMessage(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"message": message})
}

func RespondException(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, ExceptionBody{StatusCode: status, Message: message})
}

type ValidationProblem struct {
	Type    string              `json:"type"`
	Title   string              `json:"title"`
	Status  int                 `json:"status"`
	TraceID string              `json:"traceId"`
	Errors  map[string][]string `json:"errors"`
}

func RespondValidation(w http.ResponseWriter, errors map[string][]string) {
	body := ValidationProblem{
		Type:    "https://tools.ietf.org/html/rfc7231#section-6.5.1",
		Title:   "One or more validation errors occurred.",
		Status:  http.StatusBadRequest,
		TraceID: newTraceID(),
		Errors:  errors,
	}
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(body)
}

func newTraceID() string {
	traceBytes := make([]byte, 16)
	spanBytes := make([]byte, 8)
	_, _ = rand.Read(traceBytes)
	_, _ = rand.Read(spanBytes)
	return "00-" + hex.EncodeToString(traceBytes) + "-" + hex.EncodeToString(spanBytes) + "-00"
}
