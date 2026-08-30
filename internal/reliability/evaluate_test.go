package reliability

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func text(d decimal.Decimal) string {
	if d.Exponent() < 0 {
		return d.StringFixed(-d.Exponent())
	}
	return d.String()
}

func TestEvaluateMatchesReferenceArithmetic(t *testing.T) {
	cases := map[string]string{
		"(1-(1-0.95)*(1-0.9))*0.85":                                    "0.84575",
		"0.90483741803596*0.90483741803596":                            "0.8187307530779826311038531216",
		"1-(1-0.950000000000000000)*(1-0.900000000000000000)*(1-0.85)": "0.9992500000000000000000000000",
		"1*0.5":               "0.5",
		"(1-(1-0.9)*(1-0.9))": "0.99",
		"0.1234567890123456789*0.9876543210987654321":                        "0.1219326311370217952237463801",
		"0.999999999999999999*0.999999999999999999*0.999999999999999999*0.5": "0.4999999999999999985000000000",
		"1-1":            "0",
		"2*3":            "6",
		"1.0*1.0":        "1.00",
		"(0.5+0.25)*1.0": "0.750",
		"1-(1-0.123456789012345678)*(1-0.987654321098765432)*(1-0.5)*(1-0.25)":          "0.9959419296152834933790580704",
		"0.94941336*0.93418574*0.98870428*0.80361097*0.89218658":                        "0.6287189198412230722805294734",
		"(1-(1-0.01419309)*(1-0.33287108))*(0.89218658*0.4916442*0.98870428)*0.4916442": "0.0729929990100153169100228475",
		"7/2":      "3.5",
		"7.0/2":    "3.5",
		"1/3.0":    "0.3333333333333333333333333333",
		"-(0.5)*2": "-1.0",
	}
	for formula, expected := range cases {
		value, err := Evaluate(formula)
		require.NoError(t, err, formula)
		assert.Equal(t, expected, text(value), formula)
	}
}

func TestEvaluateRejectsBrokenInput(t *testing.T) {
	_, err := Evaluate("")
	assert.Error(t, err)
	_, err = Evaluate("C1*0.5")
	assert.EqualError(t, err, "Cannot find column [C1].")
	_, err = Evaluate("(0.5*0.5")
	assert.Error(t, err)
	_, err = Evaluate("0.5/0")
	assert.EqualError(t, err, "Attempted to divide by zero.")
}

func TestPrepareFormulaTrimsAndBalances(t *testing.T) {
	assert.Equal(t, "R1*R2", PrepareFormula("R1*R2*", false))
	assert.Equal(t, "(0.5*(0.4))", PrepareFormula("(0.5*(0.4", true))
	assert.Equal(t, "(0.5*(0.4", PrepareFormula("(0.5*(0.4", false))
	assert.Equal(t, "1.0*R1*1.0", PrepareFormula("START*R1*END", false))
}

func TestFromFloatRoundsToFifteenDigitsLikeReference(t *testing.T) {
	cases := []struct {
		input    float64
		expected string
	}{
		{math.Exp(-0.0001 * 1000), "0.90483741803596"},
		{0.5, "0.5"},
		{0.1, "0.1"},
		{1e-7, "0.0000001"},
		{123456.789, "123456.789"},
		{math.Exp(-1e-9), "0.999999999"},
		{1.0, "1"},
		{123456789012345678.0, "123456789012346000"},
		{0.000123456789012345678, "0.000123456789012346"},
		{0.896, "0.896"},
		{3.0, "3"},
		{1e-28, "0.0000000000000000000000000001"},
		{0.12345678901234567, "0.123456789012346"},
	}
	for _, c := range cases {
		value, err := FromFloat(c.input)
		require.NoError(t, err)
		assert.Equal(t, c.expected, text(value), "%v", c.input)
	}
	_, err := FromFloat(math.NaN())
	assert.Error(t, err)
}

func TestToFloatMatchesReference(t *testing.T) {
	assert.Equal(t, 8.51e-06, ToFloat(decimal.RequireFromString("0.00000851")))
	assert.Equal(t, 8000.0, ToFloat(decimal.RequireFromString("8000")))
	assert.Equal(t, 117508.81316099, ToFloat(decimal.RequireFromString("117508.81316099")))
	assert.Equal(t, 0.95, ToFloat(decimal.RequireFromString("0.950000000000000000")))
	assert.Equal(t, 123456.78901234567, ToFloat(decimal.RequireFromString("123456.789012345678901234567")))
}

func TestFitKeepsReferencePrecision(t *testing.T) {
	a := decimal.RequireFromString("0.94941336")
	b := decimal.RequireFromString("0.93418574")
	c := decimal.RequireFromString("0.98870428")
	assert.Equal(t, "0.876909927159398151321792", text(Mul(Mul(a, b), c)))
	assert.Equal(t, "0.8550000000000000000000000000", text(Mul(decimal.RequireFromString("0.950000000000000000"), decimal.RequireFromString("0.900000000000000000"))))
	assert.Equal(t, "0.050000000000000000", text(Sub(decimal.NewFromInt(1), decimal.RequireFromString("0.950000000000000000"))))
	assert.Equal(t, "0.1219326311370217952261850318", text(Mul(decimal.RequireFromString("0.123456789012345678901234567"), decimal.RequireFromString("0.987654321098765432109876543"))))
	quotient, err := Div(decimal.NewFromInt(1), decimal.NewFromInt(3))
	require.NoError(t, err)
	assert.Equal(t, "0.3333333333333333333333333333", text(quotient))
}

func TestDistributionsMatchReference(t *testing.T) {
	rate := decimal.RequireFromString("0.00000851")
	hours := decimal.RequireFromString("8000")
	assert.Equal(t, "0.934185735728881", text(ExponentialFromDecimals(rate, hours)))
	assert.Equal(t, "0.934185735728881", text(ExponentialFromFloats(rate, hours)))
	scale := decimal.RequireFromString("117508.81316099")
	shape := decimal.RequireFromString("1.5")
	ratio, err := WeibullFromRatio(hours, scale, shape)
	require.NoError(t, err)
	assert.Equal(t, "0.982393309558311", text(ratio))
	floats, err := WeibullFromFloats(hours, scale, shape)
	require.NoError(t, err)
	assert.Equal(t, "0.982393309558311", text(floats))
	nine := decimal.RequireFromString("0.9")
	assert.Equal(t, "0.729", text(SeriesOfIdentical(nine, 3)))
	assert.Equal(t, "0.999000000000000001", text(ParallelOfIdentical(nine, 3)))
	two, err := KOutOfN(decimal.RequireFromString("0.8"), 2, 3)
	require.NoError(t, err)
	assert.Equal(t, "0.896", text(two))
	four, err := KOutOfN(decimal.RequireFromString("0.94941336"), 2, 4)
	require.NoError(t, err)
	assert.Equal(t, "0.999501839072627", text(four))
	five, err := KOutOfN(decimal.RequireFromString("0.5"), 3, 5)
	require.NoError(t, err)
	assert.Equal(t, "0.5", text(five))
	_, err = KOutOfN(nine, 4, 3)
	assert.EqualError(t, err, "Active components (4) cannot exceed total components (3)")
	assert.Equal(t, int64(10), BinomialCoefficient(5, 2))
}
