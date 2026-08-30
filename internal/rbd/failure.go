package rbd

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/shopspring/decimal"

	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/reliability"
	"github.com/Galang17061/strata-api/internal/web"
)

type FailureService struct {
	store *Store
}

func NewFailureService(store *Store) *FailureService {
	return &FailureService{store: store}
}

func (s *FailureService) List(ctx context.Context, systemComponentId, search string) ([]domain.FailureEventHistory, error) {
	return s.store.FailureEventsOfComponent(ctx, systemComponentId, search)
}

func (s *FailureService) Find(ctx context.Context, failureEventId string) (*domain.FailureEventHistory, error) {
	return s.store.FindFailureEvent(ctx, failureEventId)
}

func (s *FailureService) Add(ctx context.Context, inputs []domain.FailureEventCreate, currentUser string) error {
	return s.store.Transact(ctx, func(tx *Store) error {
		lastId, err := tx.LastFailureEventId(ctx)
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
		numbers := map[string]int{}
		rows := []domain.FailureEventHistory{}
		for _, input := range inputs {
			componentId := domain.Deref(input.SystemComponentId)
			if _, known := numbers[componentId]; !known {
				last, err := tx.LastFailureNumber(ctx, componentId)
				if err != nil {
					return err
				}
				numbers[componentId] = domain.DerefInt(last, 0)
			}
			numbers[componentId]++
			lastNumber++
			now := domain.Now()
			failureDate := domain.DateTime{}
			if input.FailureDate != nil {
				failureDate = *input.FailureDate
			}
			rows = append(rows, domain.FailureEventHistory{
				FailureEventId:    fmt.Sprintf("FE-%05d", lastNumber),
				SystemComponentId: componentId,
				FailureDate:       failureDate,
				FailureNumber:     domain.IntPtr(numbers[componentId]),
				RunningHours:      input.RunningHours,
				CreatedAt:         now,
				UpdatedAt:         now,
				CreatedBy:         domain.StringPtr(currentUser),
				UpdatedBy:         domain.StringPtr(currentUser),
			})
		}
		if err := tx.InsertFailureEvents(ctx, rows); err != nil {
			return err
		}
		for componentId := range numbers {
			if err := tx.TouchSystemOfComponent(ctx, componentId); err != nil {
				return err
			}
		}
		return nil
	})
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *FailureService) Delete(ctx context.Context, failureEventId string) error {
	return s.store.Transact(ctx, func(tx *Store) error {
		event, err := tx.FindFailureEvent(ctx, failureEventId)
		if err != nil || event == nil {
			return err
		}
		if err := tx.DeleteFailureEvent(ctx, failureEventId); err != nil {
			return err
		}
		if event.FailureNumber == nil {
			return nil
		}
		remaining, err := tx.FailureEventsAfterNumber(ctx, event.SystemComponentId, *event.FailureNumber)
		if err != nil {
			return err
		}
		for _, later := range remaining {
			if err := tx.UpdateFailureNumber(ctx, later.FailureEventId, domain.DerefInt(later.FailureNumber, 0)-1); err != nil {
				return err
			}
		}
		return tx.TouchSystemOfComponent(ctx, event.SystemComponentId)
	})
}

func (s *FailureService) DeleteAllOfComponent(ctx context.Context, systemComponentId string) error {
	if err := s.store.DeleteFailureEventsOfComponent(ctx, systemComponentId); err != nil {
		return err
	}
	return s.store.TouchSystemOfComponent(ctx, systemComponentId)
}

type ParameterService struct {
	store *Store
}

func NewParameterService(store *Store) *ParameterService {
	return &ParameterService{store: store}
}

func calculationView(c domain.SystemComponentProperties, includeRunningHours, includeRegresi bool) domain.CalculationUpdate {
	view := domain.CalculationUpdate{
		DistributionType: c.DistributionType,
		FailureRate:      c.FailureRate,
		ScaleParameter:   c.ScaleParameter,
		ShapeParameter:   c.ShapeParameter,
		ReliabilityValue: c.ReliabilityValue,
		Mtbf:             c.Mtbf,
	}
	if includeRunningHours {
		view.RunningHours = c.RunningHours
	}
	if includeRegresi {
		view.Regresi = c.Regresi
	}
	return view
}

func (s *ParameterService) UpdateExponential(ctx context.Context, systemComponentId, currentUser string) (web.Envelope, error) {
	parameters, err := s.store.ExponentialParametersOfComponent(ctx, systemComponentId)
	if err != nil {
		return web.Envelope{}, err
	}
	component, err := s.store.FindComponent(ctx, systemComponentId)
	if err != nil {
		return web.Envelope{}, err
	}
	if len(parameters) == 0 {
		if component == nil {
			return web.Envelope{}, domain.Argument("Object reference not set to an instance of an object.")
		}
		if component.FailureRate == nil {
			return web.BadRequest("Cannot Calculated, Please input Failure Rate first"), nil
		}
		mtbf, err := reliability.Div(decimal.NewFromInt(1), component.FailureRate.Decimal)
		if err != nil {
			return web.Envelope{}, err
		}
		component.Mtbf = domain.NumberPtr(domain.NewNumber(mtbf))
		component.DistributionType = domain.StringPtr("Exponential")
		if component.RunningHours != nil {
			component.ReliabilityValue = domain.NumberPtr(domain.NewNumber(reliability.ExponentialFromDecimals(component.FailureRate.Decimal, component.RunningHours.Decimal)))
		}
		component.UpdatedBy = domain.StringPtr(currentUser)
		now := domain.Now()
		component.UpdatedAt = &now
		if err := s.store.UpdateComponent(ctx, *component); err != nil {
			return web.Envelope{}, err
		}
		return web.Success(calculationView(*component, false, false), "Data updated successfully"), nil
	}
	hours := make([]float64, 0, len(parameters))
	lnRt := make([]float64, 0, len(parameters))
	for _, parameter := range parameters {
		hours = append(hours, float64(parameter.FailureEventHours))
		lnRt = append(lnRt, math.Log(reliability.ToFloat(parameter.RT.Decimal)))
	}
	slope, err := reliability.SlopeExponential(lnRt, hours)
	if err != nil {
		return web.Envelope{}, domain.Argument(err.Error())
	}
	failureRate, err := reliability.FromFloat(math.Abs(slope))
	if err != nil {
		return web.Envelope{}, domain.Argument(err.Error())
	}
	fit, err := reliability.RSquared(lnRt, hours)
	if err != nil {
		return web.Envelope{}, domain.Argument(err.Error())
	}
	regresi, err := reliability.FromFloat(fit)
	if err != nil {
		return web.Envelope{}, domain.Argument(err.Error())
	}
	mtbf, err := reliability.Div(decimal.NewFromInt(1), failureRate)
	if err != nil {
		return web.Envelope{}, domain.Argument(err.Error())
	}
	if component == nil {
		return web.NotFound("SystemComponentProperties not found for the specified SystemComponentId"), nil
	}
	component.DistributionType = domain.StringPtr("Exponential")
	component.FailureRate = domain.NumberPtr(domain.NewNumber(failureRate))
	component.Mtbf = domain.NumberPtr(domain.NewNumber(mtbf))
	component.Regresi = domain.NumberPtr(domain.NewNumber(regresi))
	if component.RunningHours != nil {
		component.ReliabilityValue = domain.NumberPtr(domain.NewNumber(reliability.ExponentialFromDecimals(failureRate, component.RunningHours.Decimal)))
	}
	component.UpdatedBy = domain.StringPtr(currentUser)
	now := domain.Now()
	component.UpdatedAt = &now
	if err := s.store.UpdateComponent(ctx, *component); err != nil {
		return web.Envelope{}, err
	}
	return web.Success(calculationView(*component, false, true), "Data updated successfully"), nil
}

func (s *ParameterService) SeedExponential(ctx context.Context, systemComponentId, currentUser string) (web.Envelope, error) {
	component, err := s.store.FindComponent(ctx, systemComponentId)
	if err != nil {
		return web.Envelope{}, err
	}
	if component == nil {
		return web.NotFound("System Component with ID '" + systemComponentId + "' not found"), nil
	}
	master, err := s.store.MasterComponentByNameAndVendor(ctx, component.ComponentName, component.Vendor)
	if err != nil {
		return web.Envelope{}, err
	}
	if master == nil {
		vendorInfo := ""
		if component.Vendor != nil {
			vendorInfo = " from vendor '" + *component.Vendor + "'"
		}
		return web.BadRequest("Master Component '" + component.ComponentName + "'" + vendorInfo + " not found. Please ensure the component exists in master data."), nil
	}
	if master.FailureRate == nil {
		return web.BadRequest("Master Component '" + component.ComponentName + "' does not have Failure Rate defined. Please update the master data first."), nil
	}
	hours := domain.NumberFromInt(defaultRunning)
	if component.RunningHours != nil {
		hours = *component.RunningHours
	}
	component.DistributionType = domain.StringPtr("Exponential")
	component.FailureRate = master.FailureRate
	component.ReliabilityValue = domain.NumberPtr(domain.NewNumber(reliability.ExponentialFromDecimals(master.FailureRate.Decimal, hours.Decimal)))
	component.UpdatedBy = domain.StringPtr(currentUser)
	now := domain.Now()
	component.UpdatedAt = &now
	if err := s.store.UpdateComponent(ctx, *component); err != nil {
		return web.Envelope{}, err
	}
	return web.Created(component, "Data created successfully"), nil
}
