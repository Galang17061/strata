package optimize

import (
	"errors"
	"math"
	"strings"
)

type Candidate struct {
	ComponentId string
	VendorId    string
	VendorName  string
	FailureRate float64
	UnitCost    float64
	Reliability float64
}

type Slot struct {
	SystemComponentId string
	ComponentName     string
	FormulaCode       string
	Units             int
	CurrentIndex      int
	LockedIndex       int
	Candidates        []Candidate
}

type Layer struct {
	Code    string
	Formula *CompiledFormula
}

type Problem struct {
	Slots       []Slot
	FixedValues map[string]float64
	FixedCost   float64
	Layers      []Layer
	Total       *CompiledFormula
	scratch     map[string]float64
}

func binomialFloat(n, k int) float64 {
	result := 1.0
	for index := 1; index <= k; index++ {
		result = result * float64(n-k+index) / float64(index)
	}
	return result
}

func kOutOfNFloat(reliability float64, k, n int) float64 {
	sum := 0.0
	for index := k; index <= n; index++ {
		sum += binomialFloat(n, index) * math.Pow(reliability, float64(index)) * math.Pow(1-reliability, float64(n-index))
	}
	return sum
}

func AdjustReliability(base float64, connectionType string, active, total int) float64 {
	kind := strings.ToLower(strings.TrimSpace(connectionType))
	if kind == "" {
		kind = "series"
	}
	if active < 1 {
		active = 1
	}
	if total < 1 {
		total = 1
	}
	switch {
	case kind == "series" || kind == "serial":
		return math.Pow(base, float64(total))
	case kind == "parallel":
		if active == 1 {
			return 1 - math.Pow(1-base, float64(total))
		}
		if total >= active {
			return kOutOfNFloat(base, active, total)
		}
		return base
	case strings.Contains(kind, "partial") || strings.Contains(kind, "parsial"):
		if active > 1 && total >= active && active != total {
			return kOutOfNFloat(base, active, total)
		}
		return base
	default:
		return base
	}
}

func ExponentialFloat(failureRate, hours float64) float64 {
	return math.Exp(-failureRate * hours)
}

func (p *Problem) Ranges() []int {
	ranges := make([]int, len(p.Slots))
	for index, slot := range p.Slots {
		ranges[index] = len(slot.Candidates)
	}
	return ranges
}

func (p *Problem) LockedIndexes() []int {
	locked := make([]int, len(p.Slots))
	for index, slot := range p.Slots {
		locked[index] = slot.LockedIndex
	}
	return locked
}

func (p *Problem) CurrentGenome() []int {
	genome := make([]int, len(p.Slots))
	for index, slot := range p.Slots {
		genome[index] = slot.CurrentIndex
	}
	return genome
}

func (p *Problem) Evaluate(genome []int) (float64, float64) {
	if p.scratch == nil {
		p.scratch = make(map[string]float64, len(p.FixedValues)+len(p.Slots)+len(p.Layers))
	}
	values := p.scratch
	for code, value := range p.FixedValues {
		values[code] = value
	}
	cost := p.FixedCost
	for index, slot := range p.Slots {
		candidate := slot.Candidates[genome[index]]
		values[slot.FormulaCode] = candidate.Reliability
		cost += candidate.UnitCost * float64(slot.Units)
	}
	for _, layer := range p.Layers {
		value, err := layer.Formula.Eval(values)
		if err != nil {
			return 0, cost
		}
		values[layer.Code] = value
	}
	total, err := p.Total.Eval(values)
	if err != nil {
		return 0, cost
	}
	return total, cost
}

func (p *Problem) Validate() error {
	if p.Total == nil {
		return errors.New("The system has no formula to evaluate")
	}
	if len(p.Slots) == 0 {
		return errors.New("No component in this system has more than one compatible vendor to choose from")
	}
	reliability, _ := p.Evaluate(p.CurrentGenome())
	if reliability <= 0 {
		return errors.New("The system formula could not be evaluated with the current values")
	}
	return nil
}
