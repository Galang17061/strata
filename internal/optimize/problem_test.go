package optimize

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdjustReliabilityFollowsTheWiringRules(t *testing.T) {
	assert.InDelta(t, 0.81, AdjustReliability(0.9, "series", 2, 2), 1e-12)
	assert.InDelta(t, 0.99, AdjustReliability(0.9, "parallel", 1, 2), 1e-12)
	assert.InDelta(t, 0.972, AdjustReliability(0.9, "parallel", 2, 3), 1e-12)
	assert.InDelta(t, 0.972, AdjustReliability(0.9, "partial", 2, 3), 1e-12)
	assert.InDelta(t, 0.9, AdjustReliability(0.9, "partial", 3, 3), 1e-12)
	assert.InDelta(t, 0.9, AdjustReliability(0.9, "", 1, 1), 1e-12)
}

func researchProblem(t *testing.T) *Problem {
	t.Helper()
	hours := 1000.0
	fixed := map[string]float64{
		"F1": AdjustReliability(ExponentialFloat(0.0000014, hours), "parallel", 1, 2),
		"F2": AdjustReliability(ExponentialFloat(0.000000422819, hours), "series", 2, 2),
		"F3": AdjustReliability(ExponentialFloat(0.0000000971, hours), "parallel", 1, 6),
		"F4": AdjustReliability(ExponentialFloat(0.000000097119, hours), "series", 6, 6),
		"F5": AdjustReliability(ExponentialFloat(0.0000000242799159642973, hours), "series", 24, 24),
	}
	codes := []string{"F1", "F2", "F3", "F4", "F5"}
	slots := make([]Slot, 0, 15)
	for index := 0; index < 15; index++ {
		units := index%4 + 1
		candidates := make([]Candidate, 0, 3)
		for vendor := 0; vendor < 3; vendor++ {
			rate := float64(index+1) * 1e-7 * float64(vendor+1)
			candidates = append(candidates, Candidate{
				ComponentId: fmt.Sprintf("CM-%02d-%d", index, vendor),
				FailureRate: rate,
				UnitCost:    1e6 * float64(index+1) * float64(3-vendor),
				Reliability: AdjustReliability(ExponentialFloat(rate, hours), "series", units, units),
			})
		}
		code := fmt.Sprintf("X%02d", index)
		codes = append(codes, code)
		slots = append(slots, Slot{FormulaCode: code, Units: units, LockedIndex: -1, Candidates: candidates})
	}
	total, err := Compile(strings.Join(codes, "*"))
	require.NoError(t, err)
	return &Problem{Slots: slots, FixedValues: fixed, FixedCost: 990638400, Total: total}
}

func TestProblemReproducesTheResearchCaseToEightDecimals(t *testing.T) {
	problem := researchProblem(t)
	genome := make([]int, 15)
	for index := range genome {
		genome[index] = index % 3
	}
	reliability, cost := problem.Evaluate(genome)
	assert.InDelta(t, 0.9357442965783682, reliability, 1e-12)
	assert.InDelta(t, 1530638400, cost, 1e-4)
	objective := Objective{Mode: ModeBudgetFloor, MaxBudget: 2666000000, TargetReliability: 0.844, WeightCost: 0.9, WeightReliability: 0.1}
	assert.InDelta(t, -0.4231452102521407, objective.Fitness(reliability, cost), 1e-12)
	assert.True(t, objective.Feasible(reliability, cost))
}

func TestProblemWalksLayersBottomUp(t *testing.T) {
	inner, err := Compile("(C1*C2)")
	require.NoError(t, err)
	outer, err := Compile("1-(1-HS1)*(1-C3)")
	require.NoError(t, err)
	total, err := Compile("HS2")
	require.NoError(t, err)
	problem := &Problem{
		Slots: []Slot{
			{FormulaCode: "C1", Units: 1, LockedIndex: -1, Candidates: []Candidate{{Reliability: 0.9, UnitCost: 10}}},
			{FormulaCode: "C2", Units: 2, LockedIndex: -1, Candidates: []Candidate{{Reliability: 0.8, UnitCost: 5}}},
		},
		FixedValues: map[string]float64{"C3": 0.5},
		FixedCost:   100,
		Layers:      []Layer{{Code: "HS1", Formula: inner}, {Code: "HS2", Formula: outer}},
		Total:       total,
	}
	reliability, cost := problem.Evaluate([]int{0, 0})
	assert.InDelta(t, 1-(1-0.72)*0.5, reliability, 1e-12)
	assert.InDelta(t, 120, cost, 1e-12)
	require.NoError(t, problem.Validate())
}

func TestValidateNamesWhatIsWrong(t *testing.T) {
	assert.ErrorContains(t, (&Problem{}).Validate(), "no formula")
	total, err := Compile("C1")
	require.NoError(t, err)
	assert.ErrorContains(t, (&Problem{Total: total}).Validate(), "more than one compatible vendor")
}
