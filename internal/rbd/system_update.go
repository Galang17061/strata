package rbd

import (
	"context"
	"strings"

	"github.com/Galang17061/strata-api/internal/domain"
)

type hydration struct {
	tx                     *Store
	rbdSystemId            string
	currentUser            string
	hierarchiesById        map[string]*domain.Hierarchy
	hierarchyOrder         []string
	hierarchiesByParent    map[string][]domain.Hierarchy
	componentsByParent     map[string][]domain.SystemComponentProperties
	existingHierarchyCodes map[string]bool
	existingComponentCodes map[string]bool
	masterCache            map[string]domain.MasterComponentWithVendor
	masterOrder            []string
	nextHierarchyCounter   int
	nextComponentCounter   int
	hierarchiesToAdd       []domain.Hierarchy
	hierarchiesToMove      []domain.Hierarchy
	componentsToAdd        []domain.SystemComponentProperties
	componentsToUpdate     []domain.SystemComponentProperties
	hierarchyIdsToDelete   []string
	componentIdsToDelete   []string
	usedCodes              map[string]bool
}

func (s *SystemService) Update(ctx context.Context, rbdSystemId string, request domain.SystemUpdate, currentUser string) (*domain.SystemTree, error) {
	var tree *domain.SystemTree
	err := s.store.Transact(ctx, func(tx *Store) error {
		system, err := tx.FindSystem(ctx, rbdSystemId)
		if err != nil {
			return err
		}
		if system == nil {
			return domain.InvalidOperation("RBD System " + rbdSystemId + " not found")
		}
		system.SystemName = request.SystemName
		system.UpdatedBy = domain.StringPtr(currentUser)
		system.UpdatedAt = domain.Now()
		h, err := preloadHydration(ctx, tx, rbdSystemId, currentUser)
		if err != nil {
			return err
		}
		if len(request.Hierarchy) > 0 {
			if err := h.processHierarchies(ctx, request.Hierarchy, rbdSystemId); err != nil {
				return err
			}
		} else {
			if err := h.deleteEverything(ctx); err != nil {
				return err
			}
		}
		if err := tx.UpdateSystem(ctx, *system); err != nil {
			return err
		}
		tree, err = tx.tree(ctx, rbdSystemId, nil)
		return err
	})
	return tree, err
}

func preloadHydration(ctx context.Context, tx *Store, rbdSystemId, currentUser string) (*hydration, error) {
	h := &hydration{
		tx:                     tx,
		rbdSystemId:            rbdSystemId,
		currentUser:            currentUser,
		hierarchiesById:        map[string]*domain.Hierarchy{},
		hierarchiesByParent:    map[string][]domain.Hierarchy{},
		componentsByParent:     map[string][]domain.SystemComponentProperties{},
		existingHierarchyCodes: map[string]bool{},
		existingComponentCodes: map[string]bool{},
		masterCache:            map[string]domain.MasterComponentWithVendor{},
		usedCodes:              map[string]bool{},
	}
	hierarchies, err := tx.HierarchiesOfSystem(ctx, rbdSystemId)
	if err != nil {
		return nil, err
	}
	for index := range hierarchies {
		hierarchy := hierarchies[index]
		copyOf := hierarchy
		h.hierarchiesById[hierarchy.HierarchyId] = &copyOf
		h.hierarchyOrder = append(h.hierarchyOrder, hierarchy.HierarchyId)
		h.hierarchiesByParent[hierarchy.ParentId] = append(h.hierarchiesByParent[hierarchy.ParentId], hierarchy)
		if code := domain.Deref(hierarchy.FormulaCode); code != "" {
			h.existingHierarchyCodes[code] = true
		}
	}
	components, err := tx.ComponentsOfSystem(ctx, rbdSystemId)
	if err != nil {
		return nil, err
	}
	for _, component := range components {
		parent := domain.Deref(component.ParentId)
		h.componentsByParent[parent] = append(h.componentsByParent[parent], component)
		if code := domain.Deref(component.FormulaCode); code != "" {
			h.existingComponentCodes[code] = true
		}
	}
	lastHierarchy, err := tx.LastHierarchyId(ctx)
	if err != nil {
		return nil, err
	}
	h.nextHierarchyCounter = counterAfter(lastHierarchy, "H-")
	lastComponent, err := tx.LastComponentId(ctx)
	if err != nil {
		return nil, err
	}
	h.nextComponentCounter = counterAfter(lastComponent, "SCP-")
	masters, err := tx.MasterComponentsWithVendor(ctx)
	if err != nil {
		return nil, err
	}
	for _, master := range masters {
		key := master.ComponentName + "|" + domain.Deref(master.ManufacturerName)
		if _, exists := h.masterCache[key]; !exists {
			h.masterOrder = append(h.masterOrder, key)
		}
		h.masterCache[key] = master
	}
	return h, nil
}

func (h *hydration) processHierarchies(ctx context.Context, incoming []domain.TreeInput, parentId string) error {
	current := []domain.Hierarchy{}
	for _, hierarchy := range h.hierarchiesByParent[parentId] {
		if hierarchy.Level != 999 {
			current = append(current, hierarchy)
		}
	}
	incomingIds := map[string]bool{}
	for _, node := range incoming {
		if id := domain.Deref(node.HierarchyId); id != "" {
			incomingIds[id] = true
		}
	}
	for _, hierarchy := range current {
		if !incomingIds[hierarchy.HierarchyId] {
			h.markForDeletion(hierarchy.HierarchyId)
		}
	}
	level := 1
	if parentId != h.rbdSystemId {
		if parent, ok := h.hierarchiesById[parentId]; ok {
			level = parent.Level + 1
		}
	}
	for index, node := range incoming {
		hierarchyId, err := h.processNode(ctx, node, parentId, level, index)
		if err != nil {
			return err
		}
		if node.Components != nil {
			if err := h.processComponents(node.Components, hierarchyId); err != nil {
				return err
			}
		}
		if node.Hierarchy != nil {
			if err := h.processHierarchies(ctx, node.Hierarchy, hierarchyId); err != nil {
				return err
			}
		}
	}
	if parentId == h.rbdSystemId {
		h.createSystemVirtualNodes()
	}
	return h.apply(ctx)
}

func (h *hydration) processNode(ctx context.Context, node domain.TreeInput, parentId string, level, index int) (string, error) {
	posX := firstColumnX + index*columnSpacing
	var existing *domain.Hierarchy
	if id := domain.Deref(node.HierarchyId); id != "" {
		if found, ok := h.hierarchiesById[id]; ok && found.ParentId == parentId {
			existing = found
		}
	}
	hasComponents := len(node.Components) > 0
	hasChildren := len(node.Hierarchy) > 0
	if existing == nil {
		hierarchyId := formatId("H-", h.nextHierarchyCounter)
		h.nextHierarchyCounter++
		formulaCode := domain.Deref(node.FormulaCode)
		if formulaCode == "" {
			formulaCode = nextCodeAvoiding("HS", func(code string) bool { return h.existingHierarchyCodes[code] || h.usedCodes[code] })
		}
		connectionType := "series"
		if node.ConnectionType != nil {
			connectionType = *node.ConnectionType
		}
		created := domain.Hierarchy{
			HierarchyId:    hierarchyId,
			RbdSystemId:    domain.StringPtr(h.rbdSystemId),
			ParentId:       parentId,
			Level:          level,
			SubSystemName:  node.Name,
			ConnectionType: domain.StringPtr(connectionType),
			FormulaCode:    domain.StringPtr(formulaCode),
			RunningHours:   domain.IntPtr(defaultRunning),
			PositionX:      positionNumber(posX),
			PositionY:      positionNumber(hierarchyRowY),
			SourceId:       domain.StringPtr("IN" + hierarchyId),
			TargetId:       domain.StringPtr("OUT" + hierarchyId),
		}
		h.hierarchiesToAdd = append(h.hierarchiesToAdd, created)
		copyOf := created
		h.hierarchiesById[hierarchyId] = &copyOf
		h.hierarchyOrder = append(h.hierarchyOrder, hierarchyId)
		h.usedCodes[formulaCode] = true
		if hasComponents {
			h.createComponentVirtualNodes(hierarchyId, len(node.Components))
		} else if hasChildren {
			h.createHierarchyVirtualNodes(hierarchyId, len(node.Hierarchy))
		}
		return hierarchyId, nil
	}
	tracked, err := h.tx.FindHierarchy(ctx, existing.HierarchyId)
	if err != nil {
		return "", err
	}
	if tracked != nil {
		tracked.SubSystemName = node.Name
		if node.ConnectionType != nil {
			tracked.ConnectionType = node.ConnectionType
		}
		if code := domain.Deref(node.FormulaCode); code != "" {
			tracked.FormulaCode = domain.StringPtr(code)
		}
		tracked.Level = level
		tracked.PositionX = positionNumber(posX)
		tracked.PositionY = positionNumber(hierarchyRowY)
		if err := h.tx.UpdateHierarchy(ctx, *tracked); err != nil {
			return "", err
		}
	}
	if hasComponents {
		h.createComponentVirtualNodes(existing.HierarchyId, len(node.Components))
	} else if hasChildren {
		h.createHierarchyVirtualNodes(existing.HierarchyId, len(node.Hierarchy))
	}
	return existing.HierarchyId, nil
}

func (h *hydration) processComponents(inputs []domain.ComponentInput, parentId string) error {
	current := []domain.SystemComponentProperties{}
	for _, component := range h.componentsByParent[parentId] {
		if domain.Deref(component.ConnectionType) != "virtual" {
			current = append(current, component)
		}
	}
	incomingIds := map[string]bool{}
	for _, input := range inputs {
		if id := domain.Deref(input.SystemComponentId); id != "" {
			incomingIds[id] = true
		}
	}
	for _, component := range current {
		if !incomingIds[component.SystemComponentId] {
			h.componentIdsToDelete = append(h.componentIdsToDelete, component.SystemComponentId)
		}
	}
	for index, input := range inputs {
		var existing *domain.SystemComponentProperties
		for position := range current {
			if current[position].SystemComponentId == domain.Deref(input.SystemComponentId) {
				existing = &current[position]
				break
			}
		}
		posX := firstColumnX + index*columnSpacing
		if existing == nil || domain.Deref(input.SystemComponentId) == "" {
			formulaCode := domain.Deref(input.FormulaCode)
			if formulaCode == "" {
				formulaCode = nextCodeAvoiding("C", func(code string) bool { return h.existingComponentCodes[code] || h.usedCodes[code] })
			}
			name := domain.Deref(input.ComponentName)
			vendor := domain.Deref(input.Vendor)
			master, found := h.masterCache[name+"|"+vendor]
			if !found && vendor == "" {
				master, found = h.masterCache[name+"|"]
				if !found {
					for _, key := range h.masterOrder {
						if strings.HasPrefix(key, name+"|") {
							master, found = h.masterCache[key], true
							break
						}
					}
				}
			}
			if !found {
				vendorInfo := ""
				if vendor != "" {
					vendorInfo = " from vendor '" + vendor + "'"
				}
				return domain.InvalidOperation("Master Component '" + name + "'" + vendorInfo + " not found")
			}
			componentId := formatId("SCP-", h.nextComponentCounter)
			h.nextComponentCounter++
			vendorName := master.ManufacturerName
			if vendor != "" {
				vendorName = domain.StringPtr(vendor)
			}
			connectionType := "serial"
			if input.ConnectionType != nil {
				connectionType = *input.ConnectionType
			}
			now := domain.Now()
			h.componentsToAdd = append(h.componentsToAdd, domain.SystemComponentProperties{
				SystemComponentId: componentId,
				RbdSystemId:       domain.StringPtr(h.rbdSystemId),
				ParentId:          domain.StringPtr(parentId),
				ComponentName:     name,
				Vendor:            vendorName,
				FormulaCode:       domain.StringPtr(formulaCode),
				IdNode:            domain.StringPtr(formulaCode),
				ConnectionType:    domain.StringPtr(connectionType),
				TotalComponent:    domain.IntPtr(positiveOr(input.TotalComponent, 1)),
				ActiveComponent:   domain.IntPtr(positiveOr(input.ActiveComponent, 1)),
				RunningHours:      domain.NumberPtr(domain.NumberFromInt(defaultRunning)),
				FailureRate:       master.FailureRate,
				DistributionType:  domain.StringPtr("exponential"),
				ReliabilityValue:  exponentialAtThousandHours(master.FailureRate),
				Active:            domain.IntPtr(1),
				PositionX:         domain.StringPtr(formatPosition(posX)),
				PositionY:         domain.StringPtr(formatPosition(0)),
				CreatedAt:         &now,
				UpdatedAt:         &now,
				CreatedBy:         domain.StringPtr(h.currentUser),
				UpdatedBy:         domain.StringPtr(h.currentUser),
			})
			h.usedCodes[formulaCode] = true
			continue
		}
		update := domain.SystemComponentProperties{
			SystemComponentId: existing.SystemComponentId,
			ComponentName:     domain.Deref(input.ComponentName),
			TotalComponent:    domain.IntPtr(positiveOr(input.TotalComponent, domain.DerefInt(existing.TotalComponent, 1))),
			ActiveComponent:   domain.IntPtr(positiveOr(input.ActiveComponent, domain.DerefInt(existing.ActiveComponent, 1))),
			ConnectionType:    existing.ConnectionType,
			FormulaCode:       existing.FormulaCode,
		}
		if input.ConnectionType != nil {
			update.ConnectionType = input.ConnectionType
		}
		if input.FormulaCode != nil {
			update.FormulaCode = input.FormulaCode
		}
		h.componentsToUpdate = append(h.componentsToUpdate, update)
	}
	return nil
}

func (h *hydration) createComponentVirtualNodes(hierarchyId string, incomingCount int) {
	count := incomingCount
	if count == 0 {
		count = 1
	}
	for _, kind := range []string{"IN", "OUT"} {
		x := virtualInX
		if kind == "OUT" {
			x = firstColumnX + count*columnSpacing
		}
		code := kind + hierarchyId
		if h.existingComponentCodes[code] {
			for _, component := range h.componentsByParent[hierarchyId] {
				if domain.Deref(component.FormulaCode) == code && domain.Deref(component.ConnectionType) == "virtual" {
					h.componentsToUpdate = append(h.componentsToUpdate, domain.SystemComponentProperties{
						SystemComponentId: component.SystemComponentId,
						PositionX:         domain.StringPtr(formatPosition(x)),
						PositionY:         domain.StringPtr(formatPosition(0)),
						ComponentName:     component.ComponentName,
						TotalComponent:    component.TotalComponent,
						ActiveComponent:   component.ActiveComponent,
						ConnectionType:    component.ConnectionType,
						FormulaCode:       component.FormulaCode,
					})
					break
				}
			}
			continue
		}
		h.componentsToAdd = append(h.componentsToAdd, virtualComponent(formatId("SCP-", h.nextComponentCounter), h.rbdSystemId, hierarchyId, kind, x, h.currentUser))
		h.nextComponentCounter++
	}
}

func (h *hydration) createHierarchyVirtualNodes(hierarchyId string, incomingCount int) {
	count := incomingCount
	if count == 0 {
		count = 1
	}
	h.createVirtualHierarchies(hierarchyId, firstColumnX+count*columnSpacing)
}

func (h *hydration) createSystemVirtualNodes() {
	count := 0
	for _, id := range h.hierarchyOrder {
		hierarchy, ok := h.hierarchiesById[id]
		if ok && hierarchy.ParentId == h.rbdSystemId && hierarchy.Level != 999 {
			count++
		}
	}
	if count == 0 {
		count = 1
	}
	h.createVirtualHierarchies(h.rbdSystemId, firstColumnX+count*columnSpacing)
}

func (h *hydration) createVirtualHierarchies(parentId string, outX int) {
	for _, kind := range []string{"IN", "OUT"} {
		x := virtualInX
		if kind == "OUT" {
			x = outX
		}
		code := kind + parentId
		if h.existingHierarchyCodes[code] {
			for _, id := range h.hierarchyOrder {
				existing, ok := h.hierarchiesById[id]
				if ok && domain.Deref(existing.FormulaCode) == code && existing.Level == 999 {
					h.hierarchiesToMove = append(h.hierarchiesToMove, domain.Hierarchy{HierarchyId: existing.HierarchyId, PositionX: positionNumber(x), PositionY: positionNumber(0)})
					break
				}
			}
			continue
		}
		h.hierarchiesToAdd = append(h.hierarchiesToAdd, virtualHierarchy(formatId("H-", h.nextHierarchyCounter), h.rbdSystemId, parentId, kind, x))
		h.nextHierarchyCounter++
	}
}

func (h *hydration) markForDeletion(hierarchyId string) {
	h.hierarchyIdsToDelete = append(h.hierarchyIdsToDelete, hierarchyId)
	delete(h.hierarchiesById, hierarchyId)
	for _, child := range h.hierarchiesByParent[hierarchyId] {
		h.markForDeletion(child.HierarchyId)
	}
	for _, component := range h.componentsByParent[hierarchyId] {
		h.componentIdsToDelete = append(h.componentIdsToDelete, component.SystemComponentId)
	}
}

func (h *hydration) deleteEverything(ctx context.Context) error {
	for _, id := range h.hierarchyOrder {
		if _, ok := h.hierarchiesById[id]; ok {
			h.hierarchyIdsToDelete = append(h.hierarchyIdsToDelete, id)
		}
	}
	for _, components := range h.componentsByParent {
		for _, component := range components {
			h.componentIdsToDelete = append(h.componentIdsToDelete, component.SystemComponentId)
		}
	}
	return h.apply(ctx)
}

func (h *hydration) apply(ctx context.Context) error {
	if len(h.componentIdsToDelete) > 0 {
		ids := uniqueStrings(h.componentIdsToDelete)
		if err := h.tx.DeleteComponentDependents(ctx, ids); err != nil {
			return err
		}
		if err := h.tx.DeleteEdgesTouching(ctx, ids); err != nil {
			return err
		}
		if err := h.tx.DeleteComponents(ctx, ids); err != nil {
			return err
		}
	}
	if len(h.hierarchyIdsToDelete) > 0 {
		if err := h.tx.DeleteHierarchies(ctx, uniqueStrings(h.hierarchyIdsToDelete)); err != nil {
			return err
		}
	}
	if err := h.tx.InsertHierarchies(ctx, h.hierarchiesToAdd); err != nil {
		return err
	}
	for _, move := range h.hierarchiesToMove {
		existing, err := h.tx.FindHierarchy(ctx, move.HierarchyId)
		if err != nil {
			return err
		}
		if existing != nil {
			existing.PositionX = move.PositionX
			existing.PositionY = move.PositionY
			if err := h.tx.UpdateHierarchy(ctx, *existing); err != nil {
				return err
			}
		}
	}
	if err := h.tx.InsertComponents(ctx, h.componentsToAdd); err != nil {
		return err
	}
	for _, update := range h.componentsToUpdate {
		existing, err := h.tx.FindComponent(ctx, update.SystemComponentId)
		if err != nil {
			return err
		}
		if existing == nil {
			continue
		}
		existing.ComponentName = update.ComponentName
		existing.TotalComponent = update.TotalComponent
		existing.ActiveComponent = update.ActiveComponent
		existing.ConnectionType = update.ConnectionType
		if update.FormulaCode != nil {
			existing.FormulaCode = update.FormulaCode
		}
		if domain.Deref(update.PositionX) != "" {
			existing.PositionX = update.PositionX
		}
		if domain.Deref(update.PositionY) != "" {
			existing.PositionY = update.PositionY
		}
		now := domain.Now()
		existing.UpdatedAt = &now
		if err := h.tx.UpdateComponent(ctx, *existing); err != nil {
			return err
		}
	}
	h.hierarchyIdsToDelete = nil
	h.componentIdsToDelete = nil
	h.hierarchiesToAdd = nil
	h.hierarchiesToMove = nil
	h.componentsToAdd = nil
	h.componentsToUpdate = nil
	return nil
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}
