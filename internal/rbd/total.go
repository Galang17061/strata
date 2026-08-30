package rbd

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/reliability"
	"github.com/Galang17061/strata-api/internal/web"
)

type TotalService struct {
	store *Store
}

func NewTotalService(store *Store) *TotalService {
	return &TotalService{store: store}
}

func (s *TotalService) HierarchyLevel(ctx context.Context, rbdSystemId string) (string, error) {
	levels, err := s.store.HierarchyLevelsOfSystem(ctx, rbdSystemId)
	if err != nil {
		return "", err
	}
	if len(levels) == 0 {
		return "COMPONENT", nil
	}
	switch levels[0] {
	case 1:
		return "LEVEL1", nil
	case 2:
		return "LEVEL2", nil
	case 3:
		return "LEVEL3", nil
	default:
		return "", nil
	}
}

func (s *TotalService) UpdateFormula(ctx context.Context, rbdSystemId, formula string) (string, error) {
	system, err := s.store.FindSystem(ctx, rbdSystemId)
	if err != nil {
		return "", err
	}
	if system == nil {
		return "", errors.New("Object reference not set to an instance of an object.")
	}
	system.Formula = domain.StringPtr(formula)
	system.UpdatedAt = domain.Now()
	if err := s.store.UpdateSystem(ctx, *system); err != nil {
		return "", err
	}
	return formula, nil
}

func componentReliability(component domain.SystemComponentProperties) decimal.Decimal {
	if component.RunningHours == nil || component.RunningHours.Decimal.Sign() <= 0 {
		return decimal.Zero
	}
	distribution := strings.TrimSpace(strings.ToLower(domain.Deref(component.DistributionType)))
	weibull := func() decimal.Decimal {
		eta := reliability.ToFloat(component.ScaleParameter.Decimal)
		beta := reliability.ToFloat(component.ShapeParameter.Decimal)
		if eta <= 0 || beta <= 0 {
			return decimal.Zero
		}
		value, err := reliability.WeibullFromFloats(component.RunningHours.Decimal, component.ScaleParameter.Decimal, component.ShapeParameter.Decimal)
		if err != nil {
			return decimal.Zero
		}
		return value
	}
	exponential := func() decimal.Decimal {
		return reliability.ExponentialFromFloats(component.FailureRate.Decimal, component.RunningHours.Decimal)
	}
	switch distribution {
	case "weibull":
		if component.ScaleParameter == nil || component.ShapeParameter == nil {
			return decimal.Zero
		}
		return weibull()
	case "exponential":
		if component.FailureRate == nil || component.FailureRate.Decimal.Sign() <= 0 {
			return decimal.Zero
		}
		return exponential()
	default:
		if component.ScaleParameter != nil && component.ShapeParameter != nil {
			return weibull()
		}
		if component.FailureRate != nil && component.FailureRate.Decimal.Sign() > 0 {
			return exponential()
		}
		return decimal.Zero
	}
}

func adjustedReliability(component domain.SystemComponentProperties, base decimal.Decimal) (decimal.Decimal, error) {
	connectionType := strings.TrimSpace(strings.ToLower(domain.Deref(component.ConnectionType)))
	if connectionType == "" {
		connectionType = "series"
	}
	active := domain.DerefInt(component.ActiveComponent, 1)
	total := domain.DerefInt(component.TotalComponent, 1)
	switch {
	case connectionType == "series" || connectionType == "serial":
		return reliability.SeriesOfIdentical(base, total), nil
	case connectionType == "parallel":
		if active == 1 {
			return reliability.ParallelOfIdentical(base, total), nil
		}
		return reliability.KOutOfN(base, active, total)
	case connectionType == "redundantsi parsial" || connectionType == "partial" || strings.Contains(connectionType, "parsial"):
		if active > 1 && total >= active && active != total {
			return reliability.KOutOfN(base, active, total)
		}
		return base, nil
	default:
		return base, nil
	}
}

func historyAdjustedReliability(component domain.SystemComponentProperties, base decimal.Decimal) (decimal.Decimal, error) {
	connectionType := strings.TrimSpace(strings.ToLower(domain.Deref(component.ConnectionType)))
	if connectionType == "" {
		connectionType = "series"
	}
	active := domain.DerefInt(component.ActiveComponent, 1)
	total := domain.DerefInt(component.TotalComponent, 1)
	switch {
	case connectionType == "series" || connectionType == "serial":
		return reliability.SeriesOfIdentical(base, total), nil
	case connectionType == "parallel":
		if active == 1 && total > 1 {
			return reliability.ParallelOfIdentical(base, total), nil
		}
		if active > 1 && total >= active {
			return reliability.KOutOfN(base, active, total)
		}
		return base, nil
	case connectionType == "redundantsi parsial" || connectionType == "partial" || strings.Contains(connectionType, "parsial"):
		if active > 1 && total >= active && active != total {
			return reliability.KOutOfN(base, active, total)
		}
		return base, nil
	default:
		return base, nil
	}
}

type lookup struct {
	codes  []string
	values map[string]*decimal.Decimal
}

func newLookup() *lookup {
	return &lookup{values: map[string]*decimal.Decimal{}}
}

func (l *lookup) set(code string, value *decimal.Decimal) {
	if _, exists := l.values[code]; !exists {
		l.codes = append(l.codes, code)
	}
	l.values[code] = value
}

func (l *lookup) present() map[string]decimal.Decimal {
	out := map[string]decimal.Decimal{}
	for code, value := range l.values {
		if value != nil {
			out[code] = *value
		}
	}
	return out
}

func (l *lookup) ordered() *domain.OrderedMap {
	out := domain.NewOrderedMap()
	for _, code := range l.codes {
		if value := l.values[code]; value != nil {
			out.Set(code, domain.NewNumber(*value))
		} else {
			out.Set(code, nil)
		}
	}
	return out
}

func (l *lookup) evaluate(formula string, balance bool) (decimal.Decimal, error) {
	replaced := replaceCodes(formula, l.codes, l.present())
	result, err := reliability.Evaluate(reliability.PrepareFormula(replaced, balance))
	if err != nil {
		return decimal.Zero, errors.New("Failed to evaluate the formula: " + reliability.PrepareFormula(replaced, balance) + ". Original error: " + err.Error())
	}
	return result, nil
}

func (s *TotalService) CalculateHierarchy(ctx context.Context, hierarchyId string) web.Envelope {
	hierarchy, err := s.store.FindHierarchy(ctx, hierarchyId)
	if err != nil {
		return web.Failed(500, "Failed to calculate hierarchy reliability: "+err.Error(), nil)
	}
	if hierarchy == nil {
		return web.NotFound("Hierarchy with ID " + hierarchyId + " not found")
	}
	result, err := s.calculateRecursive(ctx, *hierarchy, map[string]bool{})
	if err != nil {
		var invalid domain.InvalidOperationError
		if errors.As(err, &invalid) {
			return web.Failed(400, err.Error(), nil)
		}
		return web.Failed(500, "Failed to calculate hierarchy reliability: "+err.Error(), nil)
	}
	return web.Success(result, "Hierarchy reliability calculated successfully")
}

func (s *TotalService) calculateRecursive(ctx context.Context, hierarchy domain.Hierarchy, visited map[string]bool) (*domain.HierarchyCalculation, error) {
	if visited[hierarchy.HierarchyId] {
		return nil, domain.InvalidOperation("Circular reference detected in hierarchy " + hierarchy.HierarchyId)
	}
	visited[hierarchy.HierarchyId] = true
	children, err := s.store.ChildrenExcludingSelf(ctx, hierarchy.HierarchyId)
	if err != nil {
		return nil, err
	}
	childResults := []domain.HierarchyCalculation{}
	for _, child := range children {
		childResult, err := s.calculateRecursive(ctx, child, visited)
		if err != nil {
			return nil, err
		}
		childResults = append(childResults, *childResult)
	}
	values := newLookup()
	if len(children) > 0 {
		for _, childResult := range childResults {
			if domain.Deref(childResult.FormulaCode) != "" && childResult.CalculatedReliability != nil {
				value := childResult.CalculatedReliability.Decimal
				values.set(*childResult.FormulaCode, &value)
			}
		}
	} else {
		components, err := s.store.ComponentsOfParent(ctx, hierarchy.HierarchyId, true)
		if err != nil {
			return nil, err
		}
		for index := range components {
			component := &components[index]
			code := domain.Deref(component.FormulaCode)
			if code == "" {
				continue
			}
			var base decimal.Decimal
			calculated := false
			if component.ReliabilityValue != nil && component.ReliabilityValue.Decimal.Sign() > 0 {
				base = component.ReliabilityValue.Decimal
				calculated = true
			}
			if !calculated && component.RunningHours != nil {
				base = componentReliability(*component)
				calculated = true
				component.ReliabilityValue = domain.NumberPtr(domain.NewNumber(base))
				component.UpdatedBy = domain.StringPtr(s.currentUser(ctx))
				now := domain.Now()
				component.UpdatedAt = &now
				if err := s.store.UpdateComponent(ctx, *component); err != nil {
					return nil, err
				}
			}
			if !calculated {
				continue
			}
			final, err := adjustedReliability(*component, base)
			if err != nil {
				return nil, err
			}
			values.set(code, &final)
		}
		if strings.Contains(domain.Deref(hierarchy.Formula), "HS-") {
			codes := []string{}
			for _, component := range components {
				if code := domain.Deref(component.FormulaCode); code != "" {
					codes = append(codes, code)
				}
			}
			if len(codes) > 0 {
				hierarchy.Formula = domain.StringPtr(strings.Join(codes, "*"))
				if err := s.store.UpdateHierarchy(ctx, hierarchy); err != nil {
					return nil, err
				}
			}
		}
	}
	var current *domain.Number
	if domain.Deref(hierarchy.Formula) != "" && len(values.codes) > 0 {
		result, err := values.evaluate(*hierarchy.Formula, true)
		if err != nil {
			return nil, domain.InvalidOperation("Failed to calculate reliability for hierarchy " + hierarchy.HierarchyId + ": " + err.Error())
		}
		current = domain.NumberPtr(domain.NewNumber(result))
		hierarchy.RealibilityValue = current
		if err := s.store.UpdateHierarchy(ctx, hierarchy); err != nil {
			return nil, err
		}
	}
	if hierarchy.Level == 1 && current != nil {
		system, err := s.store.FindSystem(ctx, domain.Deref(hierarchy.RbdSystemId))
		if err != nil {
			return nil, err
		}
		if system != nil {
			system.ReliabilityTotal = current
			system.UpdatedBy = domain.StringPtr(s.currentUser(ctx))
			system.UpdatedAt = domain.Now()
			if err := s.store.UpdateSystem(ctx, *system); err != nil {
				return nil, err
			}
		}
	}
	return &domain.HierarchyCalculation{
		HierarchyId:           hierarchy.HierarchyId,
		HierarchyName:         hierarchy.SubSystemName,
		Level:                 hierarchy.Level,
		FormulaCode:           hierarchy.FormulaCode,
		Formula:               hierarchy.Formula,
		CalculatedReliability: current,
		ReliabilityLookup:     values.ordered(),
		Children:              childResults,
	}, nil
}

type userKey struct{}

func WithUser(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, userKey{}, user)
}

func (s *TotalService) currentUser(ctx context.Context) string {
	if user, ok := ctx.Value(userKey{}).(string); ok {
		return user
	}
	return "Unknown"
}

func (s *TotalService) SystemTotal(ctx context.Context, rbdSystemId string) web.Envelope {
	system, err := s.store.FindSystem(ctx, rbdSystemId)
	if err != nil {
		return web.Failed(500, "Failed to get hierarchy reliability total: "+err.Error(), nil)
	}
	if system == nil {
		return web.NotFound("RBD System " + rbdSystemId + " not found")
	}
	roots, err := s.store.RootHierarchies(ctx, rbdSystemId)
	if err != nil {
		return web.Failed(500, "Failed to get hierarchy reliability total: "+err.Error(), nil)
	}
	if len(roots) == 0 {
		components, err := s.store.ActiveComponentsWithValue(ctx, rbdSystemId)
		if err != nil {
			return web.Failed(500, "Failed to get hierarchy reliability total: "+err.Error(), nil)
		}
		if len(components) == 0 {
			return web.Success(domain.SystemReliabilityTotal{RbdSystemId: rbdSystemId, SystemName: system.SystemName, Formula: system.Formula, Level: "COMPONENT", HierarchyLookup: domain.NewOrderedMap(), ComponentLookup: domain.NewOrderedMap()}, "No components found")
		}
		componentLookup := domain.NewOrderedMap()
		for _, component := range components {
			code := domain.Deref(component.FormulaCode)
			if code == "" {
				continue
			}
			if componentLookup.Has(code) {
				return web.Failed(500, "Failed to get hierarchy reliability total: An item with the same key has already been added. Key: "+code, nil)
			}
			componentLookup.Set(code, *component.ReliabilityValue)
		}
		return web.Success(domain.SystemReliabilityTotal{RbdSystemId: rbdSystemId, SystemName: system.SystemName, ReliabilityTotal: system.ReliabilityTotal, Formula: system.Formula, Level: "COMPONENT", HierarchyLookup: domain.NewOrderedMap(), ComponentLookup: componentLookup}, "Component level reliability lookup")
	}
	hierarchyLookup := domain.NewOrderedMap()
	componentLookup := domain.NewOrderedMap()
	visited := map[string]bool{}
	for _, root := range roots {
		if err := s.buildLookup(ctx, root, hierarchyLookup, componentLookup, visited); err != nil {
			return web.Failed(500, "Failed to get hierarchy reliability total: "+err.Error(), nil)
		}
	}
	return web.Success(domain.SystemReliabilityTotal{
		RbdSystemId:      rbdSystemId,
		SystemName:       system.SystemName,
		ReliabilityTotal: system.ReliabilityTotal,
		Formula:          system.Formula,
		Level:            roots[0].Level,
		HierarchyLookup:  hierarchyLookup,
		ComponentLookup:  componentLookup,
	}, "Hierarchy reliability lookup calculated successfully")
}

func (s *TotalService) buildLookup(ctx context.Context, hierarchy domain.Hierarchy, hierarchyLookup, componentLookup *domain.OrderedMap, visited map[string]bool) error {
	if visited[hierarchy.HierarchyId] {
		return nil
	}
	visited[hierarchy.HierarchyId] = true
	children, err := s.store.ChildrenExcludingSelf(ctx, hierarchy.HierarchyId)
	if err != nil {
		return err
	}
	for _, child := range children {
		if err := s.buildLookup(ctx, child, hierarchyLookup, componentLookup, visited); err != nil {
			return err
		}
	}
	if len(children) == 0 {
		components, err := s.store.ActiveComponentsOfParentWithValue(ctx, hierarchy.HierarchyId)
		if err != nil {
			return err
		}
		for _, component := range components {
			if code := domain.Deref(component.FormulaCode); code != "" {
				componentLookup.Set(code, *component.ReliabilityValue)
			}
		}
	}
	if domain.Deref(hierarchy.Formula) == "" {
		return nil
	}
	values := newLookup()
	if len(children) > 0 {
		for _, child := range children {
			if code := domain.Deref(child.FormulaCode); code != "" && child.RealibilityValue != nil {
				value := child.RealibilityValue.Decimal
				values.set(code, &value)
			}
		}
	} else {
		components, err := s.store.ComponentsOfParent(ctx, hierarchy.HierarchyId, true)
		if err != nil {
			return err
		}
		for _, component := range components {
			if code := domain.Deref(component.FormulaCode); code != "" && component.ReliabilityValue != nil {
				value := component.ReliabilityValue.Decimal
				values.set(code, &value)
			}
		}
	}
	if len(values.codes) == 0 {
		return nil
	}
	result, err := values.evaluate(*hierarchy.Formula, true)
	if err != nil {
		return nil
	}
	hierarchy.RealibilityValue = domain.NumberPtr(domain.NewNumber(result))
	if err := s.store.UpdateHierarchy(ctx, hierarchy); err != nil {
		return nil
	}
	if code := domain.Deref(hierarchy.FormulaCode); code != "" {
		hierarchyLookup.Set(code, domain.NewNumber(result))
	}
	return nil
}

func (s *TotalService) SaveHistoryAfterEdgeSave(ctx context.Context, hierarchyId, currentUser string) {
	hierarchy, err := s.store.FindHierarchy(ctx, hierarchyId)
	if err != nil || hierarchy == nil {
		return
	}
	components, err := s.store.ComponentsOfParent(ctx, hierarchyId, true)
	if err != nil {
		return
	}
	children, err := s.store.ChildHierarchies(ctx, hierarchyId)
	if err != nil {
		return
	}
	values := newLookup()
	details := []any{}
	for _, component := range components {
		code := domain.Deref(component.FormulaCode)
		if code == "" || reliability.IsVirtualCode(code) {
			continue
		}
		var base, adjusted *decimal.Decimal
		if component.ReliabilityValue != nil {
			value := component.ReliabilityValue.Decimal
			base = &value
			adjusted = &value
			if value.Sign() > 0 {
				result, err := historyAdjustedReliability(component, value)
				if err != nil {
					return
				}
				adjusted = &result
			}
		}
		values.set(code, adjusted)
		details = append(details, domain.ComponentHistoryDetail{
			FormulaCode:         component.FormulaCode,
			ComponentName:       component.ComponentName,
			Vendor:              component.Vendor,
			Type:                "component",
			ConnectionType:      component.ConnectionType,
			ActiveComponent:     component.ActiveComponent,
			TotalComponent:      component.TotalComponent,
			BaseReliability:     numberOf(base),
			AdjustedReliability: numberOf(adjusted),
			FailureRate:         component.FailureRate,
		})
	}
	for _, child := range children {
		code := domain.Deref(child.FormulaCode)
		if code == "" || reliability.IsVirtualCode(code) {
			continue
		}
		var value *decimal.Decimal
		if child.RealibilityValue != nil {
			copyOf := child.RealibilityValue.Decimal
			value = &copyOf
		}
		values.set(code, value)
		details = append(details, domain.HierarchyHistoryDetail{FormulaCode: child.FormulaCode, ComponentName: child.SubSystemName, Type: "hierarchy"})
	}
	cleaned := reliability.StripSystemVirtualCodes(domain.Deref(hierarchy.Formula))
	calculated := hierarchy.RealibilityValue
	if cleaned != "" && len(values.codes) > 0 {
		if result, err := values.evaluate(cleaned, true); err == nil {
			calculated = domain.NumberPtr(domain.NewNumber(result))
			hierarchy.RealibilityValue = calculated
			if err := s.store.UpdateHierarchy(ctx, *hierarchy); err != nil {
				return
			}
		}
	}
	history := domain.ReliabilityHistory{
		HistoryId:             domain.NewGuid().String(),
		RbdSystemId:           domain.Deref(hierarchy.RbdSystemId),
		HierarchyId:           hierarchy.HierarchyId,
		HierarchyName:         hierarchy.SubSystemName,
		HierarchyLevel:        domain.IntPtr(hierarchy.Level),
		FormulaCode:           hierarchy.FormulaCode,
		Formula:               domain.StringPtr(cleaned),
		CalculatedReliability: calculated,
		RunningHours:          domain.NumberPtr(domain.NumberFromInt(int64(domain.DerefInt(hierarchy.RunningHours, 0)))),
		CalculationTimestamp:  domain.Now(),
		CalculatedBy:          domain.StringPtr(currentUser),
	}
	if len(values.codes) > 0 {
		if encoded, err := json.Marshal(values.ordered()); err == nil {
			history.ReliabilityLookup = domain.StringPtr(string(encoded))
		}
	}
	if len(details) > 0 {
		if encoded, err := json.Marshal(details); err == nil {
			history.ComponentDetails = domain.StringPtr(string(encoded))
		}
	}
	_ = s.store.InsertHistory(ctx, history)
}

func numberOf(value *decimal.Decimal) *domain.Number {
	if value == nil {
		return nil
	}
	return domain.NumberPtr(domain.NewNumber(*value))
}

func (s *TotalService) HistoryPage(ctx context.Context, hierarchyId string, page, pageSize int, sortBy, sortOrder string) (web.Envelope, error) {
	total, err := s.store.HistoryCount(ctx, hierarchyId)
	if err != nil {
		return web.Envelope{}, err
	}
	if total == 0 {
		return web.SuccessWithMeta([]domain.HistoryView{}, "No history found for this hierarchy", web.NewMeta(0, 0, page, pageSize)), nil
	}
	rows, err := s.store.HistoryPage(ctx, hierarchyId, page, pageSize, sortBy, sortOrder)
	if err != nil {
		return web.Envelope{}, err
	}
	views := make([]domain.HistoryView, 0, len(rows))
	for _, row := range rows {
		lookupJSON, err := rawJSON(row.ReliabilityLookup)
		if err != nil {
			return web.Envelope{}, err
		}
		detailsJSON, err := rawJSON(row.ComponentDetails)
		if err != nil {
			return web.Envelope{}, err
		}
		views = append(views, domain.HistoryView{
			HistoryId:             row.HistoryId,
			RbdSystemId:           row.RbdSystemId,
			HierarchyId:           row.HierarchyId,
			HierarchyName:         row.HierarchyName,
			HierarchyLevel:        row.HierarchyLevel,
			FormulaCode:           row.FormulaCode,
			Formula:               row.Formula,
			CalculatedReliability: row.CalculatedReliability,
			ReliabilityLookup:     lookupJSON,
			ComponentDetails:      detailsJSON,
			RunningHours:          row.RunningHours,
			CalculationTimestamp:  row.CalculationTimestamp,
			CalculatedBy:          row.CalculatedBy,
		})
	}
	meta := web.NewMeta(total, web.TotalPages(total, pageSize), page, pageSize)
	return web.SuccessWithMeta(views, "Found "+web.Itoa(total)+" history records", meta), nil
}

func rawJSON(text *string) (json.RawMessage, error) {
	if text == nil || *text == "" {
		return json.RawMessage("null"), nil
	}
	if !json.Valid([]byte(*text)) {
		return nil, errors.New("The stored value is not valid JSON.")
	}
	return json.RawMessage(*text), nil
}

func (s *TotalService) DeleteHistory(ctx context.Context, historyId string) error {
	if strings.TrimSpace(historyId) == "" {
		return domain.Argument("History ID is required (Parameter 'historyId')")
	}
	history, err := s.store.FindHistory(ctx, historyId)
	if err != nil {
		return err
	}
	if history == nil {
		return domain.KeyNotFound("Reliability history with ID '" + historyId + "' not found")
	}
	return s.store.DeleteHistory(ctx, historyId)
}
