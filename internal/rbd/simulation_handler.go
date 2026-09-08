package rbd

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/Galang17061/strata-api/internal/auth"
	"github.com/Galang17061/strata-api/internal/jobs"
	"github.com/Galang17061/strata-api/internal/reliability"
	"github.com/Galang17061/strata-api/internal/web"
)

type SimulationHandler struct {
	service *SimulationService
	queue   *jobs.Store
}

func NewSimulationHandler(service *SimulationService, queue *jobs.Store) *SimulationHandler {
	return &SimulationHandler{service: service, queue: queue}
}

func (h *SimulationHandler) Mount(router chi.Router) {
	router.With(auth.Require).Post("/api/Simulation/system/{rbdSystemId}/monte-carlo", h.enqueue)
}

// @Summary Queue a Monte Carlo rehearsal of a system
// @Tags Simulation
// @Accept json
// @Produce json
// @Param rbdSystemId path string true "System id"
// @Param request body rbd.SimulationRequest true "Mission hours, trials and an optional seed"
// @Success 200 {object} web.Envelope
// @Failure 400 {object} web.ValidationProblem
// @Security BearerAuth
// @Router /api/Simulation/system/{rbdSystemId}/monte-carlo [post]
func (h *SimulationHandler) enqueue(w http.ResponseWriter, r *http.Request) {
	var request SimulationRequest
	if err := web.DecodeBody(r, &request); err != nil {
		web.RespondBodyProblem(w, err)
		return
	}
	request.RbdSystemId = chi.URLParam(r, "rbdSystemId")
	problems := map[string][]string{}
	if request.MissionHours < 0 {
		problems["MissionHours"] = []string{"Mission hours cannot be negative."}
	}
	if request.Trials < 0 || request.Trials > reliability.SimulationTrialsCeiling {
		problems["Trials"] = []string{"Trials must sit between 1 and 500000."}
	}
	if request.CurvePoints < 0 || request.CurvePoints > 200 {
		problems["CurvePoints"] = []string{"Curve points must stay under 200."}
	}
	if len(problems) > 0 {
		web.RespondValidation(w, problems)
		return
	}
	system, err := h.service.store.FindSystem(r.Context(), request.RbdSystemId)
	if err != nil {
		web.RespondMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if system == nil {
		web.RespondMessage(w, http.StatusNotFound, "System not found")
		return
	}
	if full, message := h.queueIsFull(r.Context()); full {
		web.Respond(w, http.StatusTooManyRequests, web.Failed(http.StatusTooManyRequests, message, nil))
		return
	}
	payload, err := json.Marshal(request)
	if err != nil {
		web.RespondMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	job, err := h.queue.Enqueue(r.Context(), SimulationKind, string(payload), &request.RbdSystemId, auth.CurrentUserName(r.Context()), jobs.OwnerId(r.Context()))
	if err != nil {
		web.RespondMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	web.Respond(w, http.StatusOK, web.Success(job.View(), "The rehearsal is queued. Watch its job for the result."))
}

func (h *SimulationHandler) queueIsFull(ctx context.Context) (bool, string) {
	waiting, err := h.queue.CountUnfinished(ctx, "")
	if err == nil && waiting >= jobs.QueueCeiling {
		return true, "The queue is full at the moment. Give the workers a minute and try again."
	}
	if jobs.Keeper(ctx) {
		return false, ""
	}
	owner := jobs.OwnerId(ctx)
	if owner == "" {
		return false, ""
	}
	mine, err := h.queue.CountUnfinished(ctx, owner)
	if err == nil && mine >= jobs.QueuePerUser {
		return true, "You already have " + strconv.Itoa(mine) + " rehearsals waiting. Let them finish before asking for another."
	}
	return false, ""
}
