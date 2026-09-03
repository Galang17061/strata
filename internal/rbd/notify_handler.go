package rbd

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/Galang17061/strata-api/internal/auth"
	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/reliability"
	"github.com/Galang17061/strata-api/internal/web"
)

type NotifyHandler struct {
	store *Store
}

func NewNotifyHandler(store *Store) *NotifyHandler {
	return &NotifyHandler{store: store}
}

func (h *NotifyHandler) Mount(router chi.Router) {
	router.Group(func(protected chi.Router) {
		protected.Use(auth.Require)
		protected.Get("/api/Notification", h.recent)
		protected.Get("/api/MasterSystem/{rbdSystemId}/threshold", h.threshold)
		protected.Put("/api/MasterSystem/{rbdSystemId}/threshold", h.setThreshold)
	})
}

// @Summary List the latest reliability alerts
// @Tags Notification
// @Produce json
// @Param limit query int false "How many to return"
// @Success 200 {object} web.Envelope
// @Security BearerAuth
// @Router /api/Notification [get]
func (h *NotifyHandler) recent(w http.ResponseWriter, r *http.Request) {
	limit, err := web.QueryInt(r, "limit", 15)
	if err != nil || limit < 1 || limit > 100 {
		limit = 15
	}
	rows, err := h.store.RecentNotifications(r.Context(), limit)
	if err != nil {
		web.RespondMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	web.Respond(w, http.StatusOK, web.Success(rows, "Success"))
}

// @Summary Show the reliability floor set for a system, if any
// @Tags Notification
// @Produce json
// @Param rbdSystemId path string true "System id"
// @Success 200 {object} web.Envelope
// @Security BearerAuth
// @Router /api/MasterSystem/{rbdSystemId}/threshold [get]
func (h *NotifyHandler) threshold(w http.ResponseWriter, r *http.Request) {
	threshold, err := h.store.FindThreshold(r.Context(), chi.URLParam(r, "rbdSystemId"))
	if err != nil {
		web.RespondMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	var value *float64 = nil
	if threshold != nil {
		parsed := reliability.ToFloat(threshold.Decimal)
		value = &parsed
	}
	web.Respond(w, http.StatusOK, web.Success(domain.ThresholdView{Threshold: value}, "Success"))
}

// @Summary Set or clear the reliability floor that raises an alert
// @Tags Notification
// @Accept json
// @Produce json
// @Param rbdSystemId path string true "System id"
// @Param request body domain.ThresholdUpdate true "Floor between 0 and 1, or null to clear"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} map[string]string
// @Security BearerAuth
// @Router /api/MasterSystem/{rbdSystemId}/threshold [put]
func (h *NotifyHandler) setThreshold(w http.ResponseWriter, r *http.Request) {
	var request domain.ThresholdUpdate
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	rbdSystemId := chi.URLParam(r, "rbdSystemId")
	if request.Threshold == nil {
		if err := h.store.SetThreshold(r.Context(), rbdSystemId, nil, auth.CurrentUserName(r.Context())); err != nil {
			web.RespondMessage(w, http.StatusBadRequest, err.Error())
			return
		}
		web.Respond(w, http.StatusOK, web.Success(nil, "The floor has been cleared."))
		return
	}
	if *request.Threshold <= 0 || *request.Threshold >= 1 {
		web.RespondMessage(w, http.StatusBadRequest, "The floor must sit between 0 and 1.")
		return
	}
	number := domain.NewNumber(decimal.NewFromFloat(*request.Threshold))
	if err := h.store.SetThreshold(r.Context(), rbdSystemId, &number, auth.CurrentUserName(r.Context())); err != nil {
		web.RespondMessage(w, http.StatusBadRequest, err.Error())
		return
	}
	web.Respond(w, http.StatusOK, web.Success(nil, "The floor is set. An alert will sound if the system falls beneath it."))
}
