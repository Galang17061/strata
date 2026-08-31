package reliability

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestPoissonMatchesExponentialWhenNoFaultIsTolerated(t *testing.T) {
	rate := decimal.RequireFromString("0.00000851")
	hours := decimal.RequireFromString("8000")
	poisson := PoissonFromFloats(rate, hours, 0)
	exponential := ExponentialFromFloats(rate, hours)
	assert.True(t, poisson.Equal(exponential))
	assert.Equal(t, "0.934185735728881", poisson.String())
}

func TestPoissonGrowsWithEachToleratedFault(t *testing.T) {
	rate := decimal.RequireFromString("0.00000851")
	hours := decimal.RequireFromString("8000")
	assert.Equal(t, "0.997785100617304", PoissonFromFloats(rate, hours, 1).String())
	assert.Equal(t, "0.999950022998105", PoissonFromFloats(rate, hours, 2).String())
}

func TestPoissonHoldsAtHeavyExpectedFailureCounts(t *testing.T) {
	rate := decimal.RequireFromString("0.0005")
	hours := decimal.RequireFromString("10000")
	assert.Equal(t, "0.265025915297362", PoissonFromFloats(rate, hours, 3).String())
}
