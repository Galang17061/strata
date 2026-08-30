package rbd

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestMeshedStagesAndBranchingPathsScoreLikeTheMath(t *testing.T) {
	runTopologyCases(t, []topologyCase{
		{"completeMultiStageSystem", "A B C D E F G H I J K", "", "a>f b>f c>f d>f e>f f>g g>h g>i g>j g>k",
			"(1-(1-RA)*(1-RB)*(1-RC)*(1-RD)*(1-RE))*RG*(1-(1-RH)*(1-RI)*(1-RJ)*(1-RK))*RF", "0.5330006376", func(v drawnValues) decimal.Decimal {
				return allOf(v.parallel("A", "B", "C", "D", "E"), v.of("F"), v.of("G"), v.parallel("H", "I", "J", "K"))
			}},
		{"twoStageParallel", "A B C D E", "", "a>c a>d a>e b>c b>d b>e", "((1-(1-RA)*(1-RB))*(1-(1-RC)*(1-RD)*(1-RE)))", "0.8842881209", func(v drawnValues) decimal.Decimal {
			return allOf(v.parallel("A", "B"), v.parallel("C", "D", "E"))
		}},
		{"threeToTwoStageParallel", "A B C D E", "", "a>d a>e b>d b>e c>d c>e", "((1-(1-RA)*(1-RB)*(1-RC))*(1-(1-RD)*(1-RE)))", "0.8689014484", func(v drawnValues) decimal.Decimal {
			return allOf(v.parallel("A", "B", "C"), v.parallel("D", "E"))
		}},
		{"fourStageMeshParallel", "A B C D E F G H I J K L M N", "", mesh("a b c d e", "f g h i") + " " + mesh("f g h i", "j k l") + " " + mesh("j k l", "m n"),
			"((((1-(1-RA)*(1-RB)*(1-RC)*(1-RD)*(1-RE))*(1-(1-RF)*(1-RG)*(1-RH)*(1-RI)))*(1-(1-RJ)*(1-RK)*(1-RL)))*(1-(1-RM)*(1-RN)))", "0.8888099344", func(v drawnValues) decimal.Decimal {
				return allOf(v.parallel("A", "B", "C", "D", "E"), v.parallel("F", "G", "H", "I"), v.parallel("J", "K", "L"), v.parallel("M", "N"))
			}},
		{"seriesChainBesideSinglePath", "A B C D E", "node-output:R_OUT", "a>b b>c c>d d>node-output e>node-output", "(1-(1-(RA*RB*RC*RD))*(1-RE))", "0.6659840397", func(v drawnValues) decimal.Decimal {
			return anyOf(v.series("A", "B", "C", "D"), v.of("E"))
		}},
		{"seriesPairBesideTwoSinglePaths", "A B C D", "node-output:R_OUT", "a>b b>node-output c>node-output d>node-output", "(1-(1-(RA*RB))*(1-RC)*(1-RD))", "0.9434163829", func(v drawnValues) decimal.Decimal {
			return anyOf(v.series("A", "B"), v.of("C"), v.of("D"))
		}},
		{"realWorldSystem", "P O A B C D E F G H I J K L Q S M N", "", "p>o o>a o>b o>c o>d o>e " + mesh("a b c d e", "f g h i") + " " + mesh("f g h i", "j k l") + " j>q k>q l>q q>s s>m s>n",
			"(((((((RP*RO)*(1-(1-RA)*(1-RB)*(1-RC)*(1-RD)*(1-RE)))*(1-(1-RF)*(1-RG)*(1-RH)*(1-RI)))*(1-(1-RJ)*(1-RK)*(1-RL)))*RQ)*RS)*(1-(1-RM)*(1-RN)))", "0.171669752008382", func(v drawnValues) decimal.Decimal {
				return allOf(v.of("P"), v.of("O"), v.parallel("A", "B", "C", "D", "E"), v.parallel("F", "G", "H", "I"), v.parallel("J", "K", "L"), v.of("Q"), v.of("S"), v.parallel("M", "N"))
			}},
		{"mixedPathParallelSeriesParallel", "A B C D", "", "START>a a>b START>c b>d c>d d>END", "(1-(1-(RA*RB))*(1-RC))*RD", "0.580691115233294", func(v drawnValues) decimal.Decimal {
			return allOf(anyOf(v.series("A", "B"), v.of("C")), v.of("D"))
		}},
		{"seriesParallelConvergence", "A B C D E", "", "a>b b>e c>e d>e e>node-end", "(1-(1-(RA*RB))*(1-RC)*(1-RD))*RE", "0.549570958617217", func(v drawnValues) decimal.Decimal {
			return allOf(anyOf(v.series("A", "B"), v.of("C"), v.of("D")), v.of("E"))
		}},
		{"threeSeriesPathsConverging", "A B C D E F G H I J K L", "", "a>b b>c c>d d>g g>j c>e e>h h>j c>f f>i i>j j>k k>l",
			"(RA*RB*RC*(1-(1-(RD*RG))*(1-(RE*RH))*(1-(RF*RI))))*(RK*RL)*RJ", "0.0776907587", func(v drawnValues) decimal.Decimal {
				return allOf(v.series("A", "B", "C"), anyOf(v.series("D", "G"), v.series("E", "H"), v.series("F", "I")), v.series("J", "K", "L"))
			}},
		{"pathSeriesThenParallelTwice", "A B C D E F G H I J", "", "a>b a>c b>d c>e d>f e>f f>g f>h g>i h>j",
			"RA*(1-(1-(RB*RD))*(1-(RC*RE)))*RF*(1-(1-(RG*RI))*(1-(RH*RJ)))", "0.265585263010574", func(v drawnValues) decimal.Decimal {
				return allOf(v.of("A"), anyOf(v.series("B", "D"), v.series("C", "E")), v.of("F"), anyOf(v.series("G", "I"), v.series("H", "J")))
			}},
		{"fiveSeriesPathsInParallel", "A B C D E F G H I J", "", "a>f b>g c>h d>i e>j",
			"(1-(1-(RA*RF))*(1-(RB*RG))*(1-(RC*RH))*(1-(RD*RI))*(1-(RE*RJ)))", "0.960965919141272", func(v drawnValues) decimal.Decimal {
				return anyOf(v.series("A", "F"), v.series("B", "G"), v.series("C", "H"), v.series("D", "I"), v.series("E", "J"))
			}},
		{"seriesParallelSeriesParallel", "A B C D E F G H I J K", "", "a>c c>e b>d d>e e>f f>g g>h g>i g>j g>k",
			"(1-(1-(RA*RC))*(1-(RB*RD)))*(RF*RG*(1-(1-RH)*(1-RI)*(1-RJ)*(1-RK)))*RE", "0.216497125821503", func(v drawnValues) decimal.Decimal {
				return allOf(anyOf(v.series("A", "C"), v.series("B", "D")), v.series("E", "F", "G"), v.parallel("H", "I", "J", "K"))
			}},
	})
}
