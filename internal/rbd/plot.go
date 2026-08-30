package rbd

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/reliability"
)

type PlotService struct {
	store  *Store
	totals *TotalService
}

func NewPlotService(store *Store, totals *TotalService) *PlotService {
	return &PlotService{store: store, totals: totals}
}

func PlotTimes(runningHours decimal.Decimal) []int {
	step, _ := reliability.Div(runningHours, decimal.NewFromInt(19))
	times := make([]int, 0, 20)
	for i := 0; i < 20; i++ {
		times = append(times, int(reliability.Mul(decimal.NewFromInt(int64(i)), step).RoundBank(0).IntPart()))
	}
	return times
}

func plotReliability(component domain.SystemComponentProperties, hours decimal.Decimal) (decimal.Decimal, error) {
	if domain.Deref(component.DistributionType) == "Exponential" {
		if component.FailureRate == nil {
			return decimal.Zero, errors.New("Nullable object must have a value.")
		}
		return reliability.ExponentialFromDecimals(component.FailureRate.Decimal, hours), nil
	}
	if component.ScaleParameter == nil || component.ShapeParameter == nil {
		return decimal.Zero, errors.New("Nullable object must have a value.")
	}
	return reliability.WeibullFromRatio(hours, component.ScaleParameter.Decimal, component.ShapeParameter.Decimal)
}

func (s *PlotService) UpdateRunningHours(ctx context.Context, rbdSystemId string, runningHours decimal.Decimal, currentUser string) error {
	if runningHours.Sign() < 0 {
		return domain.Argument("Running hours cannot be negative")
	}
	level, err := s.totals.HierarchyLevel(ctx, rbdSystemId)
	if err != nil {
		return err
	}
	if level != "COMPONENT" {
		return domain.InvalidOperation("Running hours can only be updated for component-level RBDs. This RBD represents " + level + ". Update the source component RBDs instead.")
	}
	times := PlotTimes(runningHours)
	components, err := s.store.ComponentsOfSystem(ctx, rbdSystemId)
	if err != nil {
		return err
	}
	if len(components) == 0 {
		return domain.KeyNotFound("No components found for RBD System " + rbdSystemId)
	}
	for _, component := range components {
		if err := s.replot(ctx, component, runningHours, times, currentUser); err != nil {
			return err
		}
	}
	return s.store.TouchSystem(ctx, rbdSystemId)
}

func (s *PlotService) replot(ctx context.Context, component domain.SystemComponentProperties, runningHours decimal.Decimal, times []int, currentUser string) error {
	return s.store.Transact(ctx, func(tx *Store) error {
		component.RunningHours = domain.NumberPtr(domain.NewNumber(runningHours))
		value, err := plotReliability(component, runningHours)
		if err != nil {
			return err
		}
		component.ReliabilityValue = domain.NumberPtr(domain.NewNumber(value))
		component.UpdatedBy = domain.StringPtr(currentUser)
		now := domain.Now()
		component.UpdatedAt = &now
		lastId, err := tx.LastPlotId(ctx)
		if err != nil {
			return err
		}
		if err := tx.DeletePlotsOfComponent(ctx, component.SystemComponentId); err != nil {
			return err
		}
		counter := 1
		if lastId != nil && strings.HasPrefix(*lastId, "RP-") {
			parsed, err := strconv.Atoi((*lastId)[3:])
			if err != nil {
				return domain.Argument("Input string was not in a correct format.")
			}
			counter = parsed + 1
		}
		rows := make([]domain.ReliabilityPlotComponent, 0, len(times))
		for _, timeT := range times {
			point, err := plotReliability(component, decimal.NewFromInt(int64(timeT)))
			if err != nil {
				return err
			}
			stamp := domain.Now()
			rows = append(rows, domain.ReliabilityPlotComponent{
				ReliabilityPlotId: fmt.Sprintf("RP-%05d", counter),
				SystemComponentId: component.SystemComponentId,
				TimeT:             timeT,
				ReliabilityComp:   domain.NumberPtr(domain.NewNumber(point)),
				CreatedAt:         &stamp,
				UpdatedAt:         &stamp,
				CreatedBy:         domain.StringPtr(currentUser),
				UpdatedBy:         domain.StringPtr(currentUser),
			})
			counter++
		}
		if err := tx.InsertPlots(ctx, rows); err != nil {
			return err
		}
		return tx.UpdateComponent(ctx, component)
	})
}

func (s *PlotService) SystemPlot(ctx context.Context, rbdSystemId string) ([]domain.PlotPoint, error) {
	system, err := s.store.FindSystem(ctx, rbdSystemId)
	if err != nil {
		return nil, err
	}
	if system == nil {
		return nil, errors.New("Object reference not set to an instance of an object.")
	}
	rows, err := s.store.PlotRowsOfSystem(ctx, rbdSystemId)
	if err != nil {
		return nil, err
	}
	groups := domain.NewOrderedMap()
	order := []int{}
	for _, row := range rows {
		key := strconv.Itoa(row.TimeT)
		if !groups.Has(key) {
			order = append(order, row.TimeT)
			groups.Set(key, []plotRow{})
		}
		existing, _ := groups.Get(key)
		groups.Set(key, append(existing.([]plotRow), row))
	}
	points := []domain.PlotPoint{}
	for _, timeT := range order {
		grouped, _ := groups.Get(strconv.Itoa(timeT))
		members := grouped.([]plotRow)
		values := newLookup()
		for _, member := range members {
			code := domain.Deref(member.FormulaCode)
			if _, exists := values.values[code]; exists {
				continue
			}
			var value *decimal.Decimal
			if member.ReliabilityComp != nil {
				copyOf := member.ReliabilityComp.Decimal
				value = &copyOf
			}
			values.set(code, value)
		}
		formula := domain.Deref(system.Formula)
		var total *domain.Number
		if formula != "" && len(values.codes) > 0 {
			result, err := values.evaluate(formula, true)
			if err != nil {
				return nil, err
			}
			total = domain.NumberPtr(domain.NewNumber(result))
		} else {
			result, err := reliability.Evaluate(reliability.PrepareFormula(formula, true))
			if err != nil {
				return nil, errors.New("Failed to evaluate the formula: " + formula + ". Original error: " + err.Error())
			}
			total = domain.NumberPtr(domain.NewNumber(result))
		}
		components := domain.NewOrderedMap()
		for _, member := range members {
			if components.Has(member.SystemComponentId) {
				return nil, errors.New("An item with the same key has already been added. Key: " + member.SystemComponentId)
			}
			components.Set(member.SystemComponentId, domain.PlotComponentValue{CompName: domain.StringPtr(member.ComponentName), RComp: member.ReliabilityComp})
		}
		components.Set("TOTAL", domain.PlotComponentValue{CompName: domain.StringPtr("TOTAL RELIABILITY"), RComp: total})
		points = append(points, domain.PlotPoint{DrawingName: system.DrawingName, Time: timeT, Components: components})
	}
	return points, nil
}

func (s *PlotService) All(ctx context.Context) ([]domain.ReliabilityPlotComponent, error) {
	return s.store.AllPlots(ctx)
}

func (s *PlotService) Find(ctx context.Context, reliabilityPlotId string) (*domain.ReliabilityPlotComponent, error) {
	return s.store.FindPlot(ctx, reliabilityPlotId)
}

func (s *PlotService) Add(ctx context.Context, request domain.PlotCreate, currentUser string) error {
	now := domain.Now()
	value := request.ReliabilityComp
	return s.store.InsertPlots(ctx, []domain.ReliabilityPlotComponent{{
		ReliabilityPlotId: domain.Deref(request.ReliabilityPlotId),
		SystemComponentId: domain.Deref(request.SystemComponentId),
		TimeT:             request.TimeT,
		ReliabilityComp:   &value,
		CreatedAt:         &now,
		UpdatedAt:         &now,
		CreatedBy:         domain.StringPtr(currentUser),
		UpdatedBy:         domain.StringPtr(currentUser),
	}})
}

func (s *PlotService) Update(ctx context.Context, reliabilityPlotId string, request domain.PlotUpdate, currentUser string) error {
	existing, err := s.store.FindPlot(ctx, reliabilityPlotId)
	if err != nil || existing == nil {
		return err
	}
	existing.SystemComponentId = domain.Deref(request.SystemComponentId)
	existing.TimeT = request.TimeT
	existing.ReliabilityComp = domain.NumberPtr(domain.NumberFromInt(int64(request.ReliabilityComp)))
	now := domain.Now()
	existing.UpdatedAt = &now
	existing.UpdatedBy = domain.StringPtr(currentUser)
	return s.store.UpdatePlot(ctx, *existing)
}

func (s *PlotService) Delete(ctx context.Context, reliabilityPlotId string) error {
	existing, err := s.store.FindPlot(ctx, reliabilityPlotId)
	if err != nil || existing == nil {
		return err
	}
	return s.store.DeletePlot(ctx, reliabilityPlotId)
}
