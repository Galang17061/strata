package rbd

import (
	"context"
	"strings"

	"github.com/Galang17061/strata-api/internal/domain"
)

type SystemService struct {
	store *Store
}

func NewSystemService(store *Store) *SystemService {
	return &SystemService{store: store}
}

func (s *SystemService) ListSystems(ctx context.Context, projectId *string) ([]domain.SystemView, error) {
	return s.store.ListSystemViews(ctx, projectId)
}

func (s *SystemService) FirstSystem(ctx context.Context) (*domain.RbdSystemDrawing, error) {
	return s.store.FirstSystem(ctx)
}

func (s *SystemService) Tree(ctx context.Context, rbdSystemId string, systemName *string) (*domain.SystemTree, error) {
	return s.store.tree(ctx, rbdSystemId, systemName)
}

func (st *Store) tree(ctx context.Context, rbdSystemId string, systemName *string) (*domain.SystemTree, error) {
	system, err := st.FindSystem(ctx, rbdSystemId)
	if err != nil || system == nil {
		return nil, err
	}
	if systemName != nil && *systemName != "" && domain.Deref(system.SystemName) != *systemName {
		return nil, nil
	}
	project, err := st.FindProject(ctx, system.ProjectId)
	if err != nil {
		return nil, err
	}
	hierarchies, err := st.HierarchiesOfSystemByLevel(ctx, rbdSystemId, false)
	if err != nil {
		return nil, err
	}
	allComponents, err := st.ComponentsOfSystem(ctx, rbdSystemId)
	if err != nil {
		return nil, err
	}
	components := []domain.SystemComponentProperties{}
	for _, component := range allComponents {
		if domain.Deref(component.ConnectionType) != "virtual" {
			components = append(components, component)
		}
	}
	edges, err := st.AllEdges(ctx)
	if err != nil {
		return nil, err
	}
	tree := &domain.SystemTree{RbdSystemId: system.RbdSystemId, ProjectId: system.ProjectId, HierarchyDepth: 3, SystemName: domain.Deref(system.SystemName)}
	if project != nil {
		tree.ProjectName = project.ProjectName
		tree.HierarchyDepth = project.HierarchyDepth
	}
	visited := map[string]bool{}
	nodes := []domain.TreeNode{}
	for _, hierarchy := range hierarchies {
		if hierarchy.ParentId != rbdSystemId {
			continue
		}
		if node := buildTreeNode(hierarchy, hierarchies, components, edges, visited); node != nil {
			nodes = append(nodes, *node)
		}
	}
	if len(nodes) > 0 {
		tree.Hierarchy = nodes
	}
	return tree, nil
}

func buildTreeNode(current domain.Hierarchy, hierarchies []domain.Hierarchy, components []domain.SystemComponentProperties, edges []domain.SystemComponentDrawing, visited map[string]bool) *domain.TreeNode {
	if visited[current.HierarchyId] {
		return nil
	}
	visited[current.HierarchyId] = true
	children := []domain.TreeNode{}
	for _, hierarchy := range hierarchies {
		if hierarchy.ParentId == current.HierarchyId && !visited[hierarchy.HierarchyId] {
			if child := buildTreeNode(hierarchy, hierarchies, components, edges, visited); child != nil {
				children = append(children, *child)
			}
		}
	}
	nodeComponents := []domain.ComponentNode{}
	for _, component := range components {
		if domain.Deref(component.ParentId) != current.HierarchyId {
			continue
		}
		targets := []string{}
		for _, edge := range edges {
			if domain.Deref(edge.SourceId) == component.SystemComponentId {
				targets = append(targets, domain.Deref(edge.TargetId))
			}
		}
		nodeComponents = append(nodeComponents, domain.ComponentNode{
			SystemComponentId: component.SystemComponentId,
			FormulaCode:       domain.StringPtr(domain.Deref(component.FormulaCode)),
			ComponentName:     domain.StringPtr(component.ComponentName),
			VendorName:        component.Vendor,
			TotalComponent:    domain.IntPtr(domain.DerefInt(component.TotalComponent, 0)),
			ActiveComponent:   domain.IntPtr(domain.DerefInt(component.ActiveComponent, 0)),
			ConnectionType:    component.ConnectionType,
			TargetEdges:       targets,
		})
	}
	node := &domain.TreeNode{
		HierarchyId:    current.HierarchyId,
		Name:           domain.Deref(current.SubSystemName),
		ConnectionType: current.ConnectionType,
		FormulaCode:    current.FormulaCode,
		Level:          current.Level,
	}
	if len(children) > 0 {
		node.Hierarchy = children
	}
	if len(nodeComponents) > 0 {
		node.Components = nodeComponents
	}
	return node
}

func (s *SystemService) InputParameters(ctx context.Context, hierarchyId string) ([]domain.ComponentInputParameters, error) {
	components, err := s.store.ComponentsOfParent(ctx, hierarchyId, true)
	if err != nil {
		return nil, err
	}
	result := []domain.ComponentInputParameters{}
	for _, component := range components {
		if strings.Contains(component.ComponentName, "Virtual Node") {
			continue
		}
		result = append(result, domain.ComponentInputParameters{
			ComponentName:        component.ComponentName,
			VendorName:           component.Vendor,
			FailureRate:          component.FailureRate,
			RunningHours:         component.RunningHours,
			FormulaCode:          component.FormulaCode,
			ComponentReliability: component.ReliabilityValue,
		})
	}
	return result, nil
}

func (s *SystemService) InputOutputParameters(ctx context.Context, hierarchyId string) ([]domain.ComponentInputOutputParameters, error) {
	components, err := s.store.ComponentsOfParent(ctx, hierarchyId, true)
	if err != nil {
		return nil, err
	}
	type lookup struct {
		cost   *string
		serial *string
	}
	lookups := map[string]lookup{}
	result := []domain.ComponentInputOutputParameters{}
	for _, component := range components {
		if strings.Contains(component.ComponentName, "Virtual Node") {
			continue
		}
		key := component.ComponentName + "|" + domain.Deref(component.Vendor)
		if _, ok := lookups[key]; !ok {
			vendor := domain.Deref(component.Vendor)
			master, err := s.store.MasterComponentByNameAndVendor(ctx, component.ComponentName, &vendor)
			if err != nil {
				return nil, err
			}
			entry := lookup{}
			if master != nil {
				entry.cost = master.Cost
				entry.serial = master.SerialNumber
			}
			lookups[key] = entry
		}
		entry := lookups[key]
		result = append(result, domain.ComponentInputOutputParameters{
			SystemComponentId:    component.SystemComponentId,
			ComponentName:        component.ComponentName,
			VendorName:           component.Vendor,
			FailureRate:          component.FailureRate,
			RunningHours:         component.RunningHours,
			FormulaCode:          component.FormulaCode,
			Cost:                 entry.cost,
			ActiveComponent:      component.ActiveComponent,
			TotalComponent:       component.TotalComponent,
			SerialNumber:         entry.serial,
			DistributionType:     component.DistributionType,
			ShapeParameter:       component.ShapeParameter,
			ScaleParameter:       component.ScaleParameter,
			ComponentReliability: component.ReliabilityValue,
			Mtbf:                 component.Mtbf,
			AllowedFailures:      component.AllowedFailures,
		})
	}
	return result, nil
}

func (s *SystemService) Delete(ctx context.Context, rbdSystemId string) error {
	return s.store.Transact(ctx, func(tx *Store) error {
		system, err := tx.FindSystem(ctx, rbdSystemId)
		if err != nil || system == nil {
			return err
		}
		if err := tx.DeleteHistoryOfSystem(ctx, rbdSystemId); err != nil {
			return err
		}
		components, err := tx.ComponentsOfSystem(ctx, rbdSystemId)
		if err != nil {
			return err
		}
		ids := make([]string, 0, len(components))
		for _, component := range components {
			ids = append(ids, component.SystemComponentId)
		}
		if err := tx.DeleteComponentDependents(ctx, ids); err != nil {
			return err
		}
		if err := tx.DeleteComponents(ctx, ids); err != nil {
			return err
		}
		hierarchies, err := tx.HierarchiesOfSystem(ctx, rbdSystemId)
		if err != nil {
			return err
		}
		hierarchyIds := make([]string, 0, len(hierarchies))
		for _, hierarchy := range hierarchies {
			hierarchyIds = append(hierarchyIds, hierarchy.HierarchyId)
		}
		if err := tx.DeleteHierarchies(ctx, hierarchyIds); err != nil {
			return err
		}
		return tx.DeleteSystem(ctx, rbdSystemId)
	})
}
