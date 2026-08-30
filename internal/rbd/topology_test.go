package rbd

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Galang17061/strata-api/internal/reliability"
)

const referenceLargest = 2147483647

type referenceRandom struct {
	seeds [56]int32
	next  int
	nextp int
}

func newReferenceRandom(seed int32) *referenceRandom {
	r := &referenceRandom{nextp: 21}
	if seed < 0 {
		seed = -seed
	}
	mj := int32(161803398) - seed
	r.seeds[55] = mj
	mk := int32(1)
	slot := 0
	for i := 1; i < 55; i++ {
		slot += 21
		if slot >= 55 {
			slot -= 55
		}
		r.seeds[slot] = mk
		mk = mj - mk
		if mk < 0 {
			mk += referenceLargest
		}
		mj = r.seeds[slot]
	}
	for round := 1; round < 5; round++ {
		for i := 1; i < 56; i++ {
			other := i + 30
			if other >= 55 {
				other -= 55
			}
			r.seeds[i] -= r.seeds[1+other]
			if r.seeds[i] < 0 {
				r.seeds[i] += referenceLargest
			}
		}
	}
	return r
}

func (r *referenceRandom) sample() int32 {
	next := r.next + 1
	if next >= 56 {
		next = 1
	}
	nextp := r.nextp + 1
	if nextp >= 56 {
		nextp = 1
	}
	value := r.seeds[next] - r.seeds[nextp]
	if value == referenceLargest {
		value--
	}
	if value < 0 {
		value += referenceLargest
	}
	r.seeds[next] = value
	r.next = next
	r.nextp = nextp
	return value
}

func (r *referenceRandom) nextFloat() float64 {
	return float64(r.sample()) * (1.0 / float64(referenceLargest))
}

func (r *referenceRandom) reliability() decimal.Decimal {
	scaled := float64(r.nextFloat() * 0.49)
	return reliability.MustFromFloat(scaled + 0.50)
}

type drawnValues map[string]decimal.Decimal

var one = decimal.NewFromInt(1)

func (v drawnValues) of(letter string) decimal.Decimal {
	return v["R"+letter]
}

func (v drawnValues) series(letters ...string) decimal.Decimal {
	parts := make([]decimal.Decimal, 0, len(letters))
	for _, letter := range letters {
		parts = append(parts, v.of(letter))
	}
	return allOf(parts...)
}

func (v drawnValues) parallel(letters ...string) decimal.Decimal {
	parts := make([]decimal.Decimal, 0, len(letters))
	for _, letter := range letters {
		parts = append(parts, v.of(letter))
	}
	return anyOf(parts...)
}

func allOf(parts ...decimal.Decimal) decimal.Decimal {
	result := one
	for _, part := range parts {
		result = reliability.Mul(result, part)
	}
	return result
}

func anyOf(parts ...decimal.Decimal) decimal.Decimal {
	failing := one
	for _, part := range parts {
		failing = reliability.Mul(failing, reliability.Sub(one, part))
	}
	return reliability.Sub(one, failing)
}

func nodeName(token string) string {
	if len(token) == 1 {
		return "node-" + token
	}
	return token
}

func mesh(sources, targets string) string {
	pairs := []string{}
	for _, source := range strings.Fields(sources) {
		for _, target := range strings.Fields(targets) {
			pairs = append(pairs, source+">"+target)
		}
	}
	return strings.Join(pairs, " ")
}

func wiredLetters(spec string) map[string]bool {
	wired := map[string]bool{}
	for _, part := range strings.Fields(spec) {
		for _, token := range strings.SplitN(part, ">", 2) {
			if len(token) == 1 {
				wired[token] = true
			}
		}
	}
	return wired
}

func componentEdges(nodes map[string]string, spec string) []reliability.Edge {
	edges := []reliability.Edge{}
	for _, part := range strings.Fields(spec) {
		pair := strings.SplitN(part, ">", 2)
		sourceCode, sourceKnown := nodes[nodeName(pair[0])]
		targetCode, targetKnown := nodes[nodeName(pair[1])]
		if sourceKnown && targetKnown {
			edges = append(edges, reliability.Edge{SourceId: sourceCode, TargetId: targetCode})
		}
	}
	return edges
}

type topologyCase struct {
	name      string
	draws     string
	virtual   string
	edges     string
	formula   string
	reference string
	expected  func(v drawnValues) decimal.Decimal
}

func TestReferenceRandomReproducesTheSeededSequence(t *testing.T) {
	random := newReferenceRandom(42)
	expected := []string{"0.827372168296656", "0.569044576203006", "0.561503961832032", "0.756154495252368", "0.582532769843253", "0.628670410890444"}
	for _, text := range expected {
		assert.Equal(t, text, random.reliability().String())
	}
}

func runTopologyCases(t *testing.T, cases []topologyCase) {
	tolerance := decimal.RequireFromString("0.00000001")
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			random := newReferenceRandom(42)
			values := drawnValues{}
			nodes := map[string]string{}
			wired := wiredLetters(c.edges)
			letters := strings.Fields(c.draws)
			for _, letter := range letters {
				values["R"+letter] = random.reliability()
				if wired[strings.ToLower(letter)] {
					nodes["node-"+strings.ToLower(letter)] = "R" + letter
				}
			}
			for _, extra := range strings.Fields(c.virtual) {
				pair := strings.SplitN(extra, ":", 2)
				nodes[pair[0]] = pair[1]
			}
			formula := reliability.StripVirtualCodes(reliability.GenerateFormula(componentEdges(nodes, c.edges)))
			formula = strings.ReplaceAll(formula, "*R_OUT", "")
			assert.Equal(t, c.formula, formula)
			lookup := newLookup()
			for _, letter := range letters {
				if !wired[strings.ToLower(letter)] {
					continue
				}
				value := values["R"+letter]
				assert.Contains(t, formula, "R"+letter, formula)
				lookup.set("R"+letter, &value)
			}
			actual, err := lookup.evaluate(formula, false)
			require.NoError(t, err, formula)
			expected := c.expected(values)
			assert.True(t, actual.Sub(expected).Abs().LessThanOrEqual(tolerance), "%s: expected %s, got %s from %s", c.name, expected.StringFixed(10), actual.StringFixed(10), formula)
			reference := decimal.RequireFromString(c.reference)
			assert.True(t, actual.Sub(reference).Abs().LessThanOrEqual(tolerance), "%s: reference %s, got %s", c.name, c.reference, actual.StringFixed(15))
		})
	}
}

func TestSeriesParallelAndMixedBlocksScoreLikeTheMath(t *testing.T) {
	runTopologyCases(t, []topologyCase{
		{"fiveParallelThenSeriesThenFourParallel", "A B C D E F G H I J K", "", "a>f b>f c>f d>f e>f f>g g>h g>i g>j g>k",
			"(1-(1-RA)*(1-RB)*(1-RC)*(1-RD)*(1-RE))*RG*(1-(1-RH)*(1-RI)*(1-RJ)*(1-RK))*RF", "0.5330006376", func(v drawnValues) decimal.Decimal {
				return allOf(v.parallel("A", "B", "C", "D", "E"), v.of("F"), v.of("G"), v.parallel("H", "I", "J", "K"))
			}},
		{"fiveSeries", "A B C D E", "", "a>b b>c c>d d>e e>node-end", "(RA*RB*RC*RD*RE)", "0.1164477014", func(v drawnValues) decimal.Decimal {
			return v.series("A", "B", "C", "D", "E")
		}},
		{"twoParallelBetweenStartAndEnd", "A B", "start:START node-end:END", "start>a start>b a>node-end b>node-end", "((1-(1-RA)*(1-RB)))", "0.9256050996", func(v drawnValues) decimal.Decimal {
			return v.parallel("A", "B")
		}},
		{"twoSeries", "A B", "", "a>b b>node-end", "(RA*RB)", "0.4708116449", func(v drawnValues) decimal.Decimal {
			return v.series("A", "B")
		}},
		{"singleBetweenStartAndEnd", "A", "start:START end:END", "start>a a>end", "(RA)", "0.8273721683", func(v drawnValues) decimal.Decimal {
			return v.of("A")
		}},
		{"threeSeries", "A B C", "", "a>b b>c c>node-end", "(RA*RB*RC)", "0.2643626039", func(v drawnValues) decimal.Decimal {
			return v.series("A", "B", "C")
		}},
		{"threeParallelBetweenStartAndEnd", "A B C", "start:START end:END", "start>a start>b start>c a>end b>end c>end", "((1-(1-RA)*(1-RB)*(1-RC)))", "0.9673781309", func(v drawnValues) decimal.Decimal {
			return v.parallel("A", "B", "C")
		}},
		{"parallelThenSeries", "A B C", "", "a>c b>c c>node-end", "(1-(1-RA)*(1-RB))*RC", "0.5197309305", func(v drawnValues) decimal.Decimal {
			return allOf(v.parallel("A", "B"), v.of("C"))
		}},
		{"fiveWayParallel", "A B C D E", "start:START end:END", "start>a start>b start>c start>d start>e a>end b>end c>end d>end e>end",
			"((1-(1-RA)*(1-RB)*(1-RC)*(1-RD)*(1-RE)))", "0.9966791750", func(v drawnValues) decimal.Decimal {
				return v.parallel("A", "B", "C", "D", "E")
			}},
		{"seriesThenFourParallel", "A B C D E", "", "a>b a>c a>d a>e b>node-end c>node-end d>node-end e>node-end", "RA*(1-(1-RB)*(1-RC)*(1-RD)*(1-RE))", "0.8114560896", func(v drawnValues) decimal.Decimal {
			return allOf(v.of("A"), v.parallel("B", "C", "D", "E"))
		}},
		{"fourParallelThenSeries", "A B C D E", "", "a>e b>e c>e d>e", "(1-(1-RA)*(1-RB)*(1-RC)*(1-RD))*RE", "0.5778988987", func(v drawnValues) decimal.Decimal {
			return allOf(v.parallel("A", "B", "C", "D"), v.of("E"))
		}},
		{"seriesParallelSeries", "A B C D E", "", "a>b a>c a>d b>e c>e d>e e>node-end", "(RA*(1-(1-RB)*(1-RC)*(1-RD))*RE)", "0.4597621430", func(v drawnValues) decimal.Decimal {
			return allOf(v.of("A"), v.parallel("B", "C", "D"), v.of("E"))
		}},
		{"threeParallelThenTwoSeries", "A B C D E", "", "a>d b>d c>d d>e e>node-end", "(1-(1-RA)*(1-RB)*(1-RC))*RE*RD", "0.4261153360", func(v drawnValues) decimal.Decimal {
			return allOf(v.parallel("A", "B", "C"), v.of("D"), v.of("E"))
		}},
		{"twoParallelThenThreeSeries", "A B C D E", "", "a>c b>c c>d d>e e>node-end", "(1-(1-RA)*(1-RB))*(RD*RE)*RC", "0.2289335607", func(v drawnValues) decimal.Decimal {
			return allOf(v.parallel("A", "B"), v.of("C"), v.of("D"), v.of("E"))
		}},
		{"seriesTwoParallelTwoSeriesKeepsTheOldScore", "A B C D E", "", "a>b a>c b>d c>d d>e", "(RA*(1-(1-RB)*(1-RC))*RD)*RE*RD", "0.2235002751", func(v drawnValues) decimal.Decimal {
			return allOf(v.of("A"), v.parallel("B", "C"), v.of("D"), v.of("E"), v.of("D"))
		}},
		{"twoSeriesParallelSeries", "A B C D E F", "", "a>b b>c b>d c>e d>e", "(RA*RB*(1-(1-RC)*(1-RD))*RE)", "0.2449375387", func(v drawnValues) decimal.Decimal {
			return allOf(v.of("A"), v.of("B"), v.parallel("C", "D"), v.of("E"))
		}},
		{"threeSeriesThenTwoParallel", "A B C D E", "", "a>b b>c c>d c>e", "(RA*RB*RC*(1-(1-RD)*(1-RE)))", "0.2374511497", func(v drawnValues) decimal.Decimal {
			return allOf(v.of("A"), v.of("B"), v.of("C"), v.parallel("D", "E"))
		}},
		{"parallelSeriesParallel", "A B C D E", "", "a>c b>c c>d c>e", "(1-(1-RA)*(1-RB))*RC*(1-(1-RD)*(1-RE))", "0.4668236173", func(v drawnValues) decimal.Decimal {
			return allOf(v.parallel("A", "B"), v.of("C"), v.parallel("D", "E"))
		}},
		{"twoSeriesThenThreeParallel", "A B C D E", "", "a>b b>c b>d b>e", "(RA*RB*(1-(1-RC)*(1-RD)*(1-RE)))", "0.4497956471", func(v drawnValues) decimal.Decimal {
			return allOf(v.of("A"), v.of("B"), v.parallel("C", "D", "E"))
		}},
		{"twoSeriesThenTwoParallel", "A B C D", "", "a>b b>c b>d", "(RA*RB*(1-(1-RC)*(1-RD)))", "0.4204699743", func(v drawnValues) decimal.Decimal {
			return allOf(v.of("A"), v.of("B"), v.parallel("C", "D"))
		}},
		{"seriesThenThreeParallel", "A B C D", "", "a>b a>c a>d", "RA*(1-(1-RB)*(1-RC)*(1-RD))", "0.7892468317", func(v drawnValues) decimal.Decimal {
			return allOf(v.of("A"), v.parallel("B", "C", "D"))
		}},
		{"threeParallelThenSeries", "A B C D", "", "a>d b>d c>d", "(1-(1-RA)*(1-RB)*(1-RC))*RD", "0.7314873223", func(v drawnValues) decimal.Decimal {
			return allOf(v.parallel("A", "B", "C"), v.of("D"))
		}},
		{"seriesParallelSeriesOfFour", "A B C D", "", "a>b a>c b>d c>d", "(RA*(1-(1-RB)*(1-RC))*RD)", "0.5073961440", func(v drawnValues) decimal.Decimal {
			return allOf(v.of("A"), v.parallel("B", "C"), v.of("D"))
		}},
		{"parallelThenTwoSeries", "A B C D", "", "a>c b>c c>d", "(1-(1-RA)*(1-RB))*RD*RC", "0.3929968794", func(v drawnValues) decimal.Decimal {
			return allOf(v.parallel("A", "B"), v.of("C"), v.of("D"))
		}},
		{"fourSeries", "A B C D", "", "a>b b>c c>d", "(RA*RB*RC*RD)", "0.1998989713", func(v drawnValues) decimal.Decimal {
			return v.series("A", "B", "C", "D")
		}},
		{"oneThenThreeParallel", "A B C D", "", "a>b a>c a>d", "RA*(1-(1-RB)*(1-RC)*(1-RD))", "0.7892468317", func(v drawnValues) decimal.Decimal {
			return allOf(v.of("A"), v.parallel("B", "C", "D"))
		}},
	})
}
