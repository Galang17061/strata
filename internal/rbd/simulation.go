package rbd

import (
	"context"
	"strings"

	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/reliability"
)

type SimulationRequest struct {
	RbdSystemId  string  `json:"rbdSystemId"`
	HierarchyId  string  `json:"hierarchyId"`
	MissionHours float64 `json:"missionHours"`
	Trials       int     `json:"trials"`
	Seed         int64   `json:"seed"`
	CurvePoints  int     `json:"curvePoints"`
}

type SimulationSummary struct {
	RbdSystemId   string                         `json:"rbdSystemId"`
	SystemName    string                         `json:"systemName"`
	HierarchyId   string                         `json:"hierarchyId"`
	HierarchyName string                         `json:"hierarchyName"`
	Formula       string                         `json:"formula"`
	Components    int                            `json:"components"`
	Trials        int                            `json:"trials"`
	MissionHours  float64                        `json:"missionHours"`
	Seed          int64                          `json:"seed"`
	Survivors     int                            `json:"survivors"`
	Reliability   float64                        `json:"reliability"`
	LowerBound    float64                        `json:"lowerBound"`
	UpperBound    float64                        `json:"upperBound"`
	MeanLife      float64                        `json:"meanLife"`
	MedianLife    float64                        `json:"medianLife"`
	B10Life       float64                        `json:"b10Life"`
	Curve         []reliability.SimulationPoint  `json:"curve"`
	Culprits      []SimulationCulprit            `json:"culprits"`
}

type SimulationCulprit struct {
	Code          string  `json:"code"`
	ComponentName string  `json:"componentName"`
	Share         float64 `json:"share"`
}

type SimulationService struct {
	store *Store
}

func NewSimulationService(store *Store) *SimulationService {
	return &SimulationService{store: store}
}

func (s *SimulationService) Run(ctx context.Context, request SimulationRequest) (*SimulationSummary, error) {
	system, err := s.store.FindSystem(ctx, request.RbdSystemId)
	if err != nil {
		return nil, err
	}
	if system == nil {
		return nil, domain.KeyNotFound("System not found")
	}
	hierarchies, err := s.store.HierarchiesOfSystem(ctx, request.RbdSystemId)
	if err != nil {
		return nil, err
	}
	root, err := chooseRoot(hierarchies, request.HierarchyId)
	if err != nil {
		return nil, err
	}
	expand := map[string]string{}
	for _, hierarchy := range hierarchies {
		code := strings.TrimSpace(domain.Deref(hierarchy.FormulaCode))
		formula := strings.TrimSpace(domain.Deref(hierarchy.Formula))
		if code == "" || formula == "" || code == domain.Deref(root.FormulaCode) {
			continue
		}
		expand[code] = formula
	}
	flat, err := reliability.FlattenFormula(strings.TrimSpace(domain.Deref(root.Formula)), expand)
	if err != nil {
		return nil, domain.InvalidOperation(err.Error())
	}
	components, err := s.store.ComponentsOfSystem(ctx, request.RbdSystemId)
	if err != nil {
		return nil, err
	}
	parts, names := simulationParts(components)
	if len(parts) == 0 {
		return nil, domain.InvalidOperation("This system has no components to rehearse yet.")
	}
	mission := request.MissionHours
	if mission <= 0 {
		mission = missionFromComponents(components)
	}
	output, err := reliability.Simulate(reliability.SimulationInput{
		Formula:      flat,
		Parts:        parts,
		MissionHours: mission,
		Trials:       request.Trials,
		Seed:         request.Seed,
		CurvePoints:  request.CurvePoints,
	})
	if err != nil {
		return nil, domain.InvalidOperation(err.Error())
	}
	summary := &SimulationSummary{
		RbdSystemId:   request.RbdSystemId,
		SystemName:    domain.Deref(system.SystemName),
		HierarchyId:   root.HierarchyId,
		HierarchyName: domain.Deref(root.SubSystemName),
		Formula:       flat,
		Components:    output.Blocks,
		Trials:        output.Trials,
		MissionHours:  output.MissionHours,
		Seed:          output.Seed,
		Survivors:     output.Survivors,
		Reliability:   output.Reliability,
		LowerBound:    output.LowerBound,
		UpperBound:    output.UpperBound,
		MeanLife:      output.MeanLife,
		MedianLife:    output.MedianLife,
		B10Life:       output.B10Life,
		Curve:         output.Curve,
		Culprits:      namedCulprits(output.Culprits, names),
	}
	return summary, nil
}

func chooseRoot(hierarchies []domain.Hierarchy, wanted string) (domain.Hierarchy, error) {
	if wanted != "" {
		for _, hierarchy := range hierarchies {
			if hierarchy.HierarchyId == wanted {
				return hierarchy, nil
			}
		}
		return domain.Hierarchy{}, domain.KeyNotFound("Hierarchy not found in this system")
	}
	best := domain.Hierarchy{}
	found := false
	for _, hierarchy := range hierarchies {
		if strings.TrimSpace(domain.Deref(hierarchy.Formula)) == "" {
			continue
		}
		if !found || hierarchy.Level < best.Level {
			best = hierarchy
			found = true
		}
	}
	if !found {
		return domain.Hierarchy{}, domain.InvalidOperation("This system has no wiring to rehearse yet.")
	}
	return best, nil
}

func simulationParts(components []domain.SystemComponentProperties) ([]reliability.SimulationPart, map[string]string) {
	parts := []reliability.SimulationPart{}
	names := map[string]string{}
	for _, component := range components {
		code := strings.TrimSpace(domain.Deref(component.FormulaCode))
		if code == "" || reliability.IsVirtualCode(code) {
			continue
		}
		part := reliability.SimulationPart{
			Code:            code,
			Distribution:    strings.ToLower(strings.TrimSpace(domain.Deref(component.DistributionType))),
			FailureRate:     numberFloat(component.FailureRate),
			Scale:           numberFloat(component.ScaleParameter),
			Shape:           numberFloat(component.ShapeParameter),
			AllowedFailures: intOrZero(component.AllowedFailures),
			Active:          intOrZero(component.ActiveComponent),
			Total:           intOrZero(component.TotalComponent),
		}
		parts = append(parts, part)
		names[code] = component.ComponentName
	}
	return parts, names
}

func missionFromComponents(components []domain.SystemComponentProperties) float64 {
	for _, component := range components {
		hours := numberFloat(component.RunningHours)
		if hours > 0 {
			return hours
		}
	}
	return 1000
}

func namedCulprits(culprits []reliability.SimulationCulprit, names map[string]string) []SimulationCulprit {
	named := make([]SimulationCulprit, 0, len(culprits))
	for _, culprit := range culprits {
		named = append(named, SimulationCulprit{
			Code:          culprit.Code,
			ComponentName: names[culprit.Code],
			Share:         culprit.Share,
		})
	}
	return named
}

func numberFloat(value *domain.Number) float64 {
	if value == nil {
		return 0
	}
	return value.Decimal.InexactFloat64()
}

func intOrZero(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
