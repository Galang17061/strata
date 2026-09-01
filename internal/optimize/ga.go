package optimize

import (
	"errors"
	"math/rand"
)

type Mode int

const (
	ModeReliability Mode = 1
	ModeBudget      Mode = 2
	ModeBudgetFloor Mode = 3
)

type Objective struct {
	Mode              Mode
	MaxBudget         float64
	TargetReliability float64
	WeightCost        float64
	WeightReliability float64
}

func (o Objective) Fitness(reliability, cost float64) float64 {
	switch o.Mode {
	case ModeBudget:
		penalty := 0.0
		if o.MaxBudget > 0 && cost > o.MaxBudget {
			penalty = (cost - o.MaxBudget) / o.MaxBudget
		}
		return reliability - penalty
	case ModeBudgetFloor:
		budget := o.MaxBudget
		if budget <= 0 {
			budget = 1
		}
		penalty := 0.0
		if cost > o.MaxBudget && o.MaxBudget > 0 {
			penalty += (cost - o.MaxBudget) / o.MaxBudget
		}
		if reliability < o.TargetReliability {
			penalty += o.TargetReliability - reliability
		}
		return -(o.WeightCost*(cost/budget) - o.WeightReliability*reliability) - penalty
	default:
		return reliability
	}
}

func (o Objective) Feasible(reliability, cost float64) bool {
	switch o.Mode {
	case ModeBudget:
		return o.MaxBudget <= 0 || cost <= o.MaxBudget
	case ModeBudgetFloor:
		return (o.MaxBudget <= 0 || cost <= o.MaxBudget) && reliability >= o.TargetReliability
	default:
		return true
	}
}

type Evaluate func(genome []int) (reliability, cost float64)

type Options struct {
	PopulationSize       int
	MaxGenerations       int
	CrossoverProbability float64
	MutationProbability  float64
	StallGenerations     int
	Seed                 int64
}

type GenerationPoint struct {
	Generation      int     `json:"generation"`
	BestFitness     float64 `json:"bestFitness"`
	BestReliability float64 `json:"bestReliability"`
	BestCost        float64 `json:"bestCost"`
}

type Result struct {
	Best            []int
	BestFitness     float64
	BestReliability float64
	BestCost        float64
	Feasible        bool
	Generations     int
	History         []GenerationPoint
	Seed            int64
}

type individual struct {
	genome      []int
	fitness     float64
	reliability float64
	cost        float64
}

func Run(ranges []int, locked []int, evaluate Evaluate, objective Objective, options Options) (Result, error) {
	if len(ranges) == 0 {
		return Result{}, errors.New("Nothing to optimise: every slot is fixed")
	}
	for _, span := range ranges {
		if span < 1 {
			return Result{}, errors.New("Every slot needs at least one candidate")
		}
	}
	rng := rand.New(rand.NewSource(options.Seed))
	stall := options.StallGenerations
	if stall <= 0 {
		stall = 25
	}
	randomGenome := func() []int {
		genome := make([]int, len(ranges))
		for index, span := range ranges {
			if locked != nil && locked[index] >= 0 {
				genome[index] = locked[index]
			} else {
				genome[index] = rng.Intn(span)
			}
		}
		return genome
	}
	score := func(genome []int) individual {
		reliability, cost := evaluate(genome)
		return individual{genome: genome, fitness: objective.Fitness(reliability, cost), reliability: reliability, cost: cost}
	}
	population := make([]individual, options.PopulationSize)
	for index := range population {
		population[index] = score(randomGenome())
	}
	bestOf := func(people []individual) individual {
		best := people[0]
		for _, person := range people[1:] {
			if person.fitness > best.fitness {
				best = person
			}
		}
		return best
	}
	clone := func(person individual) individual {
		genome := make([]int, len(person.genome))
		copy(genome, person.genome)
		return individual{genome: genome, fitness: person.fitness, reliability: person.reliability, cost: person.cost}
	}
	roulette := func(people []individual) individual {
		minimum := people[0].fitness
		for _, person := range people {
			if person.fitness < minimum {
				minimum = person.fitness
			}
		}
		shift := 0.0
		if minimum < 0 {
			shift = -minimum + 0.1
		}
		total := 0.0
		for _, person := range people {
			total += person.fitness + shift
		}
		if total <= 0 {
			return people[rng.Intn(len(people))]
		}
		point := rng.Float64() * total
		running := 0.0
		for _, person := range people {
			running += person.fitness + shift
			if running >= point {
				return person
			}
		}
		return people[len(people)-1]
	}
	crossover := func(first, second individual) ([]int, []int) {
		childA := make([]int, len(first.genome))
		childB := make([]int, len(first.genome))
		point := 1
		if len(first.genome) > 1 {
			point = 1 + rng.Intn(len(first.genome)-1)
		}
		for index := range first.genome {
			if index < point {
				childA[index] = first.genome[index]
				childB[index] = second.genome[index]
			} else {
				childA[index] = second.genome[index]
				childB[index] = first.genome[index]
			}
		}
		return childA, childB
	}
	mutate := func(genome []int) {
		for index, span := range ranges {
			if locked != nil && locked[index] >= 0 {
				genome[index] = locked[index]
				continue
			}
			if rng.Float64() < options.MutationProbability {
				genome[index] = rng.Intn(span)
			}
		}
	}
	bestEver := clone(bestOf(population))
	result := Result{Seed: options.Seed}
	stalled := 0
	for generation := 0; generation < options.MaxGenerations; generation++ {
		generationBest := bestOf(population)
		if generationBest.fitness > bestEver.fitness {
			bestEver = clone(generationBest)
			stalled = 0
		} else {
			stalled++
		}
		result.History = append(result.History, GenerationPoint{
			Generation:      generation + 1,
			BestFitness:     bestEver.fitness,
			BestReliability: bestEver.reliability,
			BestCost:        bestEver.cost,
		})
		result.Generations = generation + 1
		if stalled >= stall {
			break
		}
		next := make([]individual, 0, options.PopulationSize)
		next = append(next, clone(bestEver))
		for len(next) < options.PopulationSize {
			parentA := roulette(population)
			parentB := roulette(population)
			var childA, childB []int
			if rng.Float64() < options.CrossoverProbability {
				childA, childB = crossover(parentA, parentB)
			} else {
				childA = append([]int{}, parentA.genome...)
				childB = append([]int{}, parentB.genome...)
			}
			mutate(childA)
			mutate(childB)
			next = append(next, score(childA))
			if len(next) < options.PopulationSize {
				next = append(next, score(childB))
			}
		}
		population = next
	}
	result.Best = bestEver.genome
	result.BestFitness = bestEver.fitness
	result.BestReliability = bestEver.reliability
	result.BestCost = bestEver.cost
	result.Feasible = objective.Feasible(bestEver.reliability, bestEver.cost)
	return result, nil
}
