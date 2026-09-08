package rbd

import (
	"context"
	"sort"
	"strconv"
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

type SimulationCulprit struct {
	Code          string  `json:"code"`
	ComponentName string  `json:"componentName"`
	Share         float64 `json:"share"`
}

type SimulationDiagram struct {
	HierarchyId   string                        `json:"hierarchyId"`
	HierarchyName string                        `json:"hierarchyName"`
	Formula       string                        `json:"formula"`
	Components    int                           `json:"components"`
	Survivors     int                           `json:"survivors"`
	Reliability   float64                       `json:"reliability"`
	LowerBound    float64                       `json:"lowerBound"`
	UpperBound    float64                       `json:"upperBound"`
	MeanLife      float64                       `json:"meanLife"`
	MedianLife    float64                       `json:"medianLife"`
	B10Life       float64                       `json:"b10Life"`
	Curve         []reliability.SimulationPoint `json:"curve"`
	Culprits      []SimulationCulprit           `json:"culprits"`
	Warnings      []string                      `json:"warnings"`
}

type SimulationCoverage struct {
	DiagramsInSystem    int      `json:"diagramsInSystem"`
	DiagramsRehearsed   int      `json:"diagramsRehearsed"`
	ComponentsInSystem  int      `json:"componentsInSystem"`
	ComponentsRehearsed int      `json:"componentsRehearsed"`
	ComponentsLeftOut   []string `json:"componentsLeftOut"`
}

type SimulationSummary struct {
	RbdSystemId  string              `json:"rbdSystemId"`
	SystemName   string              `json:"systemName"`
	MissionHours float64             `json:"missionHours"`
	Trials       int                 `json:"trials"`
	Seed         int64               `json:"seed"`
	Diagrams     []SimulationDiagram `json:"diagrams"`
	Coverage     SimulationCoverage  `json:"coverage"`
	Warnings     []string            `json:"warnings"`
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
	chosen, err := chooseDiagrams(hierarchies, request.HierarchyId)
	if err != nil {
		return nil, err
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
	summary := &SimulationSummary{
		RbdSystemId:  request.RbdSystemId,
		SystemName:   domain.Deref(system.SystemName),
		MissionHours: mission,
		Seed:         request.Seed,
	}
	rehearsed := map[string]bool{}
	for _, root := range chosen {
		expand := hierarchyFormulas(hierarchies, root)
		flat, err := reliability.FlattenFormula(strings.TrimSpace(domain.Deref(root.Formula)), expand)
		if err != nil {
			summary.Warnings = append(summary.Warnings, diagramName(root)+" could not be flattened: "+err.Error())
			continue
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
			summary.Warnings = append(summary.Warnings, diagramName(root)+" could not be rehearsed: "+err.Error())
			continue
		}
		summary.Trials = output.Trials
		summary.Seed = output.Seed
		for _, culprit := range output.Culprits {
			rehearsed[culprit.Code] = true
		}
		diagram := SimulationDiagram{
			HierarchyId:   root.HierarchyId,
			HierarchyName: domain.Deref(root.SubSystemName),
			Formula:       flat,
			Components:    output.Blocks,
			Survivors:     output.Survivors,
			Reliability:   output.Reliability,
			LowerBound:    output.LowerBound,
			UpperBound:    output.UpperBound,
			MeanLife:      output.MeanLife,
			MedianLife:    output.MedianLife,
			B10Life:       output.B10Life,
			Curve:         output.Curve,
			Culprits:      namedCulprits(output.Culprits, names),
			Warnings:      diagramWarnings(output, names),
		}
		markRehearsed(rehearsed, flat, parts)
		summary.Diagrams = append(summary.Diagrams, diagram)
	}
	if len(summary.Diagrams) == 0 {
		return nil, domain.InvalidOperation("None of the diagrams in this system could be rehearsed.")
	}
	summary.Coverage = coverage(hierarchies, chosen, components, rehearsed, names)
	if left := len(summary.Coverage.ComponentsLeftOut); left > 0 {
		summary.Warnings = append(summary.Warnings, strconv.Itoa(left)+" component(s) of this system are not wired into any rehearsed diagram, so they had no say in these figures.")
	}
	if summary.Coverage.DiagramsRehearsed < summary.Coverage.DiagramsInSystem {
		summary.Warnings = append(summary.Warnings, "This system holds "+strconv.Itoa(summary.Coverage.DiagramsInSystem)+" diagram(s) but only "+strconv.Itoa(summary.Coverage.DiagramsRehearsed)+" were rehearsed. Each figure below stands for its own diagram, not for the system as a whole.")
	} else if len(summary.Diagrams) > 1 {
		summary.Warnings = append(summary.Warnings, "Each figure below stands for one diagram on its own. Strata does not assume how the diagrams depend on each other, so there is no single number for the whole system.")
	}
	return summary, nil
}

func chooseDiagrams(hierarchies []domain.Hierarchy, wanted string) ([]domain.Hierarchy, error) {
	if wanted != "" {
		for _, hierarchy := range hierarchies {
			if hierarchy.HierarchyId == wanted {
				if strings.TrimSpace(domain.Deref(hierarchy.Formula)) == "" {
					return nil, domain.InvalidOperation("That diagram has no wiring to rehearse yet.")
				}
				return []domain.Hierarchy{hierarchy}, nil
			}
		}
		return nil, domain.KeyNotFound("Hierarchy not found in this system")
	}
	roots := []domain.Hierarchy{}
	lowest := 0
	for _, hierarchy := range hierarchies {
		if strings.TrimSpace(domain.Deref(hierarchy.Formula)) == "" {
			continue
		}
		if len(roots) == 0 || hierarchy.Level < lowest {
			lowest = hierarchy.Level
		}
	}
	for _, hierarchy := range hierarchies {
		if strings.TrimSpace(domain.Deref(hierarchy.Formula)) == "" || hierarchy.Level != lowest {
			continue
		}
		roots = append(roots, hierarchy)
	}
	if len(roots) == 0 {
		return nil, domain.InvalidOperation("This system has no wiring to rehearse yet.")
	}
	sort.Slice(roots, func(a, b int) bool { return roots[a].HierarchyId < roots[b].HierarchyId })
	return roots, nil
}

func hierarchyFormulas(hierarchies []domain.Hierarchy, root domain.Hierarchy) map[string]string {
	expand := map[string]string{}
	rootCode := strings.TrimSpace(domain.Deref(root.FormulaCode))
	for _, hierarchy := range hierarchies {
		code := strings.TrimSpace(domain.Deref(hierarchy.FormulaCode))
		formula := strings.TrimSpace(domain.Deref(hierarchy.Formula))
		if code == "" || formula == "" || code == rootCode {
			continue
		}
		expand[code] = formula
	}
	return expand
}

func diagramName(hierarchy domain.Hierarchy) string {
	name := strings.TrimSpace(domain.Deref(hierarchy.SubSystemName))
	if name == "" {
		return hierarchy.HierarchyId
	}
	return name
}

func diagramWarnings(output reliability.SimulationOutput, names map[string]string) []string {
	warnings := []string{}
	if len(output.Unbacked) > 0 {
		warnings = append(warnings, "The wiring mentions "+strconv.Itoa(len(output.Unbacked))+" name(s) with no component behind them ("+strings.Join(output.Unbacked, ", ")+"); they were treated as always healthy, which flatters the result.")
	}
	if len(output.NeverFails) > 0 {
		warnings = append(warnings, strconv.Itoa(len(output.NeverFails))+" rehearsed part(s) carry no distribution figures ("+strings.Join(componentNames(output.NeverFails, names), ", ")+"), so they never failed in any run.")
	}
	return warnings
}

func componentNames(codes []string, names map[string]string) []string {
	labelled := make([]string, 0, len(codes))
	for _, code := range codes {
		if name := names[code]; name != "" {
			labelled = append(labelled, name)
			continue
		}
		labelled = append(labelled, code)
	}
	return labelled
}

func markRehearsed(rehearsed map[string]bool, formula string, parts []reliability.SimulationPart) {
	structure, err := reliability.CompileStructure(formula)
	if err != nil {
		return
	}
	for _, part := range parts {
		if _, found := structure.Slot(part.Code); found {
			rehearsed[part.Code] = true
		}
	}
}

func coverage(hierarchies, chosen []domain.Hierarchy, components []domain.SystemComponentProperties, rehearsed map[string]bool, names map[string]string) SimulationCoverage {
	diagrams := 0
	for _, hierarchy := range hierarchies {
		if strings.TrimSpace(domain.Deref(hierarchy.Formula)) != "" {
			diagrams++
		}
	}
	report := SimulationCoverage{
		DiagramsInSystem:  diagrams,
		DiagramsRehearsed: len(chosen),
		ComponentsLeftOut: []string{},
	}
	for _, component := range components {
		code := strings.TrimSpace(domain.Deref(component.FormulaCode))
		if code == "" || reliability.IsVirtualCode(code) {
			continue
		}
		report.ComponentsInSystem++
		if rehearsed[code] {
			report.ComponentsRehearsed++
			continue
		}
		report.ComponentsLeftOut = append(report.ComponentsLeftOut, component.ComponentName)
	}
	sort.Strings(report.ComponentsLeftOut)
	return report
}

func simulationParts(components []domain.SystemComponentProperties) ([]reliability.SimulationPart, map[string]string) {
	parts := []reliability.SimulationPart{}
	names := map[string]string{}
	for _, component := range components {
		code := strings.TrimSpace(domain.Deref(component.FormulaCode))
		if code == "" || reliability.IsVirtualCode(code) {
			continue
		}
		parts = append(parts, reliability.SimulationPart{
			Code:            code,
			Distribution:    strings.ToLower(strings.TrimSpace(domain.Deref(component.DistributionType))),
			FailureRate:     numberFloat(component.FailureRate),
			Scale:           numberFloat(component.ScaleParameter),
			Shape:           numberFloat(component.ShapeParameter),
			AllowedFailures: intOrZero(component.AllowedFailures),
			Active:          intOrZero(component.ActiveComponent),
			Total:           intOrZero(component.TotalComponent),
		})
		names[code] = component.ComponentName
	}
	return parts, names
}

func missionFromComponents(components []domain.SystemComponentProperties) float64 {
	longest := 0.0
	for _, component := range components {
		if hours := numberFloat(component.RunningHours); hours > longest {
			longest = hours
		}
	}
	if longest <= 0 {
		return 1000
	}
	return longest
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
