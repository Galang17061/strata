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

func (s *ParameterService) WeibullViews(ctx context.Context, systemComponentId string) ([]domain.WeibullParameterView, error) {
	component, err := s.store.FindComponent(ctx, systemComponentId)
	if err != nil {
		return nil, err
	}
	views := []domain.WeibullParameterView{}
	if component == nil {
		return views, nil
	}
	rows, err := s.store.WeibullParametersByHours(ctx, systemComponentId)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		createdAt := row.CreatedAt
		updatedAt := row.UpdatedAt
		view := domain.WeibullParameterView{
			WeibullParameterId: row.WeibullParameterId,
			SystemComponentId:  row.SystemComponentId,
			FailureTime:        domain.NumberPtr(domain.NumberFromInt(int64(row.FailureEventHours))),
			ScaleParameter:     domain.NumberPtr(row.X),
			ShapeParameter:     domain.NumberPtr(row.Y),
			CreatedAt:          &createdAt,
			UpdatedAt:          &updatedAt,
		}
		theta := reliability.ToFloat(row.X.Decimal)
		beta := reliability.ToFloat(row.Y.Decimal)
		t := float64(row.FailureEventHours)
		if theta > 0 && beta > 0 && t >= 0 {
			value := reliability.WeibullFloat(t, theta, beta)
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

func (s *ParameterService) SeedWeibull(ctx context.Context, systemComponentId, currentUser string) (web.Envelope, error) {
	var envelope web.Envelope
	err := s.store.Transact(ctx, func(tx *Store) error {
		events, err := tx.FailureEventsByHours(ctx, systemComponentId)
		if err != nil {
			return err
		}
		if len(events) < 2 {
			envelope = web.BadRequest("Weibull calculation requires at least 2 failure events. Please add more failure event data.")
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
		if err := tx.DeleteWeibullParametersOfComponent(ctx, systemComponentId); err != nil {
			return err
		}
		lastId, err := tx.LastWeibullIdOfOthers(ctx, systemComponentId)
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
		rows := make([]domain.WeibullParameter, 0, len(events))
		for index, event := range events {
			n := index + 1
			frequency := reliability.MedianRank(n, len(events))
			frequencyValue, err := reliability.FromFloat(frequency)
			if err != nil {
				return err
			}
			x, y := reliability.WeibullPlotPoint(event.RunningHours, frequency)
			xValue, err := reliability.FromFloat(x)
			if err != nil {
				return err
			}
			yValue, err := reliability.FromFloat(y)
			if err != nil {
				return err
			}
			lastNumber++
			now := domain.Now()
			rows = append(rows, domain.WeibullParameter{
				WeibullParameterId: fmt.Sprintf("WP-%05d", lastNumber),
				SystemComponentId:  systemComponentId,
				FailureEventHours:  event.RunningHours,
				N:                  n,
				FreqF:              domain.NewNumber(frequencyValue),
				X:                  domain.NewNumber(xValue),
				Y:                  domain.NewNumber(yValue),
				CreatedAt:          now,
				UpdatedAt:          now,
				CreatedBy:          domain.StringPtr(currentUser),
				UpdatedBy:          domain.StringPtr(currentUser),
			})
		}
		if err := tx.InsertWeibullParameters(ctx, rows); err != nil {
			return err
		}
		if component.ScaleParameter != nil && component.ShapeParameter != nil && component.RunningHours != nil {
			value, err := reliability.WeibullFromFloats(component.RunningHours.Decimal, component.ScaleParameter.Decimal, component.ShapeParameter.Decimal)
			if err != nil {
				return err
			}
			component.DistributionType = domain.StringPtr("Weibull")
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

func (s *ParameterService) UpdateWeibull(ctx context.Context, systemComponentId, currentUser string) (web.Envelope, error) {
	rows, err := s.store.WeibullParametersOfComponent(ctx, systemComponentId)
	if err != nil {
		return web.Envelope{}, err
	}
	if len(rows) == 0 {
		return web.BadRequest("No Weibull Parameters found for the specified SystemComponentId"), nil
	}
	if len(rows) < 2 {
		return web.BadRequest("Weibull calculation requires at least 2 failure events. Please add more failure event data."), nil
	}
	component, err := s.store.FindComponent(ctx, systemComponentId)
	if err != nil {
		return web.Envelope{}, err
	}
	xValues := make([]float64, 0, len(rows))
	yValues := make([]float64, 0, len(rows))
	for _, row := range rows {
		xValues = append(xValues, reliability.ToFloat(row.X.Decimal))
		yValues = append(yValues, reliability.ToFloat(row.Y.Decimal))
	}
	shape := reliability.SlopeWeibull(xValues, yValues)
	intercept := reliability.InterceptWeibull(xValues, yValues)
	scale := math.Exp(-intercept / shape)
	fit, err := reliability.RSquared(yValues, xValues)
	if err != nil {
		return web.Envelope{}, domain.Argument(err.Error())
	}
	gamma := math.Gamma(1 + (1 / shape))
	mtbf := scale * gamma
	runningHours := 0.0
	if component != nil && component.RunningHours != nil {
		runningHours = reliability.ToFloat(component.RunningHours.Decimal)
	}
	failureRate := 0.0
	if scale > 0 && runningHours > 0 {
		failureRate = (shape / scale) * math.Pow(runningHours/scale, shape-1)
	}
	reliabilityValue := math.Exp(-math.Pow(runningHours/scale, shape))
	values := []float64{shape, scale, fit, mtbf, failureRate, reliabilityValue}
	converted := make([]domain.Number, 0, len(values))
	for _, value := range values {
		number, err := reliability.FromFloat(reliability.Finite(value))
		if err != nil {
			return web.Envelope{}, domain.Argument(err.Error())
		}
		converted = append(converted, domain.NewNumber(number))
	}
	if component == nil {
		return web.NotFound("SystemComponentProperties not found for the specified SystemComponentId"), nil
	}
	component.Active = domain.IntPtr(1)
	component.DistributionType = domain.StringPtr("Weibull")
	component.ShapeParameter = &converted[0]
	component.ScaleParameter = &converted[1]
	component.Regresi = &converted[2]
	component.Mtbf = &converted[3]
	component.FailureRate = &converted[4]
	component.ReliabilityValue = &converted[5]
	component.UpdatedBy = domain.StringPtr(currentUser)
	now := domain.Now()
	component.UpdatedAt = &now
	if err := s.store.UpdateComponent(ctx, *component); err != nil {
		return web.Envelope{}, err
	}
	return web.Success(calculationView(*component, true, true), "Data updated successfully"), nil
}
