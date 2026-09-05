package rbd

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/optimize"
	"github.com/Galang17061/strata-api/internal/reliability"
)

type OptimizationService struct {
	store *Store
	gate  *Gate
}

func NewOptimizationService(store *Store) *OptimizationService {
	return &OptimizationService{store: store}
}

func (s *OptimizationService) WithGate(gate *Gate) *OptimizationService {
	s.gate = gate
	return s
}

type builtProblem struct {
	problem *optimize.Problem
	slots   []domain.OptimizationSlotView
	fixed   []domain.OptimizationFixedSlotView
	system  *domain.RbdSystemDrawing
}

func parseMoney(text *string) (float64, bool) {
	if text == nil {
		return 0, false
	}
	cleaned := strings.NewReplacer(",", "", " ", "", "Rp", "", "rp", "").Replace(strings.TrimSpace(*text))
	if cleaned == "" {
		return 0, false
	}
	value, err := strconv.ParseFloat(cleaned, 64)
	if err != nil || value < 0 {
		return 0, false
	}
	return value, true
}

func compatibilityIds(text *string) []string {
	if text == nil {
		return nil
	}
	parts := strings.Split(*text, ",")
	ids := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			ids = append(ids, trimmed)
		}
	}
	return ids
}

func containsFold(list []string, target string) bool {
	for _, item := range list {
		if strings.EqualFold(item, target) {
			return true
		}
	}
	return false
}

func compileCleaned(formula string) (*optimize.CompiledFormula, error) {
	return optimize.Compile(reliability.PrepareFormula(reliability.StripVirtualCodes(formula), true))
}

func (s *OptimizationService) buildProblem(ctx context.Context, rbdSystemId string, runningHours *float64, locks []domain.OptimizationLock) (*builtProblem, error) {
	system, err := s.store.FindSystem(ctx, rbdSystemId)
	if err != nil {
		return nil, err
	}
	if system == nil {
		return nil, domain.KeyNotFound("RBD System " + rbdSystemId + " not found")
	}
	hierarchies, err := s.store.HierarchiesOfSystemByLevel(ctx, rbdSystemId, false)
	if err != nil {
		return nil, err
	}
	components, err := s.store.ComponentsOfSystem(ctx, rbdSystemId)
	if err != nil {
		return nil, err
	}
	lockedComponent := map[string]string{}
	for _, lock := range locks {
		if lock.SystemComponentId != nil && lock.ComponentId != nil {
			lockedComponent[*lock.SystemComponentId] = *lock.ComponentId
		}
	}
	problem := &optimize.Problem{FixedValues: map[string]float64{}}
	built := &builtProblem{problem: problem, system: system}
	masterCache := map[string][]domain.MasterComponentWithVendor{}
	for _, component := range components {
		code := domain.Deref(component.FormulaCode)
		if code == "" || reliability.IsVirtualCode(code) || strings.Contains(component.ComponentName, "Virtual Node") {
			continue
		}
		if domain.DerefInt(component.Active, 1) != 1 {
			continue
		}
		units := domain.DerefInt(component.TotalComponent, 1)
		active := domain.DerefInt(component.ActiveComponent, 1)
		connection := domain.Deref(component.ConnectionType)
		storedBase := 0.0
		if component.ReliabilityValue != nil {
			storedBase = reliability.ToFloat(component.ReliabilityValue.Decimal)
		}
		storedValue := optimize.AdjustReliability(storedBase, connection, active, units)
		fix := func(reason string, cost float64) {
			problem.FixedValues[code] = storedValue
			problem.FixedCost += cost
			built.fixed = append(built.fixed, domain.OptimizationFixedSlotView{
				SystemComponentId: component.SystemComponentId,
				ComponentName:     component.ComponentName,
				VendorName:        component.Vendor,
				Reason:            reason,
				Reliability:       storedValue,
				Cost:              cost,
			})
		}
		rows, cached := masterCache[component.ComponentName]
		if !cached {
			rows, err = s.store.MasterComponentsByName(ctx, component.ComponentName)
			if err != nil {
				return nil, err
			}
			masterCache[component.ComponentName] = rows
		}
		var anchor *domain.MasterComponentWithVendor
		for index := range rows {
			if strings.EqualFold(domain.Deref(rows[index].ManufacturerName), domain.Deref(component.Vendor)) {
				anchor = &rows[index]
				break
			}
		}
		if anchor == nil {
			fix("Not in the master data under this vendor", 0)
			continue
		}
		anchorCost, anchorHasCost := parseMoney(anchor.Cost)
		if !anchorHasCost {
			fix("No numeric cost in the master data", 0)
			continue
		}
		if anchor.FailureRate == nil {
			fix("No failure rate in the master data", anchorCost*float64(units))
			continue
		}
		hours := 0.0
		if runningHours != nil && *runningHours > 0 {
			hours = *runningHours
		} else if component.RunningHours != nil {
			hours = reliability.ToFloat(component.RunningHours.Decimal)
		}
		if hours <= 0 {
			fix("No running hours to score against", anchorCost*float64(units))
			continue
		}
		anchorCompat := compatibilityIds(anchor.Compatibility)
		candidates := []optimize.Candidate{}
		candidateViews := []domain.OptimizationCandidateView{}
		currentIndex := -1
		for index := range rows {
			row := &rows[index]
			allowed := row.ComponentId == anchor.ComponentId ||
				len(anchorCompat) == 0 ||
				containsFold(anchorCompat, row.ComponentId) ||
				containsFold(compatibilityIds(row.Compatibility), anchor.ComponentId)
			if !allowed || row.FailureRate == nil {
				continue
			}
			unitCost, hasCost := parseMoney(row.Cost)
			if !hasCost {
				continue
			}
			rate := reliability.ToFloat(row.FailureRate.Decimal)
			candidate := optimize.Candidate{
				ComponentId: row.ComponentId,
				VendorId:    row.VendorId,
				VendorName:  domain.Deref(row.ManufacturerName),
				FailureRate: rate,
				UnitCost:    unitCost,
				Reliability: optimize.AdjustReliability(optimize.ExponentialFloat(rate, hours), connection, active, units),
			}
			if row.ComponentId == anchor.ComponentId {
				currentIndex = len(candidates)
			}
			candidates = append(candidates, candidate)
			candidateViews = append(candidateViews, domain.OptimizationCandidateView{
				ComponentId: candidate.ComponentId,
				VendorId:    candidate.VendorId,
				VendorName:  candidate.VendorName,
				FailureRate: candidate.FailureRate,
				UnitCost:    candidate.UnitCost,
				Reliability: candidate.Reliability,
			})
		}
		if len(candidates) < 2 || currentIndex < 0 {
			fix("Only one compatible vendor offers this part", anchorCost*float64(units))
			continue
		}
		lockedIndex := -1
		if wanted, isLocked := lockedComponent[component.SystemComponentId]; isLocked {
			for index, candidate := range candidates {
				if candidate.ComponentId == wanted {
					lockedIndex = index
					break
				}
			}
		}
		problem.Slots = append(problem.Slots, optimize.Slot{
			SystemComponentId: component.SystemComponentId,
			ComponentName:     component.ComponentName,
			FormulaCode:       code,
			Units:             units,
			CurrentIndex:      currentIndex,
			LockedIndex:       lockedIndex,
			Candidates:        candidates,
		})
		built.slots = append(built.slots, domain.OptimizationSlotView{
			SystemComponentId: component.SystemComponentId,
			ComponentName:     component.ComponentName,
			FormulaCode:       code,
			Units:             units,
			ConnectionType:    component.ConnectionType,
			CurrentIndex:      currentIndex,
			ProposedIndex:     currentIndex,
			Locked:            lockedIndex >= 0,
			Candidates:        candidateViews,
		})
	}
	layerCandidates := make([]domain.Hierarchy, 0, len(hierarchies))
	layerCandidates = append(layerCandidates, hierarchies...)
	sort.SliceStable(layerCandidates, func(a, b int) bool { return layerCandidates[a].Level > layerCandidates[b].Level })
	rootCodes := []string{}
	for _, hierarchy := range hierarchies {
		if hierarchy.Level == 1 {
			if code := domain.Deref(hierarchy.FormulaCode); code != "" {
				rootCodes = append(rootCodes, code)
			}
		}
	}
	for _, hierarchy := range layerCandidates {
		code := domain.Deref(hierarchy.FormulaCode)
		if code == "" || reliability.IsVirtualCode(code) {
			continue
		}
		formula := domain.Deref(hierarchy.Formula)
		if formula == "" {
			if hierarchy.RealibilityValue != nil {
				problem.FixedValues[code] = reliability.ToFloat(hierarchy.RealibilityValue.Decimal)
			}
			continue
		}
		compiled, err := compileCleaned(formula)
		if err != nil {
			if hierarchy.RealibilityValue != nil {
				problem.FixedValues[code] = reliability.ToFloat(hierarchy.RealibilityValue.Decimal)
			}
			continue
		}
		problem.Layers = append(problem.Layers, optimize.Layer{Code: code, Formula: compiled})
	}
	totalFormula := domain.Deref(system.Formula)
	if strings.TrimSpace(reliability.StripVirtualCodes(totalFormula)) == "" {
		if len(rootCodes) > 0 {
			totalFormula = strings.Join(rootCodes, "*")
		} else {
			componentCodes := []string{}
			for _, slot := range problem.Slots {
				componentCodes = append(componentCodes, slot.FormulaCode)
			}
			for code := range problem.FixedValues {
				componentCodes = append(componentCodes, code)
			}
			sort.Strings(componentCodes)
			totalFormula = strings.Join(componentCodes, "*")
		}
	}
	if strings.TrimSpace(totalFormula) != "" {
		compiled, err := compileCleaned(totalFormula)
		if err == nil {
			problem.Total = compiled
		}
	}
	provided := map[string]bool{}
	for _, slot := range problem.Slots {
		provided[slot.FormulaCode] = true
	}
	for code := range problem.FixedValues {
		provided[code] = true
	}
	for _, layer := range problem.Layers {
		provided[layer.Code] = true
	}
	needed := []string{}
	if problem.Total != nil {
		needed = append(needed, problem.Total.Names...)
	}
	for _, layer := range problem.Layers {
		needed = append(needed, layer.Formula.Names...)
	}
	for _, name := range needed {
		if !provided[name] {
			problem.FixedValues[name] = 1
			provided[name] = true
		}
	}
	return built, nil
}

const (
	defaultPopulation = 700
	defaultGeneration = 1500
	defaultCrossover  = 0.9
	defaultMutation   = 0.4
	maxPopulation     = 2000
	maxGeneration     = 5000
)

func clampInt(value *int, fallback, ceiling int) int {
	if value == nil || *value <= 0 {
		return fallback
	}
	if *value > ceiling {
		return ceiling
	}
	return *value
}

func clampProbability(value *float64, fallback float64) float64 {
	if value == nil || *value <= 0 || *value > 1 {
		return fallback
	}
	return *value
}

func (s *OptimizationService) objectiveFor(request domain.OptimizationPreviewRequest) (optimize.Objective, error) {
	mode := 0
	if request.Mode != nil {
		mode = *request.Mode
	}
	objective := optimize.Objective{Mode: optimize.Mode(mode)}
	switch objective.Mode {
	case optimize.ModeReliability:
		return objective, nil
	case optimize.ModeBudget:
		if request.MaxBudget == nil || *request.MaxBudget <= 0 {
			return objective, domain.Argument("MaxBudget is required for the budget-bounded mode")
		}
		objective.MaxBudget = *request.MaxBudget
		return objective, nil
	case optimize.ModeBudgetFloor:
		if request.MaxBudget == nil || *request.MaxBudget <= 0 {
			return objective, domain.Argument("MaxBudget is required for the budget-and-floor mode")
		}
		if request.TargetReliability == nil || *request.TargetReliability <= 0 || *request.TargetReliability > 1 {
			return objective, domain.Argument("TargetReliability between 0 and 1 is required for the budget-and-floor mode")
		}
		objective.MaxBudget = *request.MaxBudget
		objective.TargetReliability = *request.TargetReliability
		objective.WeightCost = 0.9
		objective.WeightReliability = 0.1
		if request.WeightCost != nil && *request.WeightCost > 0 && *request.WeightCost < 1 {
			objective.WeightCost = *request.WeightCost
			objective.WeightReliability = 1 - *request.WeightCost
		}
		if request.WeightReliability != nil && *request.WeightReliability > 0 && *request.WeightReliability < 1 {
			objective.WeightReliability = *request.WeightReliability
		}
		return objective, nil
	default:
		return objective, domain.Argument("Mode must be 1 (reliability), 2 (budget bound) or 3 (budget and floor)")
	}
}

func (s *OptimizationService) Preview(ctx context.Context, request domain.OptimizationPreviewRequest) (*domain.OptimizationPreviewResult, error) {
	rbdSystemId := strings.TrimSpace(domain.Deref(request.RbdSystemId))
	if rbdSystemId == "" {
		return nil, domain.Argument("RbdSystemId is required")
	}
	objective, err := s.objectiveFor(request)
	if err != nil {
		return nil, err
	}
	leave, err := s.gate.Enter(ctx)
	if err != nil {
		return nil, err
	}
	defer leave()
	built, err := s.buildProblem(ctx, rbdSystemId, request.RunningHours, request.Locks)
	if err != nil {
		return nil, err
	}
	if err := built.problem.Validate(); err != nil {
		return nil, domain.InvalidOperation(err.Error())
	}
	seed := time.Now().UnixNano()
	if request.Seed != nil && *request.Seed != 0 {
		seed = *request.Seed
	}
	options := optimize.Options{
		PopulationSize:       clampInt(request.PopulationSize, defaultPopulation, maxPopulation),
		MaxGenerations:       clampInt(request.MaxGenerations, defaultGeneration, maxGeneration),
		CrossoverProbability: clampProbability(request.CrossoverProbability, defaultCrossover),
		MutationProbability:  clampProbability(request.MutationProbability, defaultMutation),
		Seed:                 seed,
	}
	started := time.Now()
	result, err := optimize.Run(built.problem.Ranges(), built.problem.LockedIndexes(), built.problem.Evaluate, objective, options)
	if err != nil {
		return nil, domain.InvalidOperation(err.Error())
	}
	baselineReliability, baselineCost := built.problem.Evaluate(built.problem.CurrentGenome())
	for index := range built.slots {
		built.slots[index].ProposedIndex = result.Best[index]
	}
	return &domain.OptimizationPreviewResult{
		RbdSystemId: rbdSystemId,
		SystemName:  built.system.SystemName,
		Mode:        int(objective.Mode),
		Slots:       built.slots,
		FixedSlots:  built.fixed,
		Totals: domain.OptimizationTotals{
			Reliability:         result.BestReliability,
			Cost:                result.BestCost,
			BaselineReliability: baselineReliability,
			BaselineCost:        baselineCost,
		},
		Feasible:    result.Feasible,
		History:     result.History,
		Generations: result.Generations,
		Seed:        result.Seed,
		ExecutionMs: time.Since(started).Milliseconds(),
	}, nil
}

func (s *OptimizationService) Score(ctx context.Context, request domain.OptimizationScoreRequest) (*domain.OptimizationScoreResult, error) {
	rbdSystemId := strings.TrimSpace(domain.Deref(request.RbdSystemId))
	if rbdSystemId == "" {
		return nil, domain.Argument("RbdSystemId is required")
	}
	built, err := s.buildProblem(ctx, rbdSystemId, request.RunningHours, nil)
	if err != nil {
		return nil, err
	}
	if err := built.problem.Validate(); err != nil {
		return nil, domain.InvalidOperation(err.Error())
	}
	genome := built.problem.CurrentGenome()
	chosen := map[string]string{}
	for _, choice := range request.Choices {
		if choice.SystemComponentId != nil && choice.ComponentId != nil {
			chosen[*choice.SystemComponentId] = *choice.ComponentId
		}
	}
	for index, slot := range built.problem.Slots {
		if wanted, ok := chosen[slot.SystemComponentId]; ok {
			for candidateIndex, candidate := range slot.Candidates {
				if candidate.ComponentId == wanted {
					genome[index] = candidateIndex
					break
				}
			}
		}
	}
	reliabilityValue, cost := built.problem.Evaluate(genome)
	baselineReliability, baselineCost := built.problem.Evaluate(built.problem.CurrentGenome())
	return &domain.OptimizationScoreResult{
		Totals: domain.OptimizationTotals{
			Reliability:         reliabilityValue,
			Cost:                cost,
			BaselineReliability: baselineReliability,
			BaselineCost:        baselineCost,
		},
	}, nil
}
