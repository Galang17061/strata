package rbd

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Galang17061/strata-api/internal/domain"
)

func TestPlotTimesSpreadTwentyPointsWithBankersRounding(t *testing.T) {
	times := PlotTimes(decimal.NewFromInt(1000))
	require.Len(t, times, 20)
	assert.Equal(t, 0, times[0])
	assert.Equal(t, 53, times[1])
	assert.Equal(t, 105, times[2])
	assert.Equal(t, 1000, times[19])
	assert.Equal(t, []int{0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7, 8, 8, 9, 9, 10}, PlotTimes(decimal.NewFromInt(10)))
	assert.Equal(t, []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, PlotTimes(decimal.NewFromInt(1)))
}

func TestPlotReliabilityUsesStoredDistributionCase(t *testing.T) {
	exponential := component("series", 1, 1, "Exponential", "8000", "0.00000851", "", "")
	value, err := plotReliability(exponential, decimal.NewFromInt(8000))
	require.NoError(t, err)
	assert.Equal(t, "0.934185735728881", domain.NewNumber(value).Text())
	weibull := component("series", 1, 1, "weibull", "8000", "", "117508.81316099", "1.5")
	value, err = plotReliability(weibull, decimal.NewFromInt(8000))
	require.NoError(t, err)
	assert.Equal(t, "0.982393309558311", domain.NewNumber(value).Text())
	_, err = plotReliability(component("series", 1, 1, "exponential", "8000", "0.1", "", ""), decimal.NewFromInt(1))
	assert.Error(t, err)
}
