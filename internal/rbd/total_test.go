package rbd

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Galang17061/strata-api/internal/domain"
)

func component(connection string, active, total int, distribution string, hours, rate, scale, shape string) domain.SystemComponentProperties {
	c := domain.SystemComponentProperties{ConnectionType: domain.StringPtr(connection), ActiveComponent: domain.IntPtr(active), TotalComponent: domain.IntPtr(total), DistributionType: domain.StringPtr(distribution)}
	if hours != "" {
		c.RunningHours = domain.NumberPtr(domain.NewNumber(decimal.RequireFromString(hours)))
	}
	if rate != "" {
		c.FailureRate = domain.NumberPtr(domain.NewNumber(decimal.RequireFromString(rate)))
	}
	if scale != "" {
		c.ScaleParameter = domain.NumberPtr(domain.NewNumber(decimal.RequireFromString(scale)))
	}
	if shape != "" {
		c.ShapeParameter = domain.NumberPtr(domain.NewNumber(decimal.RequireFromString(shape)))
	}
	return c
}

func TestComponentReliabilityPicksDistribution(t *testing.T) {
	assert.Equal(t, "0.934185735728881", domain.NewNumber(componentReliability(component("series", 1, 1, "Exponential", "8000", "0.00000851", "", ""))).Text())
	assert.Equal(t, "0.982393309558311", domain.NewNumber(componentReliability(component("series", 1, 1, "Weibull", "8000", "", "117508.81316099", "1.5"))).Text())
	assert.Equal(t, "0.982393309558311", domain.NewNumber(componentReliability(component("series", 1, 1, "", "8000", "", "117508.81316099", "1.5"))).Text())
	assert.Equal(t, "0.934185735728881", domain.NewNumber(componentReliability(component("series", 1, 1, "other", "8000", "0.00000851", "", ""))).Text())
	assert.True(t, componentReliability(component("series", 1, 1, "Exponential", "0", "0.1", "", "")).IsZero())
	assert.True(t, componentReliability(component("series", 1, 1, "Weibull", "10", "", "", "")).IsZero())
}

func TestAdjustedReliabilityFollowsConnectionRules(t *testing.T) {
	nine := decimal.RequireFromString("0.9")
	series, err := adjustedReliability(component("series", 3, 3, "", "", "", "", ""), nine)
	require.NoError(t, err)
	assert.Equal(t, "0.729", domain.NewNumber(series).Text())
	parallel, err := adjustedReliability(component("parallel", 1, 3, "", "", "", "", ""), nine)
	require.NoError(t, err)
	assert.Equal(t, "0.999000000000000001", domain.NewNumber(parallel).Text())
	partial, err := adjustedReliability(component("parallel", 2, 3, "", "", "", "", ""), decimal.RequireFromString("0.8"))
	require.NoError(t, err)
	assert.Equal(t, "0.896", domain.NewNumber(partial).Text())
	same, err := adjustedReliability(component("redundantsi parsial", 3, 3, "", "", "", "", ""), nine)
	require.NoError(t, err)
	assert.True(t, same.Equal(nine))
	_, err = adjustedReliability(component("parallel", 4, 3, "", "", "", "", ""), nine)
	assert.Error(t, err)
	single, err := historyAdjustedReliability(component("parallel", 1, 1, "", "", "", "", ""), nine)
	require.NoError(t, err)
	assert.True(t, single.Equal(nine))
}

func TestLookupEvaluatesInInsertionOrder(t *testing.T) {
	values := newLookup()
	first := decimal.RequireFromString("0.95")
	second := decimal.RequireFromString("0.9")
	values.set("C1", &first)
	values.set("C10", &second)
	values.set("C2", nil)
	result, err := values.evaluate("(1-(1-C1)*(1-C10", true)
	require.NoError(t, err)
	assert.Equal(t, "0.99750", domain.NewNumber(result).Text())
	encoded, err := json.Marshal(values.ordered())
	require.NoError(t, err)
	assert.Equal(t, `{"C1":0.95,"C10":0.9,"C2":null}`, string(encoded))
	_, err = values.evaluate("C1*C3", false)
	assert.EqualError(t, err, "Failed to evaluate the formula: 0.95*C3. Original error: Cannot find column [C3].")
}
