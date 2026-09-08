package rbd

import (
	"context"
	"encoding/json"

	"github.com/Galang17061/strata-api/internal/jobs"
)

const SimulationKind = "simulation"

type SimulationRunner struct {
	service *SimulationService
}

func NewSimulationRunner(service *SimulationService) *SimulationRunner {
	return &SimulationRunner{service: service}
}

func (r *SimulationRunner) Kind() string {
	return SimulationKind
}

func (r *SimulationRunner) Run(ctx context.Context, job jobs.Job) (string, error) {
	var request SimulationRequest
	if err := json.Unmarshal([]byte(job.Request), &request); err != nil {
		return "", err
	}
	summary, err := r.service.Run(ctx, request)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(summary)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
