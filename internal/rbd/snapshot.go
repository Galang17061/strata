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
