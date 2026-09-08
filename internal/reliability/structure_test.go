package reliability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileStructureCollectsEveryCodeOnce(t *testing.T) {
	structure, err := CompileStructure("(A*B*A)")
	require.NoError(t, err)
	assert.Equal(t, []string{"A", "B"}, structure.Codes())
	assert.Equal(t, 2, structure.Slots())
}

func TestStructureStandsWhileASeriesIsWhole(t *testing.T) {
	structure, err := CompileStructure("(A*B)")
	require.NoError(t, err)
	assert.True(t, structure.Standing([]float64{1, 1}))
	assert.False(t, structure.Standing([]float64{1, 0}))
	assert.False(t, structure.Standing([]float64{0, 0}))
}

func TestStructureStandsWhileOneParallelLegSurvives(t *testing.T) {
	structure, err := CompileStructure("(1-((1-A)*(1-B)))")
	require.NoError(t, err)
	assert.True(t, structure.Standing([]float64{1, 0}))
	assert.True(t, structure.Standing([]float64{0, 1}))
	assert.False(t, structure.Standing([]float64{0, 0}))
}

func TestStructureAgreesWithTheDecimalEvaluator(t *testing.T) {
	structure, err := CompileStructure("(0.9*(1-((1-0.8)*(1-0.7))))")
	require.NoError(t, err)
	expected, err := Evaluate(PrepareFormula("(0.9*(1-((1-0.8)*(1-0.7))))", true))
	require.NoError(t, err)
	value, _ := expected.Float64()
	assert.InDelta(t, value, structure.Value(nil), 0.000000001)
}

func TestCompileStructureForgivesAMissingClosingParenthesis(t *testing.T) {
	structure, err := CompileStructure("(A*B")
	require.NoError(t, err)
	assert.True(t, structure.Standing([]float64{1, 1}))
}

func TestCompileStructureRefusesNonsense(t *testing.T) {
	_, err := CompileStructure("A)B")
	assert.Error(t, err)
}
