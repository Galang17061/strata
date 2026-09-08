package reliability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlattenPutsTheChildFormulaWhereItsCodeStood(t *testing.T) {
	flat, err := FlattenFormula("(HS1*C1)", map[string]string{"HS1": "(CA1*CB1)"})
	require.NoError(t, err)
	assert.Equal(t, "(((CA1*CB1))*C1)", flat)
}

func TestFlattenReachesThroughSeveralFloors(t *testing.T) {
	flat, err := FlattenFormula("HS1", map[string]string{
		"HS1": "(HS2*CA1)",
		"HS2": "(CB1*CC1)",
	})
	require.NoError(t, err)
	assert.Equal(t, "((((CB1*CC1))*CA1))", flat)
}

func TestFlattenLeavesNumbersAlone(t *testing.T) {
	flat, err := FlattenFormula("(1-((1-CA1)*(1-CB1)))", map[string]string{})
	require.NoError(t, err)
	assert.Equal(t, "(1-((1-CA1)*(1-CB1)))", flat)
}

func TestFlattenRefusesAFormulaThatPointsAtItself(t *testing.T) {
	_, err := FlattenFormula("HS1", map[string]string{"HS1": "(HS1*CA1)"})
	assert.Error(t, err)
}

func TestFlattenSurvivesAMissingChild(t *testing.T) {
	flat, err := FlattenFormula("(HS1*HS2)", map[string]string{"HS1": "CA1"})
	require.NoError(t, err)
	assert.Equal(t, "((CA1)*HS2)", flat)
}

func TestFlattenedFormulaStillCompiles(t *testing.T) {
	flat, err := FlattenFormula("(HS1*HS2)", map[string]string{
		"HS1": "(CA1*CB1)",
		"HS2": "(1-((1-CC1)*(1-CD1)))",
	})
	require.NoError(t, err)
	structure, err := CompileStructure(flat)
	require.NoError(t, err)
	assert.Equal(t, []string{"CA1", "CB1", "CC1", "CD1"}, structure.Codes())
	assert.True(t, structure.Standing([]float64{1, 1, 1, 0}))
	assert.False(t, structure.Standing([]float64{1, 0, 1, 1}))
	assert.False(t, structure.Standing([]float64{1, 1, 0, 0}))
}
