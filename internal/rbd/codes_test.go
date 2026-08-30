package rbd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Galang17061/strata-api/internal/domain"
)

func TestNextSequenceIdReadsBothIdShapes(t *testing.T) {
	assert.Equal(t, "00000013", nextSequenceId([]string{"RS-00012", "00000005", "junk"}, "RS-"))
	assert.Equal(t, "00000001", nextSequenceId(nil, "RS-"))
}

func TestCounterAfterFallsBackWhenPrefixDiffers(t *testing.T) {
	last := "SCP-00000009"
	assert.Equal(t, 10, counterAfter(&last, "SCP-"))
	legacy := "SC-00030"
	assert.Equal(t, 1, counterAfter(&legacy, "SCP-"))
	assert.Equal(t, 1, counterAfter(nil, "H-"))
	assert.Equal(t, "H-00000010", formatId("H-", 10))
}

func TestCodesRespectExistingSuffixes(t *testing.T) {
	code := nextCodeWithPrefix("C", []string{"CA1", "CA3", "CAB1"})
	assert.True(t, strings.HasPrefix(code, "C"))
	digits := strings.TrimLeft(code, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	assert.NotEmpty(t, digits)
	avoiding := nextCodeAvoiding("HS", func(candidate string) bool { return strings.HasSuffix(candidate, "1") })
	assert.True(t, strings.HasSuffix(avoiding, "2"))
}

func TestComponentFormulaRepeatsActiveUnitsAndSkipsVirtuals(t *testing.T) {
	series := "series"
	parallel := "parallel"
	components := []domain.SystemComponentProperties{
		{FormulaCode: domain.StringPtr("C1"), ActiveComponent: domain.IntPtr(2)},
		{FormulaCode: domain.StringPtr("INH-00000001")},
		{FormulaCode: domain.StringPtr("IN00000001")},
		{FormulaCode: domain.StringPtr("C2")},
	}
	assert.Equal(t, "C1*C1*INH-00000001*C2", *componentFormula(components, &series))
	assert.Equal(t, "1-((1-C1)*(1-C1)*(1-INH-00000001)*(1-C2))", *componentFormula(components, &parallel))
	assert.Equal(t, "1-(1-C2)", *componentFormula(components[3:], &parallel))
	assert.Nil(t, componentFormula(nil, nil))
	children := []domain.Hierarchy{{FormulaCode: domain.StringPtr("HSA1")}, {FormulaCode: domain.StringPtr("HSB1")}}
	assert.Equal(t, "HSA1*HSB1", *hierarchyFormula(children, nil))
	assert.Equal(t, "-100.00", formatPosition(-100))
}
