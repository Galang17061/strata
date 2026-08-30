package rbd

import (
	"context"
	"strconv"
	"strings"

	"github.com/Galang17061/strata-api/internal/domain"
)

const (
	firstColumnX   = 50
	columnSpacing  = 250
	hierarchyRowY  = 5
	virtualInX     = firstColumnX - 150
	defaultRunning = 1000
)

type creation struct {
	tx               *Store
	rbdSystemId      string
	currentUser      string
	maxDepth         int
	hierarchies      []domain.Hierarchy
	components       []domain.SystemComponentProperties
	virtualNodes     []domain.Hierarchy
	virtualParts     []domain.SystemComponentProperties
	hierarchyCounter int
	componentCounter int
}

func (s *SystemService) Create(ctx context.Context, request domain.SystemCreate, currentUser string) (string, error) {
	var newId string
	err := s.store.Transact(ctx, func(tx *Store) error {
		ids, err := tx.SystemIds(ctx)
		if err != nil {
			return err
		}
		newId = nextSequenceId(ids, "RS-")
		now := domain.Now()
		if err := tx.InsertSystem(ctx, domain.RbdSystemDrawing{
			RbdSystemId: newId,
			ProjectId:   domain.Deref(request.ProjectId),
			SystemName:  request.SystemName,
			CreatedAt:   now,
			UpdatedAt:   now,
			CreatedBy:   domain.StringPtr(currentUser),
			UpdatedBy:   domain.StringPtr(currentUser),
		}); err != nil {
			return err
		}
		if len(request.Hierarchy) == 0 {
			return nil
		}
		lastHierarchy, err := tx.LastHierarchyId(ctx)
		if err != nil {
			return err
		}
		lastComponent, err := tx.LastComponentId(ctx)
		if err != nil {
			return err
		}
		depth := 3
		if project, err := tx.FindProject(ctx, domain.Deref(request.ProjectId)); err != nil {
			return err
		} else if project != nil {
			depth = project.HierarchyDepth
		}
		c := &creation{tx: tx, rbdSystemId: newId, currentUser: currentUser, maxDepth: depth, hierarchyCounter: counterAfter(lastHierarchy, "H-"), componentCounter: counterAfter(lastComponent, "SCP-")}
		if err := c.processNodes(ctx, request.Hierarchy, nil, 1); err != nil {
			return err
		}
		c.addSystemVirtualNodes(len(request.Hierarchy))
		if err := tx.InsertHierarchies(ctx, c.hierarchies); err != nil {
			return err
		}
		if err := tx.InsertComponents(ctx, c.components); err != nil {
			return err
		}
		if err := tx.InsertComponents(ctx, c.virtualParts); err != nil {
			return err
		}
		if err := tx.InsertHierarchies(ctx, c.virtualNodes); err != nil {
			return err
		}
		for index := range c.hierarchies {
			hierarchy := &c.hierarchies[index]
			owned := []domain.SystemComponentProperties{}
			for _, component := range c.components {
				if domain.Deref(component.ParentId) == hierarchy.HierarchyId {
					owned = append(owned, component)
				}
			}
			children := []domain.Hierarchy{}
			for _, candidate := range c.hierarchies {
				if candidate.ParentId == hierarchy.HierarchyId {
					children = append(children, candidate)
				}
			}
			if len(owned) > 0 {
				hierarchy.Formula = componentFormula(owned, hierarchy.ConnectionType)
			} else if len(children) > 0 {
				hierarchy.Formula = hierarchyFormula(children, hierarchy.ConnectionType)
			}
			if err := tx.UpdateHierarchy(ctx, *hierarchy); err != nil {
				return err
			}
		}
		return nil
	})
	return newId, err
}

func (c *creation) processNodes(ctx context.Context, nodes []domain.TreeInput, parentId *string, level int) error {
	if level > c.maxDepth {
		return domain.InvalidOperation("Maximum hierarchy level is " + strconv.Itoa(c.maxDepth))
	}
	usedCodes := map[string]bool{}
	for index, node := range nodes {
		if strings.TrimSpace(domain.Deref(node.Name)) == "" {
			return domain.InvalidOperation("Node name is required")
		}
		hierarchyId := formatId("H-", c.hierarchyCounter)
		c.hierarchyCounter++
		formulaCode := domain.Deref(node.FormulaCode)
		if formulaCode == "" {
			existing, err := c.tx.HierarchyCodesOfSystem(ctx, c.rbdSystemId, "HS")
			if err != nil {
				return err
			}
			formulaCode = nextCodeWithPrefix("HS", existing)
		}
		duplicate, err := c.codeTaken(ctx, formulaCode)
		if err != nil {
			return err
		}
		if duplicate {
			return domain.InvalidOperation("Formula code '" + formulaCode + "' already exists in this parent")
		}
		connectionType := "series"
		if strings.TrimSpace(domain.Deref(node.ConnectionType)) != "" {
			connectionType = strings.ToLower(*node.ConnectionType)
		}
		owner := domain.Deref(parentId)
		if parentId == nil {
			owner = c.rbdSystemId
		}
		c.hierarchies = append(c.hierarchies, domain.Hierarchy{
			HierarchyId:    hierarchyId,
			RbdSystemId:    domain.StringPtr(c.rbdSystemId),
			ParentId:       owner,
			Level:          level,
			SubSystemName:  node.Name,
			ConnectionType: domain.StringPtr(connectionType),
			FormulaCode:    domain.StringPtr(formulaCode),
			PositionX:      positionNumber(firstColumnX + index*columnSpacing),
			PositionY:      positionNumber(hierarchyRowY),
			SourceId:       domain.StringPtr("IN" + hierarchyId),
			TargetId:       domain.StringPtr("OUT" + hierarchyId),
		})
		hasComponents := len(node.Components) > 0
		hasChildren := len(node.Hierarchy) > 0
		if hasComponents {
			if err := c.processComponents(ctx, node.Components, hierarchyId, usedCodes); err != nil {
				return err
			}
		}
		if hasChildren {
			if err := c.processNodes(ctx, node.Hierarchy, domain.StringPtr(hierarchyId), level+1); err != nil {
				return err
			}
		}
		outX := firstColumnX + columnSpacing
		if hasComponents {
			outX = firstColumnX + len(node.Components)*columnSpacing
		} else if hasChildren {
			outX = firstColumnX + len(node.Hierarchy)*columnSpacing
		}
		if hasComponents {
			c.addComponentVirtualNodes(hierarchyId, outX)
		} else {
			c.addHierarchyVirtualNodes(hierarchyId, outX)
		}
	}
	return nil
}

func (c *creation) codeTaken(ctx context.Context, code string) (bool, error) {
	if strings.HasPrefix(code, "HS") {
		return c.tx.HierarchyCodeExists(ctx, c.rbdSystemId, code)
	}
	if strings.HasPrefix(code, "C") {
		return c.tx.ComponentCodeExists(ctx, c.rbdSystemId, code)
	}
	return false, nil
}

func (c *creation) processComponents(ctx context.Context, inputs []domain.ComponentInput, parentId string, usedCodes map[string]bool) error {
	cache := map[string]*domain.MasterComponentWithVendor{}
	created := 0
	for _, input := range inputs {
		formulaCode := domain.Deref(input.FormulaCode)
		if formulaCode == "" {
			for {
				existing, err := c.tx.ComponentCodesStartingWith(ctx, "C")
				if err != nil {
					return err
				}
				formulaCode = nextCodeWithPrefix("C", existing)
				taken, err := c.codeTaken(ctx, formulaCode)
				if err != nil {
					return err
				}
				if !usedCodes[formulaCode] && !taken {
					break
				}
			}
		}
		duplicate, err := c.codeTaken(ctx, formulaCode)
		if err != nil {
			return err
		}
		if duplicate {
			return domain.InvalidOperation("Formula code '" + formulaCode + "' already exists in this RBD system")
		}
		usedCodes[formulaCode] = true
		name := domain.Deref(input.ComponentName)
		vendor := domain.Deref(input.Vendor)
		cacheKey := name + "|" + vendor
		master, cached := cache[cacheKey]
		if !cached {
			var vendorFilter *string
			if vendor != "" {
				vendorFilter = domain.StringPtr(vendor)
			}
			master, err = c.tx.MasterComponentByNameAndVendor(ctx, name, vendorFilter)
			if err != nil {
				return err
			}
			if master == nil {
				vendorInfo := ""
				if vendor != "" {
					vendorInfo = " from vendor '" + vendor + "'"
				}
				return domain.InvalidOperation("Master Component '" + name + "'" + vendorInfo + " not found")
			}
			cache[cacheKey] = master
		}
		componentId := formatId("SCP-", c.componentCounter)
		c.componentCounter++
		vendorName := master.ManufacturerName
		if vendor != "" {
			vendorName = domain.StringPtr(vendor)
		}
		connectionType := "serial"
		if input.ConnectionType != nil {
			connectionType = *input.ConnectionType
		}
		now := domain.Now()
		c.components = append(c.components, domain.SystemComponentProperties{
			SystemComponentId: componentId,
			RbdSystemId:       domain.StringPtr(c.rbdSystemId),
			ParentId:          domain.StringPtr(parentId),
			ComponentName:     name,
			Vendor:            vendorName,
			FormulaCode:       domain.StringPtr(formulaCode),
			IdNode:            domain.StringPtr(componentId),
			ConnectionType:    domain.StringPtr(connectionType),
			TotalComponent:    domain.IntPtr(positiveOr(input.TotalComponent, 1)),
			ActiveComponent:   domain.IntPtr(positiveOr(input.ActiveComponent, 1)),
			RunningHours:      domain.NumberPtr(domain.NumberFromInt(defaultRunning)),
			FailureRate:       master.FailureRate,
			DistributionType:  domain.StringPtr("exponential"),
			ReliabilityValue:  exponentialAtThousandHours(master.FailureRate),
			Active:            domain.IntPtr(1),
			PositionX:         domain.StringPtr(formatPosition(firstColumnX + created*columnSpacing)),
			PositionY:         domain.StringPtr(formatPosition(0)),
			CreatedAt:         &now,
			UpdatedAt:         &now,
			CreatedBy:         domain.StringPtr(c.currentUser),
			UpdatedBy:         domain.StringPtr(c.currentUser),
		})
		created++
	}
	return nil
}

func positiveOr(value *int, fallback int) int {
	if value != nil && *value > 0 {
		return *value
	}
	return fallback
}

func (c *creation) addComponentVirtualNodes(hierarchyId string, outX int) {
	for _, kind := range []string{"IN", "OUT"} {
		x := virtualInX
		if kind == "OUT" {
			x = outX
		}
		c.virtualParts = append(c.virtualParts, virtualComponent(formatId("SCP-", c.componentCounter), c.rbdSystemId, hierarchyId, kind, x, c.currentUser))
		c.componentCounter++
	}
}

func (c *creation) addHierarchyVirtualNodes(hierarchyId string, outX int) {
	for _, kind := range []string{"IN", "OUT"} {
		x := virtualInX
		if kind == "OUT" {
			x = outX
		}
		c.virtualNodes = append(c.virtualNodes, virtualHierarchy(formatId("H-", c.hierarchyCounter), c.rbdSystemId, hierarchyId, kind, x))
		c.hierarchyCounter++
	}
}

func (c *creation) addSystemVirtualNodes(topLevelCount int) {
	c.addHierarchyVirtualNodes(c.rbdSystemId, firstColumnX+topLevelCount*columnSpacing)
}

func virtualComponent(id, rbdSystemId, parentId, kind string, x int, currentUser string) domain.SystemComponentProperties {
	now := domain.Now()
	code := kind + parentId
	return domain.SystemComponentProperties{
		SystemComponentId: id,
		RbdSystemId:       domain.StringPtr(rbdSystemId),
		ParentId:          domain.StringPtr(parentId),
		ComponentName:     kind + " Virtual Node",
		FormulaCode:       domain.StringPtr(code),
		IdNode:            domain.StringPtr(code),
		ConnectionType:    domain.StringPtr("virtual"),
		PositionX:         domain.StringPtr(formatPosition(x)),
		PositionY:         domain.StringPtr(formatPosition(0)),
		Active:            domain.IntPtr(1),
		TotalComponent:    domain.IntPtr(1),
		ActiveComponent:   domain.IntPtr(1),
		CreatedAt:         &now,
		UpdatedAt:         &now,
		CreatedBy:         domain.StringPtr(currentUser),
		UpdatedBy:         domain.StringPtr(currentUser),
	}
}

func virtualHierarchy(id, rbdSystemId, parentId, kind string, x int) domain.Hierarchy {
	return domain.Hierarchy{
		HierarchyId:    id,
		RbdSystemId:    domain.StringPtr(rbdSystemId),
		ParentId:       parentId,
		Level:          999,
		SubSystemName:  domain.StringPtr(kind + " Virtual Node"),
		FormulaCode:    domain.StringPtr(kind + parentId),
		ConnectionType: domain.StringPtr("virtual"),
		PositionX:      positionNumber(x),
		PositionY:      positionNumber(0),
	}
}
