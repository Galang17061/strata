package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func Query(r *http.Request, name string) (string, bool) {
	for key, values := range r.URL.Query() {
		if strings.EqualFold(key, name) && len(values) > 0 {
			return values[0], true
		}
	}
	return "", false
}

func QueryString(r *http.Request, name string) string {
	value, _ := Query(r, name)
	return value
}

func QueryStringOr(r *http.Request, name, fallback string) string {
	value, ok := Query(r, name)
	if !ok {
		return fallback
	}
	return value
}

func InvalidValue(raw string) error {
	return errors.New("The value '" + raw + "' is not valid.")
}

func QueryInt(r *http.Request, name string, fallback int) (int, error) {
	raw, ok := Query(r, name)
	if !ok || raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, InvalidValue(raw)
	}
	return value, nil
}

func QueryOptionalInt(r *http.Request, name string) (*int, error) {
	raw, ok := Query(r, name)
	if !ok || raw == "" {
		return nil, nil
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return nil, InvalidValue(raw)
	}
	return &value, nil
}

func DecodeBody(r *http.Request, target any) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return errors.New("A non-empty request body is required.")
	}
	return json.Unmarshal(body, target)
}

func RespondBodyProblem(w http.ResponseWriter, err error) {
	RespondValidation(w, map[string][]string{"$": {err.Error()}})
}

func RespondFieldProblem(w http.ResponseWriter, field string, err error) {
	RespondValidation(w, map[string][]string{field: {err.Error()}})
}

func RequiredMessage(field string) string {
	return "The " + field + " field is required."
}
