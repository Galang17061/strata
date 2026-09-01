package optimize

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustEval(t *testing.T, formula string, values map[string]float64) float64 {
	t.Helper()
	compiled, err := Compile(formula)
	require.NoError(t, err)
	result, err := compiled.Eval(values)
	require.NoError(t, err)
	return result
}

func TestCompiledFormulaReadsTheFormulasTheServiceWrites(t *testing.T) {
	assert.InDelta(t, 0.72, mustEval(t, "(CR1*CHPZ1)", map[string]float64{"CR1": 0.9, "CHPZ1": 0.8}), 1e-12)
	assert.InDelta(t, 0.98, mustEval(t, "1-(1-RA)*(1-RB)", map[string]float64{"RA": 0.9, "RB": 0.8}), 1e-12)
	assert.InDelta(t, 0.6075, mustEval(t, "(1-(1-R1)*(1-R3))*R2*R4", map[string]float64{"R1": 0.5, "R3": 0.5, "R2": 0.9, "R4": 0.9}), 1e-12)
	assert.InDelta(t, 0.25, mustEval(t, "HS-1*C2_1", map[string]float64{"HS-1": 0.5, "C2_1": 0.5}), 1e-12)
	assert.InDelta(t, -0.5, mustEval(t, "-C1", map[string]float64{"C1": 0.5}), 1e-12)
}

func TestCompiledFormulaKeepsFifteenSeriesBlocksExact(t *testing.T) {
	values := map[string]float64{}
	formula := ""
	expected := 1.0
	rates := []float64{2.1e-6, 3.4e-7, 5.5e-6, 1.2e-7, 9.9e-6, 4.2e-7, 7.7e-6, 8.8e-8, 6.1e-6, 2.9e-7, 1.5e-6, 3.3e-7, 4.4e-6, 5.6e-8, 6.7e-7}
	for index, rate := range rates {
		name := "X" + string(rune('A'+index))
		value := math.Exp(-rate * 1000)
		values[name] = value
		expected *= value
		if index > 0 {
			formula += "*"
		}
		formula += name
	}
	assert.InDelta(t, expected, mustEval(t, formula, values), 1e-15)
}

func TestCompiledFormulaRefusesWhatItCannotRead(t *testing.T) {
	_, err := Compile("(A")
	assert.Error(t, err)
	_, err = Compile("")
	assert.Error(t, err)
	compiled, err := Compile("A*B")
	require.NoError(t, err)
	_, err = compiled.Eval(map[string]float64{"A": 1})
	assert.ErrorContains(t, err, "No value for B")
	assert.Equal(t, []string{"A", "B"}, compiled.Names)
}
