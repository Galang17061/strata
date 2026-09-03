package rbd

import (
	"context"
	"strconv"
	"strings"

	"github.com/Galang17061/strata-api/internal/auth"
	"github.com/Galang17061/strata-api/internal/domain"
)

type HierarchyService struct {
	store     *Store
	snapshots *SnapshotService
}

func NewHierarchyService(store *Store) *HierarchyService {
	return &HierarchyService{store: store}
}

func (s *HierarchyService) WithSnapshots(snapshots *SnapshotService) *HierarchyService {
	s.snapshots = snapshots
	return s
}

func (s *HierarchyService) List(ctx context.Context, search, sortBy, sortOrder string) ([]domain.HierarchyView, error) {
	rows, err := s.store.SearchHierarchies(ctx, search)
	if err != nil {
		return nil, err
	}
	views := make([]domain.HierarchyView, 0, len(rows))
	for _, row := range rows {
		views = append(views, domain.HierarchyViewOf(row))
	}
	if sortBy != "" && sortOrder != "" {
		domain.OrderBy(views, strings.ToLower(sortOrder) == "desc", hierarchyComparer(sortBy))
	} else {
		domain.OrderBy(views, false, hierarchyComparer(""))
	}
	return views, nil
}

func hierarchyComparer(sortBy string) func(a, b domain.HierarchyView) int {
	byText := func(pick func(v domain.HierarchyView) string) func(a, b domain.HierarchyView) int {
		return func(a, b domain.HierarchyView) int { return domain.CompareText(pick(a), pick(b)) }
	}
	switch strings.ToLower(sortBy) {
	case "subsystemname":
		return byText(func(v domain.HierarchyView) string { return domain.Deref(v.SubSystemName) })
	case "formula":
		return byText(func(v domain.HierarchyView) string { return domain.Deref(v.Formula) })
	case "formulacode":
		return byText(func(v domain.HierarchyView) string { return domain.Deref(v.FormulaCode) })
	case "connectiontype":
		return byText(func(v domain.HierarchyView) string { return domain.Deref(v.ConnectionType) })
	case "level":
		return func(a, b domain.HierarchyView) int { return domain.CompareInt64(int64(a.Level), int64(b.Level)) }
	case "realibilityvalue":
		return func(a, b domain.HierarchyView) int {
			return domain.CompareNumber(a.RealibilityValue, b.RealibilityValue)
		}
	case "runninghours":
		return func(a, b domain.HierarchyView) int {
			return domain.CompareInt64(int64(domain.DerefInt(a.RunningHours, 0)), int64(domain.DerefInt(b.RunningHours, 0)))
		}
	default:
		return byText(func(v domain.HierarchyView) string {
			if v.SubSystemName != nil {
				return *v.SubSystemName
			}
			return v.HierarchyId
		})
	}
}

func (s *HierarchyService) Find(ctx context.Context, hierarchyId string) (*domain.HierarchyView, error) {
	row, err := s.store.FindHierarchy(ctx, hierarchyId)
	if err != nil || row == nil {
		return nil, err
	}
	view := domain.HierarchyViewOf(*row)
	return &view, nil
}

func (s *HierarchyService) Children(ctx context.Context, parentId string) ([]domain.HierarchyView, error) {
	rows, err := s.store.ChildHierarchies(ctx, parentId)
	if err != nil {
		return nil, err
	}
	views := make([]domain.HierarchyView, 0, len(rows))
	for _, row := range rows {
		views = append(views, domain.HierarchyViewOf(row))
	}
	return views, nil
}

func (s *HierarchyService) Create(ctx context.Context, request domain.HierarchyCreate) (*domain.HierarchyView, error) {
	level := domain.DerefInt(request.Level, 0)
	rbdSystemId := domain.Deref(request.RbdSystemId)
	exists, err := s.store.SystemExists(ctx, rbdSystemId)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.InvalidOperation("RBD System " + rbdSystemId + " not found")
	}
	depth, err := s.store.ProjectDepthOfSystem(ctx, rbdSystemId)
	if err != nil {
		return nil, err
	}
	if level < 1 || level > depth {
		return nil, domain.InvalidOperation("Level must be between 1 and " + strconv.Itoa(depth))
	}
	parentId := domain.Deref(request.ParentId)
	if parentId != rbdSystemId {
		parentExists, err := s.store.HierarchyExists(ctx, parentId)
		if err != nil {
			return nil, err
		}
		if !parentExists {
			systemExists, err := s.store.SystemExists(ctx, parentId)
			if err != nil {
				return nil, err
			}
			if !systemExists {
				return nil, domain.InvalidOperation("Parent with ID " + parentId + " not found")
			}
		}
	}
	ids, err := s.store.HierarchyIds(ctx)
	if err != nil {
		return nil, err
	}
	counter := 0
	for _, id := range ids {
		if number, ok := parseAfter(id, "H-"); ok && number > counter {
			counter = number
		}
	}
	formulaCode := strings.TrimSpace(domain.Deref(request.FormulaCode))
	if formulaCode == "" {
		existing, err := s.store.HierarchyCodesOfSystem(ctx, rbdSystemId, "HS")
		if err != nil {
			return nil, err
		}
		formulaCode = nextCodeWithPrefix("HS", existing)
	} else {
		formulaCode = *request.FormulaCode
	}
	hierarchy := domain.Hierarchy{
		HierarchyId:    formatId("H-", counter+1),
		RbdSystemId:    domain.StringPtr(rbdSystemId),
		ParentId:       parentId,
		Level:          level,
		SubSystemName:  request.SubSystemName,
		Formula:        request.Formula,
		FormulaCode:    domain.StringPtr(formulaCode),
		ConnectionType: request.ConnectionType,
		RunningHours:   request.RunningHours,
		PositionX:      request.PositionX,
		PositionY:      request.PositionY,
		SourceId:       request.SourceId,
		TargetId:       request.TargetId,
	}
	if err := s.store.InsertHierarchies(ctx, []domain.Hierarchy{hierarchy}); err != nil {
		return nil, err
	}
	if err := s.store.TouchSystem(ctx, rbdSystemId); err != nil {
		return nil, err
	}
	view := domain.HierarchyViewOf(hierarchy)
	return &view, nil
}

func parseAfter(id, prefix string) (int, bool) {
	number := 0
	text := strings.Replace(id, prefix, "", 1)
	if text == "" {
		return 0, false
	}
	for _, c := range text {
		if c < '0' || c > '9' {
			return 0, false
		}
		number = number*10 + int(c-'0')
	}
	return number, true
}

func (s *HierarchyService) Update(ctx context.Context, hierarchyId string, request domain.HierarchyUpdate) (*domain.HierarchyView, error) {
	existing, err := s.store.FindHierarchy(ctx, hierarchyId)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, domain.InvalidOperation("Hierarchy with ID " + hierarchyId + " not found")
	}
	if request.SubSystemName != nil {
		existing.SubSystemName = request.SubSystemName
	}
	if request.Formula != nil {
		existing.Formula = request.Formula
	}
	if request.FormulaCode != nil {
		existing.FormulaCode = request.FormulaCode
	}
	if request.ConnectionType != nil {
		existing.ConnectionType = request.ConnectionType
	}
	if request.RealibilityValue != nil {
		existing.RealibilityValue = request.RealibilityValue
	}
	if request.RunningHours != nil {
		existing.RunningHours = request.RunningHours
	}
	if request.PositionX != nil {
		existing.PositionX = request.PositionX
	}
	if request.PositionY != nil {
		existing.PositionY = request.PositionY
	}
	if request.SourceId != nil {
		existing.SourceId = request.SourceId
	}
	if request.TargetId != nil {
		existing.TargetId = request.TargetId
	}
	if err := s.store.UpdateHierarchy(ctx, *existing); err != nil {
		return nil, err
	}
	if err := s.store.TouchSystem(ctx, domain.Deref(existing.RbdSystemId)); err != nil {
		return nil, err
	}
	view := domain.HierarchyViewOf(*existing)
	return &view, nil
}

func (s *HierarchyService) Delete(ctx context.Context, hierarchyId string) error {
	existing, err := s.store.FindHierarchy(ctx, hierarchyId)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.InvalidOperation("Hierarchy with ID " + hierarchyId + " not found")
	}
	if s.snapshots != nil && domain.Deref(existing.RbdSystemId) != "" {
		label := "Before removing " + strings.TrimSpace(domain.Deref(existing.SubSystemName))
		s.snapshots.Create(ctx, domain.Deref(existing.RbdSystemId), label, "auto", auth.CurrentUserName(ctx))
	}
	hasChildren, err := s.store.HasChildHierarchies(ctx, hierarchyId)
	if err != nil {
		return err
	}
	if hasChildren {
		return domain.InvalidOperation("Cannot delete hierarchy with children. Delete children first.")
	}
	components, err := s.store.ComponentsOfParent(ctx, hierarchyId, false)
	if err != nil {
		return err
	}
	if len(components) > 0 {
		return domain.InvalidOperation("Cannot delete hierarchy with components. Delete components first.")
	}
	if err := s.store.DeleteHierarchies(ctx, []string{hierarchyId}); err != nil {
		return err
	}
	return s.store.TouchSystem(ctx, domain.Deref(existing.RbdSystemId))
}

func (s *HierarchyService) CanAddComponent(ctx context.Context, hierarchyId string) (bool, error) {
	hierarchy, err := s.store.FindHierarchy(ctx, hierarchyId)
	if err != nil || hierarchy == nil {
		return false, err
	}
	maxLevel, err := s.store.MaxLevelOfSystem(ctx, hierarchy.RbdSystemId)
	if err != nil {
		return false, err
	}
	return hierarchy.Level == maxLevel, nil
}
