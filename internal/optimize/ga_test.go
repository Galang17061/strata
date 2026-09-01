package optimize

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBudgetFloorFitnessMatchesTheResearchFormula(t *testing.T) {
	objective := Objective{Mode: ModeBudgetFloor, MaxBudget: 2666000000, TargetReliability: 0.844, WeightCost: 0.9, WeightReliability: 0.1}
	assert.InDelta(t, -0.5851687921980495, objective.Fitness(0.9, 2000000000), 1e-12)
	assert.True(t, objective.Feasible(0.9, 2000000000))
	assert.InDelta(t, -1.1020345086271567, objective.Fitness(0.8, 3000000000), 1e-12)
	assert.False(t, objective.Feasible(0.8, 3000000000))
}

func TestBudgetFitnessPunishesOverspendOnly(t *testing.T) {
	objective := Objective{Mode: ModeBudget, MaxBudget: 1000}
	assert.InDelta(t, 0.95, objective.Fitness(0.95, 800), 1e-12)
	assert.InDelta(t, 0.95-0.2, objective.Fitness(0.95, 1200), 1e-12)
	assert.False(t, objective.Feasible(0.95, 1200))
	free := Objective{Mode: ModeReliability}
	assert.InDelta(t, 0.5, free.Fitness(0.5, 999999999), 1e-12)
	assert.True(t, free.Feasible(0.5, 999999999))
}

func toyEvaluate(genome []int) (float64, float64) {
	reliabilities := []float64{0.90, 0.95, 0.99}
	costs := []float64{100, 200, 400}
	reliability := 1.0
	cost := 0.0
	for _, gene := range genome {
		reliability *= reliabilities[gene]
		cost += costs[gene]
	}
	return reliability, cost
}

func TestSeededSearchFindsTheKnownBestLineUp(t *testing.T) {
	options := Options{PopulationSize: 60, MaxGenerations: 200, CrossoverProbability: 0.9, MutationProbability: 0.4, Seed: 42}
	unconstrained, err := Run([]int{3, 3, 3}, nil, toyEvaluate, Objective{Mode: ModeReliability}, options)
	require.NoError(t, err)
	assert.Equal(t, []int{2, 2, 2}, unconstrained.Best)
	assert.InDelta(t, 0.970299, unconstrained.BestReliability, 1e-12)

	budgeted, err := Run([]int{3, 3, 3}, nil, toyEvaluate, Objective{Mode: ModeBudget, MaxBudget: 700}, options)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 1, 1}, budgeted.Best)
	assert.InDelta(t, 0.857375, budgeted.BestReliability, 1e-12)
	assert.True(t, budgeted.Feasible)
	assert.NotEmpty(t, budgeted.History)

	again, err := Run([]int{3, 3, 3}, nil, toyEvaluate, Objective{Mode: ModeBudget, MaxBudget: 700}, options)
	require.NoError(t, err)
	assert.Equal(t, budgeted.Best, again.Best)
	assert.Equal(t, budgeted.Generations, again.Generations)
}

func TestALockedSlotStaysWhereTheUserPinnedIt(t *testing.T) {
	options := Options{PopulationSize: 60, MaxGenerations: 200, CrossoverProbability: 0.9, MutationProbability: 0.4, Seed: 7}
	locked, err := Run([]int{3, 3, 3}, []int{2, -1, -1}, toyEvaluate, Objective{Mode: ModeBudget, MaxBudget: 700}, options)
	require.NoError(t, err)
	assert.Equal(t, 2, locked.Best[0])
	assert.InDelta(t, 0.84645, locked.BestReliability, 1e-12)
	assert.InDelta(t, 700, locked.BestCost, 1e-12)
}

func TestRunRefusesAnEmptyOrImpossibleProblem(t *testing.T) {
	_, err := Run(nil, nil, toyEvaluate, Objective{Mode: ModeReliability}, Options{PopulationSize: 4, MaxGenerations: 4})
	assert.Error(t, err)
	_, err = Run([]int{0}, nil, toyEvaluate, Objective{Mode: ModeReliability}, Options{PopulationSize: 4, MaxGenerations: 4})
	assert.Error(t, err)
}
