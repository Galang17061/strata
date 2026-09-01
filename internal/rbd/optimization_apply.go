package rbd

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/optimize"
	"github.com/Galang17061/strata-api/internal/reliability"
)

type idRemapper struct {
	exact   map[string]string
	ordered [][2]string
}

func newIdRemapper() *idRemapper {
	return &idRemapper{exact: map[string]string{}}
}

func (m *idRemapper) add(oldId, newId string) {
	if oldId == "" || oldId == newId {
		return
	}
	if _, exists := m.exact[oldId]; exists {
		return
	}
	m.exact[oldId] = newId
	m.ordered = append(m.ordered, [2]string{oldId, newId})
}

func (m *idRemapper) text(value string) string {
	if value == "" {
		return value
	}
	if mapped, ok := m.exact[value]; ok {
		return mapped
	}
	for _, pair := range m.ordered {
		value = strings.ReplaceAll(value, pair[0], pair[1])
	}
	return value
}

func (m *idRemapper) textPtr(value *string) *string {
	if value == nil {
		return nil
	}
	mapped := m.text(*value)
	return &mapped
}

func (s *OptimizationService) Apply(ctx context.Context, request domain.OptimizationApplyRequest, currentUser string) (*domain.OptimizationApplyResult, error) {
	rbdSystemId := strings.TrimSpace(domain.Deref(request.RbdSystemId))
	if rbdSystemId == "" {
		return nil, domain.Argument("RbdSystemId is required")
	}
	projectName := strings.TrimSpace(domain.Deref(request.ProjectName))
	if projectName == "" {
		return nil, domain.Argument("ProjectName is required")
	}
	built, err := s.buildProblem(ctx, rbdSystemId, request.RunningHours, nil)
	if err != nil {
		return nil, err
	}
	chosen := map[string]optimize.Candidate{}
	wanted := map[string]string{}
	for _, choice := range request.Choices {
		if choice.SystemComponentId != nil && choice.ComponentId != nil {
			wanted[*choice.SystemComponentId] = *choice.ComponentId
		}
	}
	for _, slot := range built.problem.Slots {
		target, requested := wanted[slot.SystemComponentId]
		if !requested {
			continue
		}
		found := false
		for _, candidate := range slot.Candidates {
			if candidate.ComponentId == target {
				chosen[slot.SystemComponentId] = candidate
				found = true
				break
			}
		}
		if !found {
			return nil, domain.Argument("Component " + slot.SystemComponentId + " has no compatible candidate " + target)
		}
	}
	result := &domain.OptimizationApplyResult{}
	err = s.store.Transact(ctx, func(tx *Store) error {
		taken, err := tx.ProjectNameExists(ctx, projectName)
		if err != nil {
			return err
		}
		if taken {
			return domain.InvalidOperation("Project Name Already Exist")
		}
		lastProject, err := tx.LastProjectId(ctx)
		if err != nil {
			return err
		}
		projectNumber := 0
		if lastProject != nil && strings.HasPrefix(*lastProject, "PJ-") {
			if parsed, err := strconv.Atoi((*lastProject)[3:]); err == nil {
				projectNumber = parsed
			}
		}
		newProjectId := "PJ-" + padNumber(projectNumber+1, 5)
		depth := 3
		if sourceProject, err := tx.FindProject(ctx, built.system.ProjectId); err != nil {
			return err
		} else if sourceProject != nil {
			depth = sourceProject.HierarchyDepth
		}
		now := domain.Now()
		if err := tx.InsertProject(ctx, domain.MasterProject{
			ProjectId:      newProjectId,
			ProjectName:    projectName,
			HierarchyDepth: depth,
			CreatedAt:      now,
			UpdatedAt:      now,
			CreatedBy:      domain.StringPtr(currentUser),
			UpdatedBy:      domain.StringPtr(currentUser),
		}); err != nil {
			return err
		}
		systemIds, err := tx.SystemIds(ctx)
		if err != nil {
			return err
		}
		newSystemId := nextSequenceId(systemIds, "RS-")
		hierarchies, err := tx.HierarchiesOfSystemByLevel(ctx, rbdSystemId, true)
		if err != nil {
			return err
		}
		components, err := tx.ComponentsOfSystem(ctx, rbdSystemId)
		if err != nil {
			return err
		}
		lastHierarchy, err := tx.LastHierarchyId(ctx)
		if err != nil {
			return err
		}
		lastComponent, err := tx.LastComponentId(ctx)
		if err != nil {
			return err
		}
		remap := newIdRemapper()
		hierarchyCounter := counterAfter(lastHierarchy, "H-")
		for _, hierarchy := range hierarchies {
			remap.add(hierarchy.HierarchyId, formatId("H-", hierarchyCounter))
			hierarchyCounter++
		}
		componentCounter := counterAfter(lastComponent, "SCP-")
		for _, component := range components {
			remap.add(component.SystemComponentId, formatId("SCP-", componentCounter))
			componentCounter++
		}
		remap.add(rbdSystemId, newSystemId)
		systemName := built.system.SystemName
		if trimmed := strings.TrimSpace(domain.Deref(request.SystemName)); trimmed != "" {
			systemName = domain.StringPtr(trimmed)
		}
		if err := tx.InsertSystem(ctx, domain.RbdSystemDrawing{
			RbdSystemId:  newSystemId,
			ProjectId:    newProjectId,
			DrawingName:  built.system.DrawingName,
			SystemName:   systemName,
			RunningHours: built.system.RunningHours,
			Formula:      remap.textPtr(built.system.Formula),
			CreatedAt:    now,
			UpdatedAt:    now,
			CreatedBy:    domain.StringPtr(currentUser),
			UpdatedBy:    domain.StringPtr(currentUser),
		}); err != nil {
			return err
		}
		newHierarchies := make([]domain.Hierarchy, 0, len(hierarchies))
		for _, hierarchy := range hierarchies {
			copied := hierarchy
			copied.HierarchyId = remap.text(hierarchy.HierarchyId)
			copied.RbdSystemId = domain.StringPtr(newSystemId)
			copied.ParentId = remap.text(hierarchy.ParentId)
			copied.Formula = remap.textPtr(hierarchy.Formula)
			copied.FormulaCode = remap.textPtr(hierarchy.FormulaCode)
			copied.SourceId = remap.textPtr(hierarchy.SourceId)
			copied.TargetId = remap.textPtr(hierarchy.TargetId)
			newHierarchies = append(newHierarchies, copied)
		}
		if err := tx.InsertHierarchies(ctx, newHierarchies); err != nil {
			return err
		}
		nodeIds := []string{}
		for _, hierarchy := range hierarchies {
			nodeIds = append(nodeIds, hierarchy.HierarchyId, domain.Deref(hierarchy.FormulaCode), domain.Deref(hierarchy.SourceId), domain.Deref(hierarchy.TargetId))
		}
		newComponents := make([]domain.SystemComponentProperties, 0, len(components))
		for _, component := range components {
			nodeIds = append(nodeIds, component.SystemComponentId, domain.Deref(component.IdNode), domain.Deref(component.FormulaCode))
			copied := component
			copied.SystemComponentId = remap.text(component.SystemComponentId)
			copied.RbdSystemId = domain.StringPtr(newSystemId)
			copied.ParentId = remap.textPtr(component.ParentId)
			copied.FormulaCode = remap.textPtr(component.FormulaCode)
			copied.IdNode = remap.textPtr(component.IdNode)
			copied.ConnectionToId = remap.textPtr(component.ConnectionToId)
			copied.CreatedAt = &now
			copied.UpdatedAt = &now
			copied.CreatedBy = domain.StringPtr(currentUser)
			copied.UpdatedBy = domain.StringPtr(currentUser)
			if candidate, swapped := chosen[component.SystemComponentId]; swapped {
				hours := 1000.0
				if request.RunningHours != nil && *request.RunningHours > 0 {
					hours = *request.RunningHours
				} else if component.RunningHours != nil {
					hours = reliability.ToFloat(component.RunningHours.Decimal)
				}
				rate, err := reliability.FromFloat(candidate.FailureRate)
				if err != nil {
					return err
				}
				value, err := reliability.FromFloat(optimize.ExponentialFloat(candidate.FailureRate, hours))
				if err != nil {
					return err
				}
				copied.Vendor = domain.StringPtr(candidate.VendorName)
				copied.FailureRate = domain.NumberPtr(domain.NewNumber(rate))
				copied.DistributionType = domain.StringPtr("Exponential")
				copied.ReliabilityValue = domain.NumberPtr(domain.NewNumber(value))
				copied.Regresi = nil
				if candidate.FailureRate > 0 {
					mtbf, err := reliability.FromFloat(1 / candidate.FailureRate)
					if err != nil {
						return err
					}
					copied.Mtbf = domain.NumberPtr(domain.NewNumber(mtbf))
				}
			}
			newComponents = append(newComponents, copied)
		}
		if err := tx.InsertComponents(ctx, newComponents); err != nil {
			return err
		}
		uniqueNodeIds := []string{}
		seen := map[string]bool{}
		for _, nodeId := range nodeIds {
			if nodeId != "" && !seen[nodeId] {
				seen[nodeId] = true
				uniqueNodeIds = append(uniqueNodeIds, nodeId)
			}
		}
		edges, err := tx.EdgesTouching(ctx, uniqueNodeIds)
		if err != nil {
			return err
		}
		newEdges := make([]domain.SystemComponentDrawing, 0, len(edges))
		for _, edge := range edges {
			newEdges = append(newEdges, domain.SystemComponentDrawing{
				IdEdge:   domain.NewGuid().String(),
				SourceId: remap.textPtr(edge.SourceId),
				TargetId: remap.textPtr(edge.TargetId),
			})
		}
		if err := tx.InsertEdges(ctx, newEdges); err != nil {
			return err
		}
		mode := 0
		if request.Mode != nil {
			mode = *request.Mode
		}
		var choicesJSON *string
		if len(request.Choices) > 0 {
			if encoded, err := json.Marshal(request.Choices); err == nil {
				choicesJSON = domain.StringPtr(string(encoded))
			}
		}
		if err := tx.InsertOptimizationRun(ctx, domain.OptimizationRun{
			OptimizationRunId:    domain.NewGuid().String(),
			RbdSystemId:          rbdSystemId,
			Mode:                 mode,
			MaxBudget:            request.MaxBudget,
			TargetReliability:    request.TargetReliability,
			WeightCost:           request.WeightCost,
			WeightReliability:    request.WeightReliability,
			RunningHours:         request.RunningHours,
			PopulationSize:       request.PopulationSize,
			MaxGenerations:       request.MaxGenerations,
			CrossoverProbability: request.CrossoverProbability,
			MutationProbability:  request.MutationProbability,
			Seed:                 request.Seed,
			Choices:              choicesJSON,
			ResultProjectId:      domain.StringPtr(newProjectId),
			ResultRbdSystemId:    domain.StringPtr(newSystemId),
			CreatedAt:            now,
			CreatedBy:            domain.StringPtr(currentUser),
		}); err != nil {
			return err
		}
		result.ProjectId = newProjectId
		result.RbdSystemId = newSystemId
		return nil
	})
	if err != nil {
		return nil, err
	}
	totals := NewTotalService(s.store)
	calcCtx := WithUser(ctx, currentUser)
	roots, err := s.store.RootHierarchies(calcCtx, result.RbdSystemId)
	if err == nil {
		for _, root := range roots {
			totals.CalculateHierarchy(calcCtx, root.HierarchyId)
		}
	}
	return result, nil
}

func padNumber(value, width int) string {
	text := strconv.Itoa(value)
	for len(text) < width {
		text = "0" + text
	}
	return text
}
