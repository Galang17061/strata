package reliability

import (
	"errors"
	"math"
	"math/rand"
	"sort"
)

const (
	SimulationTrialsDefault = 10000
	SimulationTrialsCeiling = 500000
	SimulationCurveDefault  = 40
)

type SimulationPart struct {
	Code            string
	Distribution    string
	FailureRate     float64
	Scale           float64
	Shape           float64
	AllowedFailures int
	Active          int
	Total           int
}

type SimulationInput struct {
	Formula      string
	Parts        []SimulationPart
	MissionHours float64
	Trials       int
	Seed         int64
	CurvePoints  int
}

type SimulationPoint struct {
	Hours       float64 `json:"hours"`
	Reliability float64 `json:"reliability"`
}

type SimulationCulprit struct {
	Code  string  `json:"code"`
	Share float64 `json:"share"`
}

type SimulationOutput struct {
	Trials       int
	MissionHours float64
	Seed         int64
	Blocks       int
	Survivors    int
	Reliability  float64
	LowerBound   float64
	UpperBound   float64
	MeanLife     float64
	MedianLife   float64
	B10Life      float64
	Curve        []SimulationPoint
	Culprits     []SimulationCulprit
	Unbacked     []string
	NeverFails   []string
	Immortal     int
}

type simulationBlock struct {
	slot            int
	code            string
	distribution    string
	rate            float64
	scale           float64
	shape           float64
	allowedFailures int
	active          int
	total           int
	lives           []float64
}

type simulationEvent struct {
	hours float64
	block int
}

func Simulate(input SimulationInput) (SimulationOutput, error) {
	if input.MissionHours <= 0 {
		return SimulationOutput{}, errors.New("The mission time must be greater than zero.")
	}
	if len(input.Parts) == 0 {
		return SimulationOutput{}, errors.New("The simulation needs at least one component.")
	}
	structure, err := CompileStructure(input.Formula)
	if err != nil {
		return SimulationOutput{}, err
	}
	blocks, err := prepareBlocks(structure, input.Parts)
	if err != nil {
		return SimulationOutput{}, err
	}
	trials := input.Trials
	if trials < 1 {
		trials = SimulationTrialsDefault
	}
	if trials > SimulationTrialsCeiling {
		trials = SimulationTrialsCeiling
	}
	seed := input.Seed
	if seed == 0 {
		seed = 1
	}
	stream := rand.New(rand.NewSource(seed))

	slots := make([]float64, structure.Slots())
	events := make([]simulationEvent, len(blocks))
	lifetimes := make([]float64, 0, trials)
	blame := make([]int, len(blocks))
	survivors := 0

	for trial := 0; trial < trials; trial++ {
		for i := range slots {
			slots[i] = 1
		}
		for i := range blocks {
			events[i] = simulationEvent{hours: blocks[i].sample(stream), block: i}
		}
		sort.Slice(events, func(a, b int) bool { return events[a].hours < events[b].hours })
		failure := math.Inf(1)
		culprit := -1
		for _, event := range events {
			slots[blocks[event.block].slot] = 0
			if !structure.Standing(slots) {
				failure = event.hours
				culprit = event.block
				break
			}
		}
		if culprit >= 0 {
			blame[culprit]++
		}
		if failure > input.MissionHours {
			survivors++
		}
		lifetimes = append(lifetimes, failure)
	}

	sort.Float64s(lifetimes)
	lower, upper := wilsonBounds(survivors, trials)
	output := SimulationOutput{
		Trials:       trials,
		MissionHours: input.MissionHours,
		Seed:         seed,
		Blocks:       len(blocks),
		Survivors:    survivors,
		Reliability:  float64(survivors) / float64(trials),
		LowerBound:   lower,
		UpperBound:   upper,
		MeanLife:     meanOfFinite(lifetimes),
		MedianLife:   quantile(lifetimes, 0.5),
		B10Life:      quantile(lifetimes, 0.1),
		Curve:        buildCurve(lifetimes, input.MissionHours, input.CurvePoints),
		Culprits:     rankCulprits(blocks, blame, trials),
		Unbacked:     unbackedCodes(structure, blocks),
		NeverFails:   neverFailingCodes(blocks),
	}
	output.Immortal = len(output.NeverFails)
	return output, nil
}

func unbackedCodes(structure *Structure, blocks []simulationBlock) []string {
	backed := map[string]bool{}
	for _, block := range blocks {
		backed[block.code] = true
	}
	loose := []string{}
	for _, code := range structure.Codes() {
		if !backed[code] && !IsVirtualCode(code) {
			loose = append(loose, code)
		}
	}
	return loose
}

func neverFailingCodes(blocks []simulationBlock) []string {
	immortal := []string{}
	for i := range blocks {
		if !blocks[i].mortal() {
			immortal = append(immortal, blocks[i].code)
		}
	}
	return immortal
}

func prepareBlocks(structure *Structure, parts []SimulationPart) ([]simulationBlock, error) {
	blocks := make([]simulationBlock, 0, len(parts))
	for _, part := range parts {
		slot, found := structure.Slot(part.Code)
		if !found {
			continue
		}
		total := part.Total
		if total < 1 {
			total = 1
		}
		active := part.Active
		if active < 1 || active > total {
			active = total
		}
		block := simulationBlock{
			slot:            slot,
			code:            part.Code,
			distribution:    part.Distribution,
			rate:            part.FailureRate,
			scale:           part.Scale,
			shape:           part.Shape,
			allowedFailures: part.AllowedFailures,
			active:          active,
			total:           total,
		}
		if total > 1 {
			block.lives = make([]float64, total)
		}
		blocks = append(blocks, block)
	}
	if len(blocks) == 0 {
		return nil, errors.New("None of the components appear in the system formula.")
	}
	return blocks, nil
}

func (b *simulationBlock) sample(stream *rand.Rand) float64 {
	if b.total == 1 {
		return b.sampleUnit(stream)
	}
	for i := range b.lives {
		b.lives[i] = b.sampleUnit(stream)
	}
	sort.Float64s(b.lives)
	return b.lives[b.total-b.active]
}

func (b *simulationBlock) mortal() bool {
	if b.distribution == "weibull" {
		return b.scale > 0 && b.shape > 0
	}
	return b.rate > 0
}

func (b *simulationBlock) sampleUnit(stream *rand.Rand) float64 {
	switch b.distribution {
	case "weibull":
		if b.scale <= 0 || b.shape <= 0 {
			return math.Inf(1)
		}
		return b.scale * math.Pow(-math.Log(uniform(stream)), 1/b.shape)
	case "poisson":
		if b.rate <= 0 {
			return math.Inf(1)
		}
		waits := b.allowedFailures + 1
		total := 0.0
		for i := 0; i < waits; i++ {
			total += -math.Log(uniform(stream)) / b.rate
		}
		return total
	default:
		if b.rate <= 0 {
			return math.Inf(1)
		}
		return -math.Log(uniform(stream)) / b.rate
	}
}

func uniform(stream *rand.Rand) float64 {
	for {
		value := stream.Float64()
		if value > 0 {
			return value
		}
	}
}

func wilsonBounds(successes, trials int) (float64, float64) {
	if trials < 1 {
		return 0, 0
	}
	const z = 1.959963984540054
	count := float64(trials)
	share := float64(successes) / count
	denominator := 1 + z*z/count
	center := (share + z*z/(2*count)) / denominator
	margin := z * math.Sqrt(share*(1-share)/count+z*z/(4*count*count)) / denominator
	lower := center - margin
	upper := center + margin
	if lower < 0 {
		lower = 0
	}
	if upper > 1 {
		upper = 1
	}
	return lower, upper
}

func meanOfFinite(sorted []float64) float64 {
	total := 0.0
	counted := 0
	for _, value := range sorted {
		if math.IsInf(value, 1) {
			break
		}
		total += value
		counted++
	}
	if counted == 0 {
		return 0
	}
	return total / float64(counted)
}

func quantile(sorted []float64, share float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	position := share * float64(len(sorted)-1)
	lower := int(math.Floor(position))
	upper := int(math.Ceil(position))
	if math.IsInf(sorted[upper], 1) {
		return math.Inf(1)
	}
	if lower == upper {
		return sorted[lower]
	}
	weight := position - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

func buildCurve(sorted []float64, mission float64, points int) []SimulationPoint {
	if points < 2 {
		points = SimulationCurveDefault
	}
	if points > 200 {
		points = 200
	}
	curve := make([]SimulationPoint, 0, points+1)
	trials := len(sorted)
	if trials == 0 {
		return curve
	}
	for step := 0; step <= points; step++ {
		hours := mission * float64(step) / float64(points)
		failed := sort.SearchFloat64s(sorted, hours)
		for failed < trials && sorted[failed] <= hours {
			failed++
		}
		curve = append(curve, SimulationPoint{
			Hours:       hours,
			Reliability: float64(trials-failed) / float64(trials),
		})
	}
	return curve
}

func rankCulprits(blocks []simulationBlock, blame []int, trials int) []SimulationCulprit {
	culprits := make([]SimulationCulprit, 0, len(blocks))
	for i, count := range blame {
		if count == 0 {
			continue
		}
		culprits = append(culprits, SimulationCulprit{
			Code:  blocks[i].code,
			Share: float64(count) / float64(trials),
		})
	}
	sort.Slice(culprits, func(a, b int) bool {
		if culprits[a].Share == culprits[b].Share {
			return culprits[a].Code < culprits[b].Code
		}
		return culprits[a].Share > culprits[b].Share
	})
	return culprits
}
