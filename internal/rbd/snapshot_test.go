package rbd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Galang17061/strata-api/internal/domain"
)

func TestSnapshotPayloadSurvivesTheRoundTrip(t *testing.T) {
	payload := snapshotPayload{
		System: domain.RbdSystemDrawing{RbdSystemId: "00000010", ProjectId: "PJ-00001", SystemName: domain.StringPtr("Cooling")},
		Hierarchies: []domain.Hierarchy{{
			HierarchyId: "H-00000001", ParentId: "00000010", Level: 1,
			SubSystemName: domain.StringPtr("Fans"), FormulaCode: domain.StringPtr("RA"),
		}},
		Components: []domain.SystemComponentProperties{{
			SystemComponentId: "SCP-00000001", ComponentName: "Fan", FormulaCode: domain.StringPtr("RB"),
		}},
		Edges: []domain.SystemComponentDrawing{{IdEdge: "e1", SourceId: domain.StringPtr("RA"), TargetId: domain.StringPtr("RB")}},
	}
	encoded, err := json.Marshal(payload)
	require.NoError(t, err)
	var decoded snapshotPayload
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	assert.Equal(t, payload.System.RbdSystemId, decoded.System.RbdSystemId)
	assert.Equal(t, "Fans", domain.Deref(decoded.Hierarchies[0].SubSystemName))
	assert.Equal(t, "Fan", decoded.Components[0].ComponentName)
	assert.Equal(t, "RB", domain.Deref(decoded.Edges[0].TargetId))
}

func TestSnapshotNodeIdsGatherEveryCodeOnceAndSkipBlanks(t *testing.T) {
	hierarchies := []domain.Hierarchy{
		{HierarchyId: "H-1", FormulaCode: domain.StringPtr("RA"), SourceId: domain.StringPtr("in"), TargetId: domain.StringPtr("out")},
		{HierarchyId: "H-2", FormulaCode: domain.StringPtr("RA")},
	}
	components := []domain.SystemComponentProperties{
		{SystemComponentId: "SCP-1", IdNode: domain.StringPtr("n1"), FormulaCode: domain.StringPtr("RB")},
	}
	ids := snapshotNodeIds(hierarchies, components)
	assert.ElementsMatch(t, []string{"H-1", "H-2", "RA", "in", "out", "SCP-1", "n1", "RB"}, ids)
}
