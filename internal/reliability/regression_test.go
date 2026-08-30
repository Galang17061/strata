package reliability

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegressionRecoversStraightLine(t *testing.T) {
	x := []float64{1, 2, 3, 4}
	y := []float64{3, 5, 7, 9}
	assert.InDelta(t, 2, SlopeWeibull(x, y), 1e-12)
	assert.InDelta(t, 1, InterceptWeibull(x, y), 1e-12)
	slope, err := SlopeExponential(y, x)
	require.NoError(t, err)
	assert.InDelta(t, 2, slope, 1e-12)
	fit, err := RSquared(y, x)
	require.NoError(t, err)
	assert.InDelta(t, 1, fit, 1e-12)
	_, err = SlopeExponential(nil, nil)
	assert.EqualError(t, err, "Invalid input data for slope calculation.")
}

func TestWeibullPlotPointMatchesMedianRankSteps(t *testing.T) {
	frequency := MedianRank(1, 2)
	assert.InDelta(t, 0.29166666666666669, frequency, 1e-15)
	x, y := WeibullPlotPoint(1000, frequency)
	assert.InDelta(t, math.Log(1000), x, 1e-12)
	assert.InDelta(t, math.Log(math.Log(1/(1-frequency))), y, 1e-12)
	zeroX, _ := WeibullPlotPoint(0, frequency)
	assert.InDelta(t, math.Log(0.001), zeroX, 1e-12)
	assert.Equal(t, 0.0, Finite(math.NaN()))
}
