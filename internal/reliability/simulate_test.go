package reliability

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func exponentialPart(code string, rate float64) SimulationPart {
	return SimulationPart{Code: code, Distribution: "exponential", FailureRate: rate, Active: 1, Total: 1}
}

func TestSimulateMatchesTheExponentialCurve(t *testing.T) {
	output, err := Simulate(SimulationInput{
		Formula:      "A",
		Parts:        []SimulationPart{exponentialPart("A", 0.0001)},
		MissionHours: 1000,
		Trials:       40000,
		Seed:         7,
	})
	require.NoError(t, err)
	assert.InDelta(t, math.Exp(-0.1), output.Reliability, 0.01)
	assert.InDelta(t, 10000, output.MeanLife, 400)
	assert.True(t, output.LowerBound < output.Reliability && output.Reliability < output.UpperBound)
}

func TestSimulateMatchesASeriesOfTwo(t *testing.T) {
	output, err := Simulate(SimulationInput{
		Formula:      "(A*B)",
		Parts:        []SimulationPart{exponentialPart("A", 0.0001), exponentialPart("B", 0.0001)},
		MissionHours: 1000,
		Trials:       40000,
		Seed:         11,
	})
	require.NoError(t, err)
	assert.InDelta(t, math.Exp(-0.2), output.Reliability, 0.01)
}

func TestSimulateMatchesAParallelPair(t *testing.T) {
	single := math.Exp(-0.1)
	output, err := Simulate(SimulationInput{
		Formula:      "(1-((1-A)*(1-B)))",
		Parts:        []SimulationPart{exponentialPart("A", 0.0001), exponentialPart("B", 0.0001)},
		MissionHours: 1000,
		Trials:       40000,
		Seed:         13,
	})
	require.NoError(t, err)
	assert.InDelta(t, 1-(1-single)*(1-single), output.Reliability, 0.01)
}

func TestSimulateMatchesTwoOutOfThree(t *testing.T) {
	single := math.Exp(-0.1)
	expected := 3*single*single*(1-single) + single*single*single
	output, err := Simulate(SimulationInput{
		Formula:      "A",
		Parts:        []SimulationPart{{Code: "A", Distribution: "exponential", FailureRate: 0.0001, Active: 2, Total: 3}},
		MissionHours: 1000,
		Trials:       40000,
		Seed:         17,
	})
	require.NoError(t, err)
	assert.InDelta(t, expected, output.Reliability, 0.01)
}

func TestSimulateMatchesTheWeibullCurve(t *testing.T) {
	output, err := Simulate(SimulationInput{
		Formula:      "A",
		Parts:        []SimulationPart{{Code: "A", Distribution: "weibull", Scale: 1000, Shape: 2, Active: 1, Total: 1}},
		MissionHours: 500,
		Trials:       40000,
		Seed:         19,
	})
	require.NoError(t, err)
	expected, err := WeibullFromFloats(decimal.NewFromInt(500), decimal.NewFromInt(1000), decimal.NewFromInt(2))
	require.NoError(t, err)
	analytical, _ := expected.Float64()
	assert.InDelta(t, analytical, output.Reliability, 0.01)
}

func TestSimulateMatchesPoissonWithOneFailureAllowed(t *testing.T) {
	output, err := Simulate(SimulationInput{
		Formula:      "A",
		Parts:        []SimulationPart{{Code: "A", Distribution: "poisson", FailureRate: 0.001, AllowedFailures: 1, Active: 1, Total: 1}},
		MissionHours: 1000,
		Trials:       40000,
		Seed:         23,
	})
	require.NoError(t, err)
	assert.InDelta(t, PoissonFloat(0.001, 1000, 1), output.Reliability, 0.01)
}

func TestSimulateBlamesTheWeakestComponent(t *testing.T) {
	output, err := Simulate(SimulationInput{
		Formula:      "(A*B)",
		Parts:        []SimulationPart{exponentialPart("A", 0.001), exponentialPart("B", 0.00001)},
		MissionHours: 1000,
		Trials:       20000,
		Seed:         29,
	})
	require.NoError(t, err)
	require.NotEmpty(t, output.Culprits)
	assert.Equal(t, "A", output.Culprits[0].Code)
	assert.Greater(t, output.Culprits[0].Share, 0.9)
}

func TestSimulateRepeatsItselfForTheSameSeed(t *testing.T) {
	input := SimulationInput{
		Formula:      "(A*B)",
		Parts:        []SimulationPart{exponentialPart("A", 0.0002), exponentialPart("B", 0.0001)},
		MissionHours: 1000,
		Trials:       5000,
		Seed:         31,
	}
	first, err := Simulate(input)
	require.NoError(t, err)
	second, err := Simulate(input)
	require.NoError(t, err)
	assert.Equal(t, first.Reliability, second.Reliability)
	assert.Equal(t, first.MeanLife, second.MeanLife)
}

func TestSimulateDrawsACurveThatOnlyFalls(t *testing.T) {
	output, err := Simulate(SimulationInput{
		Formula:      "(A*B)",
		Parts:        []SimulationPart{exponentialPart("A", 0.0001), exponentialPart("B", 0.0001)},
		MissionHours: 2000,
		Trials:       10000,
		Seed:         37,
		CurvePoints:  10,
	})
	require.NoError(t, err)
	require.Len(t, output.Curve, 11)
	assert.Equal(t, 1.0, output.Curve[0].Reliability)
	for i := 1; i < len(output.Curve); i++ {
		assert.LessOrEqual(t, output.Curve[i].Reliability, output.Curve[i-1].Reliability)
	}
	assert.InDelta(t, output.Reliability, output.Curve[len(output.Curve)-1].Reliability, 0.000001)
}

func TestWilsonBoundsReachTheEdgesWhenEveryRunAgrees(t *testing.T) {
	lower, upper := wilsonBounds(500, 500)
	assert.Equal(t, 1.0, upper)
	assert.Less(t, lower, 1.0)
	lower, upper = wilsonBounds(0, 500)
	assert.Equal(t, 0.0, lower)
	assert.Greater(t, upper, 0.0)
}

func TestSimulateRefusesAnEmptyMission(t *testing.T) {
	_, err := Simulate(SimulationInput{Formula: "A", Parts: []SimulationPart{exponentialPart("A", 0.001)}})
	assert.Error(t, err)
}

func TestSimulateRefusesComponentsTheFormulaNeverMentions(t *testing.T) {
	_, err := Simulate(SimulationInput{
		Formula:      "A",
		Parts:        []SimulationPart{exponentialPart("Z", 0.001)},
		MissionHours: 100,
	})
	assert.Error(t, err)
}
