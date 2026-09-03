package rbd

import (
	"context"
	"net/http"
	"strings"

	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/reliability"
	"github.com/Galang17061/strata-api/internal/web"
)

type BatchService struct {
	store      *Store
	parameters *ParameterService
	totals     *TotalService
}

func NewBatchService(store *Store, parameters *ParameterService, totals *TotalService) *BatchService {
	return &BatchService{store: store, parameters: parameters, totals: totals}
}

func envelopeOk(envelope web.Envelope, err error) bool {
	return err == nil && envelope.StatusCode < http.StatusBadRequest
}

func (s *BatchService) refit(ctx context.Context, component domain.SystemComponentProperties, currentUser string) bool {
	id := component.SystemComponentId
	switch strings.ToLower(domain.Deref(component.DistributionType)) {
	case "weibull":
		if !envelopeOk(s.parameters.SeedWeibull(ctx, id, currentUser)) {
			return false
		}
		return envelopeOk(s.parameters.UpdateWeibull(ctx, id, currentUser))
	case "poisson":
		s.parameters.SeedPoisson(ctx, id, currentUser)
		return envelopeOk(s.parameters.UpdatePoisson(ctx, id, currentUser))
	default:
		if !envelopeOk(s.parameters.SeedExponential(ctx, id, currentUser)) {
			return false
		}
		return envelopeOk(s.parameters.UpdateExponential(ctx, id, currentUser))
	}
}

func (s *BatchService) RecalculateSystem(ctx context.Context, rbdSystemId string, runningHours float64, currentUser string) (*domain.BatchRecalculateResult, error) {
	system, err := s.store.FindSystem(ctx, rbdSystemId)
	if err != nil {
		return nil, err
	}
	if system == nil {
		return nil, domain.KeyNotFound("System not found")
	}
	components, err := s.store.ComponentsOfSystem(ctx, rbdSystemId)
	if err != nil {
		return nil, err
	}
	hoursDecimal, err := reliability.FromFloat(runningHours)
	if err != nil {
		return nil, domain.Argument(err.Error())
	}
	hours := domain.NewNumber(hoursDecimal)
	now := domain.Now()
	unfitted := []string{}
	for _, component := range components {
		component.RunningHours = &hours
		component.UpdatedBy = domain.StringPtr(currentUser)
		component.UpdatedAt = &now
		if err := s.store.UpdateComponent(ctx, component); err != nil {
			return nil, err
		}
		if !s.refit(ctx, component, currentUser) {
			unfitted = append(unfitted, component.ComponentName)
		}
	}
	roots, err := s.store.RootHierarchies(ctx, rbdSystemId)
	if err != nil {
		return nil, err
	}
	calcCtx := WithUser(ctx, currentUser)
	for _, root := range roots {
		envelope := s.totals.CalculateHierarchy(calcCtx, root.HierarchyId)
		if envelope.StatusCode >= http.StatusBadRequest {
			return nil, domain.InvalidOperation(envelope.Message)
		}
	}
	refreshed, err := s.store.FindSystem(ctx, rbdSystemId)
	if err != nil {
		return nil, err
	}
	var total *float64
	if refreshed != nil && refreshed.ReliabilityTotal != nil {
		parsed := reliability.ToFloat(refreshed.ReliabilityTotal.Decimal)
		total = &parsed
	}
	return &domain.BatchRecalculateResult{
		Components:       len(components),
		Unfitted:         unfitted,
		ReliabilityTotal: total,
	}, nil
}
