package rbd

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Galang17061/strata-api/internal/domain"
)

func hierarchyRow(id, parent string) domain.Hierarchy {
	return domain.Hierarchy{HierarchyId: id, ParentId: parent}
}

func TestAStraightChainRaisesNoAlarm(t *testing.T) {
	rows := []domain.Hierarchy{
		hierarchyRow("H-1", "SYS"),
		hierarchyRow("H-2", "H-1"),
		hierarchyRow("H-3", "H-2"),
	}
	assert.Nil(t, CycleAmongHierarchies(rows))
}

func TestALoopIsNamedLinkByLink(t *testing.T) {
	rows := []domain.Hierarchy{
		hierarchyRow("H-1", "H-3"),
		hierarchyRow("H-2", "H-1"),
		hierarchyRow("H-3", "H-2"),
	}
	chain := CycleAmongHierarchies(rows)
	assert.Len(t, chain, 3)
	assert.ElementsMatch(t, []string{"H-1", "H-2", "H-3"}, chain)
}

func TestASelfParentIsTheSmallestLoop(t *testing.T) {
	rows := []domain.Hierarchy{hierarchyRow("H-1", "H-1")}
	assert.Equal(t, []string{"H-1"}, CycleAmongHierarchies(rows))
}

func TestBranchesOffACleanTrunkStayQuiet(t *testing.T) {
	rows := []domain.Hierarchy{
		hierarchyRow("H-1", "SYS"),
		hierarchyRow("H-2", "H-1"),
		hierarchyRow("H-3", "H-1"),
		hierarchyRow("H-4", "H-3"),
	}
	assert.Nil(t, CycleAmongHierarchies(rows))
}
