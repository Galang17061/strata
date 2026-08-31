package rbd

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/reliability"
	"github.com/Galang17061/strata-api/internal/web"
)

func (s *ParameterService) PoissonViews(ctx context.Context, systemComponentId string) ([]domain.PoissonParameterView, error) {
	component, err := s.store.FindComponent(ctx, systemComponentId)
	if err != nil {
		return nil, err
	}
	views := []domain.PoissonParameterView{}
	if component == nil {
		return views, nil
	}
	allowance := domain.DerefInt(component.AllowedFailures, 0)
	rows, err := s.store.PoissonParametersByHours(ctx, systemComponentId)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		createdAt := row.CreatedAt
		updatedAt := row.UpdatedAt
		view := domain.PoissonParameterView{
			PoissonParameterId: row.PoissonParameterId,
			SystemComponentId:  row.SystemComponentId,
			FailureTime:        domain.NumberPtr(domain.NumberFromInt(int64(row.FailureEventHours))),
			FailureRate:        domain.NumberPtr(row.Rate),
			AllowedFailures:    domain.IntPtr(allowance),
			CreatedAt:          &createdAt,
			UpdatedAt:          &updatedAt,
		}
		lambda := reliability.ToFloat(row.Rate.Decimal)
		t := float64(row.FailureEventHours)
		if lambda > 0 && t >= 0 {
			value := reliability.PoissonFloat(lambda, t, allowance)
			if math.IsNaN(value) || math.IsInf(value, 0) {
				view.TotalReliability = domain.NumberPtr(domain.NumberFromInt(0))
			} else {
				clamped, err := reliability.FromFloat(math.Min(math.Max(value, 0), 1))
				if err != nil {
					return nil, err
				}
				view.TotalReliability = domain.NumberPtr(domain.NewNumber(clamped))
			}
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *ParameterService) SeedPoisson(ctx context.Context, systemComponentId, currentUser string) (web.Envelope, error) {
	var envelope web.Envelope
	err := s.store.Transact(ctx, func(tx *Store) error {
		events, err := tx.FailureEventsByHours(ctx, systemComponentId)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			envelope = web.BadRequest("Poisson calculation requires at least 1 failure event. Please add failure event data.")
			return nil
		}
		component, err := tx.FindComponent(ctx, systemComponentId)
		if err != nil {
			return err
		}
		if component == nil {
			envelope = web.BadRequest("System Component not found")
			return nil
		}
		if err := tx.DeletePoissonParametersOfComponent(ctx, systemComponentId); err != nil {
			return err
		}
		lastId, err := tx.LastPoissonIdOfOthers(ctx, systemComponentId)
		if err != nil {
			return err
		}
		lastNumber := 0
		if lastId != nil {
			parsed, err := strconv.Atoi((*lastId)[minInt(3, len(*lastId)):])
			if err != nil {
				return domain.Argument("Input string was not in a correct format.")
			}
			lastNumber = parsed
		}
		rows := make([]domain.PoissonParameter, 0, len(events))
		for index, event := range events {
			n := index + 1
			rate := poissonEventRate(n, event.RunningHours)
			rateValue, err := reliability.FromFloat(rate)
			if err != nil {
				return err
			}
			lastNumber++
			now := domain.Now()
			rows = append(rows, domain.PoissonParameter{
				PoissonParameterId: fmt.Sprintf("PP-%05d", lastNumber),
				SystemComponentId:  systemComponentId,
				FailureEventHours:  event.RunningHours,
				N:                  n,
				Rate:               domain.NewNumber(rateValue),
				CreatedAt:          now,
				UpdatedAt:          now,
				CreatedBy:          domain.StringPtr(currentUser),
				UpdatedBy:          domain.StringPtr(currentUser),
			})
		}
		if err := tx.InsertPoissonParameters(ctx, rows); err != nil {
			return err
		}
		if component.FailureRate != nil && component.RunningHours != nil {
			value := reliability.PoissonFromFloats(component.FailureRate.Decimal, component.RunningHours.Decimal, domain.DerefInt(component.AllowedFailures, 0))
			component.DistributionType = domain.StringPtr("Poisson")
			component.ReliabilityValue = domain.NumberPtr(domain.NewNumber(value))
			component.UpdatedBy = domain.StringPtr(currentUser)
			now := domain.Now()
			component.UpdatedAt = &now
			if err := tx.UpdateComponent(ctx, *component); err != nil {
				return err
			}
		}
		envelope = web.Created(rows, "Data created successfully")
		return nil
	})
	return envelope, err
}

func poissonEventRate(n, hours int) float64 {
	if hours <= 0 {
		return 0
	}
	return float64(n) / float64(hours)
}
