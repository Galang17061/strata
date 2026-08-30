package rbd

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/reliability"
)

type DrawingService struct {
	store *Store
}

func NewDrawingService(store *Store) *DrawingService {
	return &DrawingService{store: store}
}

func codesOf(components []domain.SystemComponentProperties, hierarchies []domain.Hierarchy, distinct bool) []string {
	codes := []string{}
	seen := map[string]bool{}
	add := func(code string) {
		if code == "" {
			return
		}
		if distinct {
			if seen[code] {
				return
			}
			seen[code] = true
		}
		codes = append(codes, code)
	}
	for _, component := range components {
		add(domain.Deref(component.FormulaCode))
	}
	for _, hierarchy := range hierarchies {
		add(domain.Deref(hierarchy.FormulaCode))
	}
	return codes
}

func toEdges(rows []domain.SystemComponentDrawing) []reliability.Edge {
	edges := make([]reliability.Edge, 0, len(rows))
	for _, row := range rows {
		edges = append(edges, reliability.Edge{SourceId: domain.Deref(row.SourceId), TargetId: domain.Deref(row.TargetId)})
	}
	return edges
}

func (st *Store) formulaForHierarchy(ctx context.Context, hierarchyId string) (string, error) {
	components, err := st.ComponentsOfParent(ctx, hierarchyId, true)
	if err != nil {
		return "", err
	}
	children, err := st.ChildHierarchies(ctx, hierarchyId)
	if err != nil {
		return "", err
	}
	edges, err := st.EdgesTouching(ctx, codesOf(components, children, false))
	if err != nil {
		return "", err
	}
	return reliability.StripVirtualCodes(reliability.GenerateFormula(toEdges(edges))), nil
}

func (st *Store) regenerateTree(ctx context.Context, rbdSystemId, startingHierarchyId string) error {
	all, err := st.HierarchiesOfSystem(ctx, rbdSystemId)
	if err != nil {
		return err
	}
	byId := map[string]domain.Hierarchy{}
	for _, hierarchy := range all {
		byId[hierarchy.HierarchyId] = hierarchy
	}
	affected := []domain.Hierarchy{}
	if current, ok := byId[startingHierarchyId]; ok {
		affected = append(affected, current)
		parentId := current.ParentId
		for parentId != "" && parentId != rbdSystemId {
			parent, ok := byId[parentId]
			if !ok {
				break
			}
			affected = append(affected, parent)
			parentId = parent.ParentId
		}
	}
	domain.OrderBy(affected, true, func(a, b domain.Hierarchy) int { return domain.CompareInt64(int64(a.Level), int64(b.Level)) })
	for _, hierarchy := range affected {
		formula, err := st.formulaForHierarchy(ctx, hierarchy.HierarchyId)
		if err != nil {
			return err
		}
		hierarchy.Formula = domain.StringPtr(formula)
		if err := st.UpdateHierarchy(ctx, hierarchy); err != nil {
			return err
		}
	}
	return nil
}

func (st *Store) regenerateSystem(ctx context.Context, rbdSystemId string) error {
	hierarchies, err := st.HierarchiesOfSystemLevelDescending(ctx, rbdSystemId)
	if err != nil {
		return err
	}
	for _, hierarchy := range hierarchies {
		formula, err := st.formulaForHierarchy(ctx, hierarchy.HierarchyId)
		if err != nil {
			return err
		}
		hierarchy.Formula = domain.StringPtr(formula)
		if err := st.UpdateHierarchy(ctx, hierarchy); err != nil {
			return err
		}
	}
	return nil
}

func replaceCodes(formula string, codes []string, values map[string]decimal.Decimal) string {
	replaced := formula
	for _, code := range codes {
		if value, ok := values[code]; ok {
			replaced = strings.ReplaceAll(replaced, code, domain.NewNumber(value).Text())
		}
	}
	return replaced
}

func (st *Store) hierarchyReliabilityValue(ctx context.Context, hierarchyId, formula string) (*domain.Number, error) {
	if formula == "" {
		return nil, nil
	}
	hierarchy, err := st.FindHierarchy(ctx, hierarchyId)
	if err != nil || hierarchy == nil {
		return nil, err
	}
	children, err := st.ChildHierarchies(ctx, hierarchyId)
	if err != nil {
		return nil, err
	}
	codes := []string{}
	values := map[string]decimal.Decimal{}
	if len(children) > 0 {
		for _, child := range children {
			code := domain.Deref(child.FormulaCode)
			if code != "" && child.RealibilityValue != nil {
				if _, exists := values[code]; !exists {
					codes = append(codes, code)
				}
				values[code] = child.RealibilityValue.Decimal
			}
		}
	} else {
		components, err := st.ComponentsOfParent(ctx, hierarchyId, true)
		if err != nil {
			return nil, err
		}
		for _, component := range components {
			code := domain.Deref(component.FormulaCode)
			if code != "" && component.ReliabilityValue != nil {
				if _, exists := values[code]; !exists {
					codes = append(codes, code)
				}
				values[code] = component.ReliabilityValue.Decimal
			}
		}
	}
	if len(values) == 0 {
		return nil, nil
	}
	result, err := reliability.Evaluate(reliability.PrepareFormula(replaceCodes(formula, codes, values), false))
	if err != nil {
		return nil, nil
	}
	return domain.NumberPtr(domain.NewNumber(result)), nil
}

func (s *DrawingService) SaveNodesByHierarchy(ctx context.Context, hierarchyId string, inputs []domain.DrawingNodeInput, currentUser string) error {
	if len(inputs) == 0 {
		return nil
	}
	return s.store.Transact(ctx, func(tx *Store) error {
		hierarchy, err := tx.FindHierarchy(ctx, hierarchyId)
		if err != nil {
			return err
		}
		if hierarchy == nil {
			return domain.KeyNotFound("Hierarchy " + hierarchyId + " not found")
		}
		if err := tx.TouchSystem(ctx, domain.Deref(hierarchy.RbdSystemId)); err != nil {
			return err
		}
		idNodes := []string{}
		for _, input := range inputs {
			if id := domain.Deref(input.IdNode); id != "" {
				idNodes = append(idNodes, id)
			}
		}
		components, err := tx.ComponentsByCodesInParent(ctx, idNodes, hierarchyId)
		if err != nil {
			return err
		}
		hierarchies, err := tx.HierarchiesByCodesInParent(ctx, idNodes, hierarchyId, false)
		if err != nil {
			return err
		}
		if len(components) > 0 && len(hierarchies) > 0 {
			return domain.InvalidOperation("Cannot mix components and subsystems in the same save operation. All nodes must be of the same type.")
		}
		if len(components) == 0 && len(hierarchies) == 0 {
			return domain.KeyNotFound("No matching components or hierarchies found for the provided IdNodes in this hierarchy.")
		}
		if len(hierarchies) > 0 {
			if err := checkSingleLevel(hierarchies, "All subsystems must be at the same level. Found levels: "); err != nil {
				return err
			}
			for _, input := range inputs {
				target := findHierarchyByCode(hierarchies, domain.Deref(input.IdNode))
				if target == nil {
					return domain.KeyNotFound("Hierarchy with FormulaCode '" + domain.Deref(input.IdNode) + "' not found in parent " + hierarchyId)
				}
				taken, err := tx.OtherHierarchyUsesCode(ctx, domain.Deref(input.IdNode), target.HierarchyId, hierarchyId)
				if err != nil {
					return err
				}
				if taken {
					return domain.InvalidOperation("IdNode '" + domain.Deref(input.IdNode) + "' already exists in this hierarchy. Please choose another IdNode.")
				}
				applyHierarchyPosition(target, input)
				if err := tx.UpdateHierarchy(ctx, *target); err != nil {
					return err
				}
			}
			return nil
		}
		for _, input := range inputs {
			target := findComponentByCode(components, domain.Deref(input.IdNode))
			if target == nil {
				return domain.KeyNotFound("Component with FormulaCode '" + domain.Deref(input.IdNode) + "' not found in parent " + hierarchyId)
			}
			taken, err := tx.OtherComponentUsesCode(ctx, domain.Deref(input.IdNode), target.SystemComponentId, hierarchyId)
			if err != nil {
				return err
			}
			if taken {
				return domain.InvalidOperation("IdNode '" + domain.Deref(input.IdNode) + "' already exists in this hierarchy. Please choose another IdNode.")
			}
			target.Active = domain.IntPtr(1)
			target.ConnectionType = input.ConnectionType
			target.PositionX = input.PositionX
			target.PositionY = input.PositionY
			target.IdNode = input.IdNode
			target.UpdatedBy = domain.StringPtr(currentUser)
			now := domain.Now()
			target.UpdatedAt = &now
			if err := tx.UpdateComponent(ctx, *target); err != nil {
				return err
			}
		}
		return nil
	})
}

func checkSingleLevel(hierarchies []domain.Hierarchy, prefix string) error {
	levels := []string{}
	seen := map[int]bool{}
	for _, hierarchy := range hierarchies {
		if hierarchy.Level == 999 || seen[hierarchy.Level] {
			continue
		}
		seen[hierarchy.Level] = true
		levels = append(levels, fmt.Sprintf("%d", hierarchy.Level))
	}
	if len(levels) > 1 {
		return domain.InvalidOperation(prefix + strings.Join(levels, ", "))
	}
	return nil
}

func findHierarchyByCode(hierarchies []domain.Hierarchy, code string) *domain.Hierarchy {
	for index := range hierarchies {
		if domain.Deref(hierarchies[index].FormulaCode) == code {
			return &hierarchies[index]
		}
	}
	return nil
}

func findComponentByCode(components []domain.SystemComponentProperties, code string) *domain.SystemComponentProperties {
	for index := range components {
		if domain.Deref(components[index].FormulaCode) == code {
			return &components[index]
		}
	}
	return nil
}

func applyHierarchyPosition(target *domain.Hierarchy, input domain.DrawingNodeInput) {
	target.ConnectionType = input.ConnectionType
	if text := domain.Deref(input.PositionX); text != "" {
		if parsed, err := domain.NumberFromString(strings.TrimSpace(text)); err == nil {
			target.PositionX = &parsed
		}
	}
	if text := domain.Deref(input.PositionY); text != "" {
		if parsed, err := domain.NumberFromString(strings.TrimSpace(text)); err == nil {
			target.PositionY = &parsed
		}
	}
}

func (s *DrawingService) SaveNodesBySystem(ctx context.Context, rbdSystemId string, inputs []domain.DrawingNodeInput) error {
	if len(inputs) == 0 {
		return nil
	}
	return s.store.Transact(ctx, func(tx *Store) error {
		system, err := tx.FindSystem(ctx, rbdSystemId)
		if err != nil {
			return err
		}
		if system == nil {
			return domain.KeyNotFound("RBD System " + rbdSystemId + " not found")
		}
		if err := tx.TouchSystem(ctx, rbdSystemId); err != nil {
			return err
		}
		idNodes := []string{}
		for _, input := range inputs {
			if id := domain.Deref(input.IdNode); id != "" {
				idNodes = append(idNodes, id)
			}
		}
		hierarchies, err := tx.HierarchiesByCodesInParent(ctx, idNodes, rbdSystemId, true)
		if err != nil {
			return err
		}
		if len(hierarchies) == 0 {
			return domain.KeyNotFound("No matching level 1 hierarchies or virtual nodes found for the provided IdNodes in this RBD System.")
		}
		if err := checkSingleLevel(hierarchies, "All subsystems must be at level 1 for RBD System. Found levels: "); err != nil {
			return err
		}
		for _, input := range inputs {
			target := findHierarchyByCode(hierarchies, domain.Deref(input.IdNode))
			if target == nil {
				return domain.KeyNotFound("Hierarchy with FormulaCode '" + domain.Deref(input.IdNode) + "' not found in RBD System " + rbdSystemId)
			}
			taken, err := tx.OtherHierarchyUsesCode(ctx, domain.Deref(input.IdNode), target.HierarchyId, rbdSystemId)
			if err != nil {
				return err
			}
			if taken {
				return domain.InvalidOperation("IdNode '" + domain.Deref(input.IdNode) + "' already exists in this RBD System. Please choose another IdNode.")
			}
			applyHierarchyPosition(target, input)
			if err := tx.UpdateHierarchy(ctx, *target); err != nil {
				return err
			}
		}
		return nil
	})
}

func mappedEdges(inputs []domain.EdgeInput, idToCode map[string]string) []domain.SystemComponentDrawing {
	edges := make([]domain.SystemComponentDrawing, 0, len(inputs))
	for _, input := range inputs {
		source := domain.Deref(input.SourceId)
		target := domain.Deref(input.TargetId)
		if mapped, ok := idToCode[source]; ok {
			source = mapped
		}
		if mapped, ok := idToCode[target]; ok {
			target = mapped
		}
		edges = append(edges, domain.SystemComponentDrawing{IdEdge: domain.Deref(input.IdEdge), SourceId: domain.StringPtr(source), TargetId: domain.StringPtr(target)})
	}
	return edges
}

func (s *DrawingService) SaveEdgesByHierarchy(ctx context.Context, hierarchyId string, inputs []domain.EdgeInput) error {
	return s.store.Transact(ctx, func(tx *Store) error {
		hierarchy, err := tx.FindHierarchy(ctx, hierarchyId)
		if err != nil {
			return err
		}
		if hierarchy == nil {
			return domain.KeyNotFound("Hierarchy " + hierarchyId + " not found")
		}
		components, err := tx.ComponentsOfParent(ctx, hierarchyId, true)
		if err != nil {
			return err
		}
		children, err := tx.ChildHierarchies(ctx, hierarchyId)
		if err != nil {
			return err
		}
		idToCode := map[string]string{}
		for _, component := range components {
			if code := domain.Deref(component.FormulaCode); code != "" {
				idToCode[component.SystemComponentId] = code
			}
		}
		for _, child := range children {
			if code := domain.Deref(child.FormulaCode); code != "" {
				idToCode[child.HierarchyId] = code
			}
		}
		if err := tx.DeleteEdgesTouching(ctx, codesOf(components, children, true)); err != nil {
			return err
		}
		if err := tx.InsertEdges(ctx, mappedEdges(inputs, idToCode)); err != nil {
			return err
		}
		if err := tx.regenerateTree(ctx, domain.Deref(hierarchy.RbdSystemId), hierarchyId); err != nil {
			return err
		}
		return tx.TouchSystem(ctx, domain.Deref(hierarchy.RbdSystemId))
	})
}

func (s *DrawingService) SaveEdgesBySystem(ctx context.Context, rbdSystemId string, inputs []domain.EdgeInput, currentUser string) error {
	return s.store.Transact(ctx, func(tx *Store) error {
		system, err := tx.FindSystem(ctx, rbdSystemId)
		if err != nil {
			return err
		}
		if system == nil {
			return domain.KeyNotFound("RBD System " + rbdSystemId + " not found")
		}
		levelOne, err := tx.HierarchiesByParentAndLevel(ctx, rbdSystemId, 1)
		if err != nil {
			return err
		}
		idToCode := map[string]string{}
		for _, hierarchy := range levelOne {
			if code := domain.Deref(hierarchy.FormulaCode); code != "" {
				idToCode[hierarchy.HierarchyId] = code
			}
		}
		if err := tx.DeleteEdgesTouching(ctx, codesOf(nil, levelOne, true)); err != nil {
			return err
		}
		if err := tx.InsertEdges(ctx, mappedEdges(inputs, idToCode)); err != nil {
			return err
		}
		return tx.regenerateSystemLevel(ctx, rbdSystemId, currentUser)
	})
}

func (st *Store) HierarchiesByParentAndLevel(ctx context.Context, parentId string, level int) ([]domain.Hierarchy, error) {
	rows := []domain.Hierarchy{}
	return rows, st.q.SelectContext(ctx, &rows, `SELECT `+domain.HierarchyColumns+` FROM dbo.Hierarchy WHERE parent_id = @p1 AND level = @p2 ORDER BY hierarchy_id`, parentId, level)
}

func (st *Store) regenerateSystemLevel(ctx context.Context, rbdSystemId, currentUser string) error {
	levelOne, err := st.LevelOneHierarchies(ctx, rbdSystemId)
	if err != nil {
		return err
	}
	for index := range levelOne {
		hierarchy := &levelOne[index]
		formula, err := st.formulaForHierarchy(ctx, hierarchy.HierarchyId)
		if err != nil {
			return domain.KeyNotFound("Error generating formula for hierarchy " + hierarchy.HierarchyId + ": " + err.Error())
		}
		if formula == "" {
			continue
		}
		hierarchy.Formula = domain.StringPtr(formula)
		value, err := st.hierarchyReliabilityValue(ctx, hierarchy.HierarchyId, formula)
		if err != nil {
			return domain.KeyNotFound("Error generating formula for hierarchy " + hierarchy.HierarchyId + ": " + err.Error())
		}
		hierarchy.RealibilityValue = value
		if err := st.UpdateHierarchy(ctx, *hierarchy); err != nil {
			return domain.KeyNotFound("Error generating formula for hierarchy " + hierarchy.HierarchyId + ": " + err.Error())
		}
	}
	return st.calculateSystemFormula(ctx, rbdSystemId, levelOne, currentUser)
}

func (st *Store) calculateSystemFormula(ctx context.Context, rbdSystemId string, levelOne []domain.Hierarchy, currentUser string) error {
	if len(levelOne) == 0 {
		return nil
	}
	codes := codesOf(nil, levelOne, true)
	if len(codes) == 0 {
		return nil
	}
	rows, err := st.EdgesTouching(ctx, codes)
	if err != nil {
		return err
	}
	codeSet := map[string]bool{}
	for _, code := range codes {
		codeSet[code] = true
	}
	edges := []reliability.Edge{}
	for _, row := range rows {
		if codeSet[domain.Deref(row.SourceId)] && codeSet[domain.Deref(row.TargetId)] {
			edges = append(edges, reliability.Edge{SourceId: domain.Deref(row.SourceId), TargetId: domain.Deref(row.TargetId)})
		}
	}
	if len(edges) == 0 {
		if len(levelOne) == 1 {
			edges = []reliability.Edge{{SourceId: domain.Deref(levelOne[0].FormulaCode), TargetId: "OUT"}}
		} else {
			for index := 0; index < len(levelOne)-1; index++ {
				edges = append(edges, reliability.Edge{SourceId: domain.Deref(levelOne[index].FormulaCode), TargetId: domain.Deref(levelOne[index+1].FormulaCode)})
			}
		}
	}
	formula := reliability.StripVirtualCodes(reliability.GenerateFormula(edges))
	system, err := st.FindSystem(ctx, rbdSystemId)
	if err != nil || system == nil {
		return err
	}
	system.Formula = domain.StringPtr(formula)
	lookupCodes := []string{}
	values := map[string]decimal.Decimal{}
	for _, hierarchy := range levelOne {
		code := domain.Deref(hierarchy.FormulaCode)
		if code != "" && hierarchy.RealibilityValue != nil {
			if _, exists := values[code]; !exists {
				lookupCodes = append(lookupCodes, code)
			}
			values[code] = hierarchy.RealibilityValue.Decimal
		}
	}
	if len(values) > 0 && formula != "" {
		if result, err := reliability.Evaluate(reliability.PrepareFormula(replaceCodes(formula, lookupCodes, values), false)); err == nil {
			system.ReliabilityTotal = domain.NumberPtr(domain.NewNumber(result))
		}
	}
	system.UpdatedBy = domain.StringPtr(currentUser)
	system.UpdatedAt = domain.Now()
	return st.UpdateSystem(ctx, *system)
}

func positionText(value *domain.Number) *string {
	if value == nil {
		return nil
	}
	return domain.StringPtr(value.Decimal.StringFixed(2))
}

func hierarchyNodeViews(rows []domain.Hierarchy) []domain.ComponentNodeView {
	views := make([]domain.ComponentNodeView, 0, len(rows))
	for _, row := range rows {
		views = append(views, domain.ComponentNodeView{
			SystemComponentId: row.HierarchyId,
			ConnectionType:    row.ConnectionType,
			PositionX:         positionText(row.PositionX),
			PositionY:         positionText(row.PositionY),
			IdNode:            row.FormulaCode,
			ComponentName:     row.SubSystemName,
		})
	}
	return views
}

func (s *DrawingService) NodesByHierarchy(ctx context.Context, hierarchyId string) ([]domain.ComponentNodeView, error) {
	components, err := s.store.ComponentsOfParent(ctx, hierarchyId, true)
	if err != nil {
		return nil, err
	}
	views := []domain.ComponentNodeView{}
	for _, component := range components {
		views = append(views, domain.ComponentNodeView{
			SystemComponentId: component.SystemComponentId,
			ConnectionType:    component.ConnectionType,
			PositionX:         component.PositionX,
			PositionY:         component.PositionY,
			IdNode:            component.FormulaCode,
			ComponentName:     domain.StringPtr(component.ComponentName),
			VendorName:        component.Vendor,
			ActiveComponent:   component.ActiveComponent,
		})
	}
	children, err := s.store.ChildHierarchiesByVirtual(ctx, hierarchyId, false)
	if err != nil {
		return nil, err
	}
	virtuals, err := s.store.ChildHierarchiesByVirtual(ctx, hierarchyId, true)
	if err != nil {
		return nil, err
	}
	views = append(views, hierarchyNodeViews(children)...)
	views = append(views, hierarchyNodeViews(virtuals)...)
	return views, nil
}

func (s *DrawingService) NodesBySystem(ctx context.Context, rbdSystemId string) ([]domain.HierarchyNodeView, error) {
	views := []domain.HierarchyNodeView{}
	for _, virtual := range []bool{false, true} {
		rows, err := s.store.RootHierarchiesByLevel(ctx, rbdSystemId, virtual)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			views = append(views, domain.HierarchyNodeView{
				HierarchyId:    row.HierarchyId,
				ConnectionType: row.ConnectionType,
				PositionX:      positionText(row.PositionX),
				PositionY:      positionText(row.PositionY),
				IdNode:         row.FormulaCode,
				ComponentName:  row.SubSystemName,
			})
		}
	}
	return views, nil
}

func edgeViews(rows []domain.SystemComponentDrawing) []domain.EdgeView {
	views := make([]domain.EdgeView, 0, len(rows))
	for _, row := range rows {
		views = append(views, domain.EdgeView{IdEdge: domain.StringPtr(row.IdEdge), SourceId: row.SourceId, TargetId: row.TargetId})
	}
	return views
}

func (s *DrawingService) EdgesByHierarchy(ctx context.Context, hierarchyId string) ([]domain.EdgeView, error) {
	components, err := s.store.ComponentsOfParent(ctx, hierarchyId, true)
	if err != nil {
		return nil, err
	}
	children, err := s.store.ChildHierarchies(ctx, hierarchyId)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.EdgesTouching(ctx, codesOf(components, children, false))
	if err != nil {
		return nil, err
	}
	return edgeViews(rows), nil
}

func (s *DrawingService) EdgesBySystem(ctx context.Context, rbdSystemId string) ([]domain.EdgeView, error) {
	hierarchies, err := s.store.RootHierarchies(ctx, rbdSystemId)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.EdgesTouching(ctx, codesOf(nil, hierarchies, false))
	if err != nil {
		return nil, err
	}
	return edgeViews(rows), nil
}
