package rbd

import (
	"context"
	"math"
	"strings"

	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/reliability"
)

type ComponentService struct {
	store *Store
}

func NewComponentService(store *Store) *ComponentService {
	return &ComponentService{store: store}
}

func (s *ComponentService) ScpList(ctx context.Context, search, sortBy, sortOrder string) ([]domain.ScpListItem, error) {
	rows, err := s.store.ScpList(ctx, search)
	if err != nil {
		return nil, err
	}
	compare := func(a, b domain.ScpListItem) int {
		return domain.CompareText(domain.Deref(a.ComponentName), domain.Deref(b.ComponentName))
	}
	if sortBy != "" && sortOrder != "" {
		switch strings.ToLower(sortBy) {
		case "totalcomponent":
			compare = func(a, b domain.ScpListItem) int { return compareOptionalInt(a.TotalComponent, b.TotalComponent) }
		case "totalactive":
			compare = func(a, b domain.ScpListItem) int { return compareOptionalInt(a.TotalActive, b.TotalActive) }
		}
		domain.OrderBy(rows, strings.ToLower(sortOrder) == "desc", compare)
	} else {
		domain.OrderBy(rows, false, compare)
	}
	return rows, nil
}

func compareOptionalInt(a, b *int) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return -1
	case b == nil:
		return 1
	default:
		return domain.CompareInt64(int64(*a), int64(*b))
	}
}

func (s *ComponentService) Find(ctx context.Context, systemComponentId string) (*domain.SystemComponentProperties, error) {
	return s.store.FindComponent(ctx, systemComponentId)
}

func (s *ComponentService) Detail(ctx context.Context, systemComponentId string) (*domain.ComponentDetail, error) {
	component, err := s.store.FindComponent(ctx, systemComponentId)
	if err != nil || component == nil {
		return nil, err
	}
	detail := domain.ComponentDetailOf(*component)
	return &detail, nil
}

func (s *ComponentService) CreateSimple(ctx context.Context, request domain.ComponentSimpleCreate, currentUser string) (*domain.ComponentDetail, error) {
	parentId := domain.Deref(request.ParentId)
	rbdSystemId := parentId
	hierarchy, err := s.store.FindHierarchy(ctx, parentId)
	if err != nil {
		return nil, err
	}
	if hierarchy != nil {
		rbdSystemId = domain.Deref(hierarchy.RbdSystemId)
	}
	name := domain.Deref(request.ComponentName)
	master, err := s.store.MasterComponentByNameAndVendor(ctx, name, request.Vendor)
	if err != nil {
		return nil, err
	}
	if master == nil {
		vendorInfo := ""
		if request.Vendor != nil {
			vendorInfo = " from vendor '" + *request.Vendor + "'"
		}
		return nil, domain.InvalidOperation("Master Component '" + name + "'" + vendorInfo + " not found")
	}
	ids, err := s.store.ComponentIds(ctx)
	if err != nil {
		return nil, err
	}
	maxNumber := 0
	for _, id := range ids {
		if number, ok := parseAfter(id, "SCP-"); ok && number > maxNumber {
			maxNumber = number
		}
	}
	existingCodes, err := s.store.ComponentCodesStartingWith(ctx, "C")
	if err != nil {
		return nil, err
	}
	vendor := request.Vendor
	if vendor == nil {
		vendor = master.ManufacturerName
	}
	now := domain.Now()
	component := domain.SystemComponentProperties{
		SystemComponentId:  formatId("SCP-", maxNumber+1),
		RbdSystemId:        domain.StringPtr(rbdSystemId),
		ParentId:           domain.StringPtr(parentId),
		ComponentName:      name,
		ComponentTagNumber: request.ComponentTagNumber,
		Vendor:             vendor,
		FormulaCode:        domain.StringPtr(nextCodeWithPrefix("C", existingCodes)),
		FailureRate:        master.FailureRate,
		RunningHours:       domain.NumberPtr(domain.NumberFromInt(defaultRunning)),
		ConnectionType:     domain.StringPtr("series"),
		ActiveComponent:    domain.IntPtr(1),
		TotalComponent:     domain.IntPtr(1),
		DistributionType:   domain.StringPtr("exponential"),
		ReliabilityValue:   exponentialAtThousandHours(master.FailureRate),
		Active:             domain.IntPtr(1),
		CreatedAt:          &now,
		UpdatedAt:          &now,
		CreatedBy:          domain.StringPtr(currentUser),
		UpdatedBy:          domain.StringPtr(currentUser),
	}
	component.IdNode = domain.StringPtr(component.SystemComponentId)
	if err := s.store.InsertComponents(ctx, []domain.SystemComponentProperties{component}); err != nil {
		return nil, err
	}
	detail := domain.ComponentDetailOf(component)
	return &detail, nil
}

func (s *ComponentService) UpdateDetail(ctx context.Context, systemComponentId string, request domain.ComponentDetailUpdate, currentUser string) (*domain.ComponentDetail, error) {
	var detail *domain.ComponentDetail
	err := s.store.Transact(ctx, func(tx *Store) error {
		existing, err := tx.FindComponent(ctx, systemComponentId)
		if err != nil || existing == nil {
			return err
		}
		topologyChanged := false
		rbdSystemId := domain.Deref(existing.RbdSystemId)
		setText := func(target **string, value *string) {
			if strings.TrimSpace(domain.Deref(value)) != "" {
				*target = value
			}
		}
		setText(&existing.ParentId, request.ParentId)
		if strings.TrimSpace(domain.Deref(request.ComponentName)) != "" {
			existing.ComponentName = *request.ComponentName
		}
		setText(&existing.ComponentTagNumber, request.ComponentTagNumber)
		setText(&existing.Vendor, request.Vendor)
		if strings.TrimSpace(domain.Deref(request.FormulaCode)) != "" {
			if domain.Deref(existing.FormulaCode) != *request.FormulaCode {
				topologyChanged = true
			}
			existing.FormulaCode = request.FormulaCode
		}
		setText(&existing.DistributionType, request.DistributionType)
		if request.FailureRate != nil {
			existing.FailureRate = request.FailureRate
		}
		if request.RunningHours != nil {
			existing.RunningHours = request.RunningHours
		}
		if request.ScaleParameter != nil {
			existing.ScaleParameter = request.ScaleParameter
		}
		if request.ShapeParameter != nil {
			existing.ShapeParameter = request.ShapeParameter
		}
		setText(&existing.ConnectionType, request.ConnectionType)
		if strings.TrimSpace(domain.Deref(request.ConnectionToId)) != "" {
			if domain.Deref(existing.ConnectionToId) != *request.ConnectionToId {
				topologyChanged = true
			}
			existing.ConnectionToId = request.ConnectionToId
		}
		setText(&existing.PositionX, request.PositionX)
		setText(&existing.PositionY, request.PositionY)
		setText(&existing.SourcePosition, request.SourcePosition)
		setText(&existing.TargetPosition, request.TargetPosition)
		if strings.TrimSpace(domain.Deref(request.IdNode)) != "" {
			if domain.Deref(existing.IdNode) != *request.IdNode {
				topologyChanged = true
			}
			existing.IdNode = request.IdNode
		}
		if request.ReliabilityValue != nil {
			existing.ReliabilityValue = request.ReliabilityValue
		}
		if request.Active != nil {
			existing.Active = request.Active
		}
		if request.ActiveComponent != nil {
			existing.ActiveComponent = request.ActiveComponent
		}
		if request.TotalComponent != nil {
			existing.TotalComponent = request.TotalComponent
		}
		if request.Regresi != nil {
			existing.Regresi = request.Regresi
		}
		if request.Mtbf != nil {
			existing.Mtbf = request.Mtbf
		}
		if request.AllowedFailures != nil {
			existing.AllowedFailures = request.AllowedFailures
		}
		existing.UpdatedBy = domain.StringPtr(currentUser)
		now := domain.Now()
		existing.UpdatedAt = &now
		if err := tx.UpdateComponent(ctx, *existing); err != nil {
			return err
		}
		if topologyChanged {
			if err := tx.regenerateSystem(ctx, rbdSystemId); err != nil {
				return err
			}
		}
		if err := tx.TouchSystem(ctx, rbdSystemId); err != nil {
			return err
		}
		view := domain.ComponentDetailOf(*existing)
		detail = &view
		return nil
	})
	return detail, err
}

func (s *ComponentService) Delete(ctx context.Context, systemComponentId string) error {
	component, err := s.store.FindComponent(ctx, systemComponentId)
	if err != nil || component == nil {
		return err
	}
	if err := s.store.DeleteComponents(ctx, []string{systemComponentId}); err != nil {
		return err
	}
	return s.store.TouchSystem(ctx, domain.Deref(component.RbdSystemId))
}

func (s *ComponentService) MonitoredComponents(ctx context.Context, search, sortBy, sortOrder string) ([]domain.MonitoredComponent, error) {
	rows, err := s.store.NamedComponents(ctx, search, false)
	if err != nil {
		return nil, err
	}
	result := make([]domain.MonitoredComponent, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.MonitoredComponent{
			ComponentName:    domain.StringPtr("(" + row.SystemComponentId + ")" + row.ComponentName),
			Vendor:           row.Vendor,
			DistributionType: row.DistributionType,
			FailureRate:      row.FailureRate,
			RunningHours:     row.RunningHours,
			ScaleParameter:   row.ScaleParameter,
			ShapeParameter:   row.ShapeParameter,
		})
	}
	compare := func(a, b domain.MonitoredComponent) int {
		return domain.CompareText(domain.Deref(a.ComponentName), domain.Deref(b.ComponentName))
	}
	if sortBy != "" && sortOrder != "" {
		switch strings.ToLower(sortBy) {
		case "vendor":
			compare = func(a, b domain.MonitoredComponent) int {
				return domain.CompareText(domain.Deref(a.Vendor), domain.Deref(b.Vendor))
			}
		case "distributiontype":
			compare = func(a, b domain.MonitoredComponent) int {
				return domain.CompareText(domain.Deref(a.DistributionType), domain.Deref(b.DistributionType))
			}
		case "failurerate":
			compare = func(a, b domain.MonitoredComponent) int { return domain.CompareNumber(a.FailureRate, b.FailureRate) }
		case "runninghours":
			compare = func(a, b domain.MonitoredComponent) int { return domain.CompareNumber(a.RunningHours, b.RunningHours) }
		}
		domain.OrderBy(result, strings.ToLower(sortOrder) == "desc", compare)
	} else {
		domain.OrderBy(result, false, compare)
	}
	return result, nil
}

func monitoredReliability(row domain.SystemComponentProperties) float64 {
	if domain.Deref(row.DistributionType) == "Exponential" {
		if row.FailureRate != nil && row.RunningHours != nil {
			return math.Exp(-(reliability.ToFloat(row.FailureRate.Decimal) * reliability.ToFloat(row.RunningHours.Decimal)))
		}
		return 0
	}
	if domain.Deref(row.DistributionType) == "Poisson" {
		if row.FailureRate != nil && row.RunningHours != nil {
			return reliability.PoissonFloat(reliability.ToFloat(row.FailureRate.Decimal), reliability.ToFloat(row.RunningHours.Decimal), domain.DerefInt(row.AllowedFailures, 0))
		}
		return 0
	}
	if row.RunningHours != nil && row.ScaleParameter != nil && row.ShapeParameter != nil && !row.ScaleParameter.IsZero() && !row.ShapeParameter.IsZero() {
		return reliability.WeibullFloat(reliability.ToFloat(row.RunningHours.Decimal), reliability.ToFloat(row.ScaleParameter.Decimal), reliability.ToFloat(row.ShapeParameter.Decimal))
	}
	return 0
}

func (s *ComponentService) CalculationSummary(ctx context.Context, search, sortBy, sortOrder string) (*domain.CalculationSummary, error) {
	rows, err := s.store.NamedComponents(ctx, search, true)
	if err != nil {
		return nil, err
	}
	result := make([]domain.CalculationRow, 0, len(rows))
	for _, row := range rows {
		value := monitoredReliability(row)
		result = append(result, domain.CalculationRow{
			ComponentName:    domain.StringPtr("(" + row.SystemComponentId + ")" + row.ComponentName),
			ConnectionType:   row.ConnectionType,
			ReliabilityValue: &value,
		})
	}
	compare := func(a, b domain.CalculationRow) int {
		return domain.CompareText(domain.Deref(a.ComponentName), domain.Deref(b.ComponentName))
	}
	if sortBy != "" && sortOrder != "" {
		switch strings.ToLower(sortBy) {
		case "connectiontype":
			compare = func(a, b domain.CalculationRow) int {
				return domain.CompareText(domain.Deref(a.ConnectionType), domain.Deref(b.ConnectionType))
			}
		case "reliabilityvalue":
			compare = func(a, b domain.CalculationRow) int {
				return domain.CompareFloat(derefFloat(a.ReliabilityValue), derefFloat(b.ReliabilityValue))
			}
		}
		domain.OrderBy(result, strings.ToLower(sortOrder) == "desc", compare)
	} else {
		domain.OrderBy(result, false, compare)
	}
	total := 1.0
	for _, row := range result {
		total *= derefFloat(row.ReliabilityValue)
	}
	return &domain.CalculationSummary{DataCalculation: result, ReliabilityTotal: &total}, nil
}

func derefFloat(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}
