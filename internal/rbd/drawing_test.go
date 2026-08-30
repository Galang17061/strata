package rbd

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/Galang17061/strata-api/internal/domain"
)

func TestReplaceCodesFollowsLookupOrder(t *testing.T) {
	values := map[string]decimal.Decimal{
		"C1":  decimal.RequireFromString("0.950000000000000000"),
		"C10": decimal.RequireFromString("0.5"),
	}
	assert.Equal(t, "0.950000000000000000*0.9500000000000000000", replaceCodes("C1*C10", []string{"C1", "C10"}, values))
	assert.Equal(t, "0.950000000000000000*0.5", replaceCodes("C1*C10", []string{"C10", "C1"}, values))
}

func TestCodesOfKeepsOrderAndOptionalDistinct(t *testing.T) {
	components := []domain.SystemComponentProperties{{FormulaCode: domain.StringPtr("C1")}, {FormulaCode: domain.StringPtr("C1")}, {}}
	hierarchies := []domain.Hierarchy{{FormulaCode: domain.StringPtr("HSA1")}}
	assert.Equal(t, []string{"C1", "C1", "HSA1"}, codesOf(components, hierarchies, false))
	assert.Equal(t, []string{"C1", "HSA1"}, codesOf(components, hierarchies, true))
}

func TestMappedEdgesSwapIdsForCodes(t *testing.T) {
	inputs := []domain.EdgeInput{{IdEdge: domain.StringPtr("e1"), SourceId: domain.StringPtr("SCP-00000001"), TargetId: domain.StringPtr("OUT")}}
	edges := mappedEdges(inputs, map[string]string{"SCP-00000001": "C1"})
	assert.Equal(t, "C1", *edges[0].SourceId)
	assert.Equal(t, "OUT", *edges[0].TargetId)
	assert.Equal(t, "e1", edges[0].IdEdge)
}

func TestCheckSingleLevelIgnoresVirtualNodes(t *testing.T) {
	rows := []domain.Hierarchy{{Level: 2}, {Level: 999}, {Level: 2}}
	assert.NoError(t, checkSingleLevel(rows, "x: "))
	rows = append(rows, domain.Hierarchy{Level: 3})
	assert.EqualError(t, checkSingleLevel(rows, "Found levels: "), "Found levels: 2, 3")
}

func TestPositionTextUsesTwoDecimals(t *testing.T) {
	value := domain.NewNumber(decimal.RequireFromString("50.5"))
	assert.Equal(t, "50.50", *positionText(&value))
	assert.Nil(t, positionText(nil))
}
