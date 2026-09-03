package rbd

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Galang17061/strata-api/internal/domain"
)

const autoSnapshotsKept = 20

type SnapshotService struct {
	store *Store
}

func NewSnapshotService(store *Store) *SnapshotService {
	return &SnapshotService{store: store}
}

type snapshotPayload struct {
	System      domain.RbdSystemDrawing            `json:"system"`
	Hierarchies []domain.Hierarchy                 `json:"hierarchies"`
	Components  []domain.SystemComponentProperties `json:"components"`
	Edges       []domain.SystemComponentDrawing    `json:"edges"`
}

func snapshotNodeIds(hierarchies []domain.Hierarchy, components []domain.SystemComponentProperties) []string {
	nodeIds := []string{}
	for _, hierarchy := range hierarchies {
		nodeIds = append(nodeIds, hierarchy.HierarchyId, domain.Deref(hierarchy.FormulaCode), domain.Deref(hierarchy.SourceId), domain.Deref(hierarchy.TargetId))
	}
	for _, component := range components {
		nodeIds = append(nodeIds, component.SystemComponentId, domain.Deref(component.IdNode), domain.Deref(component.FormulaCode))
	}
	unique := []string{}
	seen := map[string]bool{}
	for _, nodeId := range nodeIds {
		if nodeId != "" && !seen[nodeId] {
			seen[nodeId] = true
			unique = append(unique, nodeId)
		}
	}
	return unique
}

func (s *SnapshotService) collect(ctx context.Context, rbdSystemId string) (*snapshotPayload, error) {
	system, err := s.store.FindSystem(ctx, rbdSystemId)
	if err != nil {
		return nil, err
	}
	if system == nil {
		return nil, domain.KeyNotFound("System not found")
	}
	hierarchies, err := s.store.HierarchiesOfSystemByLevel(ctx, rbdSystemId, true)
	if err != nil {
		return nil, err
	}
	components, err := s.store.ComponentsOfSystem(ctx, rbdSystemId)
	if err != nil {
		return nil, err
	}
	edges, err := s.store.EdgesTouching(ctx, snapshotNodeIds(hierarchies, components))
	if err != nil {
		return nil, err
	}
	return &snapshotPayload{System: *system, Hierarchies: hierarchies, Components: components, Edges: edges}, nil
}

func (s *SnapshotService) Create(ctx context.Context, rbdSystemId, label, kind, currentUser string) (*domain.SystemSnapshot, error) {
	payload, err := s.collect(ctx, rbdSystemId)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(label)
	if trimmed == "" {
		trimmed = "Version of " + domain.Now().Text()
	}
	row := domain.SystemSnapshot{
		SystemSnapshotId: domain.NewGuid().String(),
		RbdSystemId:      rbdSystemId,
		Label:            trimmed,
		Kind:             kind,
		Payload:          string(encoded),
		CreatedBy:        domain.StringPtr(currentUser),
		CreatedAt:        domain.Now(),
	}
	if err := s.store.InsertSnapshot(ctx, row); err != nil {
		return nil, err
	}
	if kind == "auto" {
		if err := s.store.PruneAutoSnapshots(ctx, rbdSystemId, autoSnapshotsKept); err != nil {
			return nil, err
		}
	}
	row.Payload = ""
	return &row, nil
}

func (s *SnapshotService) List(ctx context.Context, rbdSystemId string) ([]domain.SystemSnapshot, error) {
	return s.store.SnapshotsOfSystem(ctx, rbdSystemId)
}

func (s *SnapshotService) Restore(ctx context.Context, snapshotId string, systemName *string, currentUser string) (*domain.SnapshotRestoreResult, error) {
	snapshot, err := s.store.FindSnapshot(ctx, snapshotId)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, domain.KeyNotFound("Snapshot not found")
	}
	var payload snapshotPayload
	if err := json.Unmarshal([]byte(snapshot.Payload), &payload); err != nil {
		return nil, domain.InvalidOperation("This version cannot be read any more.")
	}
	if chain := CycleAmongHierarchies(payload.Hierarchies); chain != nil {
		return nil, domain.InvalidOperation("This version is corrupt: its layers loop back on themselves (" + strings.Join(chain, " -> ") + ").")
	}
	result := &domain.SnapshotRestoreResult{}
	err = s.store.Transact(ctx, func(tx *Store) error {
		project, err := tx.FindProject(ctx, payload.System.ProjectId)
		if err != nil {
			return err
		}
		if project == nil {
			return domain.InvalidOperation("The project this version belonged to no longer exists.")
		}
		systemIds, err := tx.SystemIds(ctx)
		if err != nil {
			return err
		}
		newSystemId := nextSequenceId(systemIds, "RS-")
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
		for _, hierarchy := range payload.Hierarchies {
			remap.add(hierarchy.HierarchyId, formatId("H-", hierarchyCounter))
			hierarchyCounter++
		}
		componentCounter := counterAfter(lastComponent, "SCP-")
		for _, component := range payload.Components {
			remap.add(component.SystemComponentId, formatId("SCP-", componentCounter))
			componentCounter++
		}
		remap.add(payload.System.RbdSystemId, newSystemId)
		now := domain.Now()
		name := domain.Deref(payload.System.SystemName) + " (" + snapshot.Label + ")"
		if trimmed := strings.TrimSpace(domain.Deref(systemName)); trimmed != "" {
			name = trimmed
		}
		if err := tx.InsertSystem(ctx, domain.RbdSystemDrawing{
			RbdSystemId:  newSystemId,
			ProjectId:    payload.System.ProjectId,
			DrawingName:  payload.System.DrawingName,
			SystemName:   domain.StringPtr(name),
			RunningHours: payload.System.RunningHours,
			Formula:      remap.textPtr(payload.System.Formula),
			CreatedAt:    now,
			UpdatedAt:    now,
			CreatedBy:    domain.StringPtr(currentUser),
			UpdatedBy:    domain.StringPtr(currentUser),
		}); err != nil {
			return err
		}
		newHierarchies := make([]domain.Hierarchy, 0, len(payload.Hierarchies))
		for _, hierarchy := range payload.Hierarchies {
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
		newComponents := make([]domain.SystemComponentProperties, 0, len(payload.Components))
		for _, component := range payload.Components {
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
			newComponents = append(newComponents, copied)
		}
		if err := tx.InsertComponents(ctx, newComponents); err != nil {
			return err
		}
		newEdges := make([]domain.SystemComponentDrawing, 0, len(payload.Edges))
		for _, edge := range payload.Edges {
			newEdges = append(newEdges, domain.SystemComponentDrawing{
				IdEdge:   domain.NewGuid().String(),
				SourceId: remap.textPtr(edge.SourceId),
				TargetId: remap.textPtr(edge.TargetId),
			})
		}
		if err := tx.InsertEdges(ctx, newEdges); err != nil {
			return err
		}
		result.ProjectId = payload.System.ProjectId
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

func (s *SnapshotService) Delete(ctx context.Context, snapshotId string) error {
	snapshot, err := s.store.FindSnapshot(ctx, snapshotId)
	if err != nil {
		return err
	}
	if snapshot == nil {
		return nil
	}
	return s.store.DeleteSnapshot(ctx, snapshotId)
}
