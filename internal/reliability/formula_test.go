package reliability

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func edgesFromSpec(spec string) []Edge {
	edges := []Edge{}
	for _, part := range strings.Fields(spec) {
		pair := strings.SplitN(part, ">", 2)
		edges = append(edges, Edge{SourceId: pair[0], TargetId: pair[1]})
	}
	return edges
}

const wideSpec = "IN>P P>O O>A O>B O>C O>D O>E A>F A>G A>H A>I B>F B>G B>H B>I C>F C>G C>H C>I D>F D>G D>H D>I E>F E>G E>H E>I F>J F>K F>L G>J G>K G>L H>J H>K H>L I>J I>K I>L J>Q K>Q L>Q Q>R R>M R>N M>OUT N>OUT"

func TestGenerateFormulaMatchesReferenceOutputs(t *testing.T) {
	cases := []struct {
		name     string
		spec     string
		expected string
	}{
		{"series3", "A>B B>C", "(A*B*C)"},
		{"parallel2", "IN>A A>OUT IN>B B>OUT", "(IN*(1-(1-A)*(1-B))*OUT)"},
		{"parallel4", "IN>C20 C20>OUT IN>C21 C21>OUT IN>C22 C22>OUT IN>C23 C23>OUT", "(IN*(1-(1-C20)*(1-C21)*(1-C22)*(1-C23))*OUT)"},
		{"parThenSeries", "IN>C14 IN>C15 C14>C16 C15>C16 C16>OUT", "(IN*(1-(1-C14)*(1-C15))*C16)*OUT"},
		{"seriesThenPar", "C1>C2 C1>C3", "C1*(1-(1-C2)*(1-C3))"},
		{"mixed", "R1>R2 R3>R2 R2>R4", "(1-(1-R1)*(1-R3))*R4*R2"},
		{"twoStage", "A>C A>D B>C B>D", "((1-(1-A)*(1-B))*(1-(1-C)*(1-D)))"},
		{"multiStage", "A>D A>E B>D B>E C>D C>E D>F D>G D>H E>F E>G E>H F>X G>X H>X", "(1-(1-(A*(1-(1-D)*(1-E))*F))*(1-(B*(1-(1-D)*(1-E))*F))*(1-(C*(1-(1-D)*(1-E))*F)))*(D*(1-(1-F)*(1-G)*(1-H))*X)*(E*(1-(1-F)*(1-G)*(1-H))*X)"},
		{"single", "A>OUT", "(A*OUT)"},
		{"seriesVirtual", "INH-00000005>C1 C1>C2 C2>OUTH-00000005", "(INH-00000005*C1*C2*OUTH-00000005)"},
		{"parallelVirtual", "INH-00000001>C1 INH-00000001>C2 C1>OUTH-00000001 C2>OUTH-00000001", "(INH-00000001*(1-(1-C1)*(1-C2))*OUTH-00000001)"},
		{"fivePaths", "A>F B>G C>H D>I E>J", "(1-(1-(A*F))*(1-(B*G))*(1-(C*H))*(1-(D*I))*(1-(E*J)))"},
		{"r39", "A>C B>D C>E D>E E>F F>G G>H G>I G>J G>K", "(1-(1-(A*C))*(1-(B*D)))*(F*G*(1-(1-H)*(1-I)*(1-J)*(1-K)))*E"},
		{"diamond", "A>B A>C B>D C>D", "(A*(1-(1-B)*(1-C))*D)"},
		{"seriesParSeries", "IN>C1 C1>C2 C1>C3 C2>C4 C3>C4 C4>OUT", "(IN*C1*(1-(1-C2)*(1-C3))*C4*OUT)"},
		{"hsCodes", "INH-00000010>HSA1 HSA1>HSB1 HSB1>HSC1 HSC1>OUTH-00000010", "(INH-00000010*HSA1*HSB1*HSC1*OUTH-00000010)"},
		{"componentIdsSeries", "SCP-00000001>SCP-00000002 SCP-00000002>SCP-00000003", "(SCP-00000001*SCP-00000002*SCP-00000003)"},
		{"startEnd", "START>R1 R1>R2 R2>END", "(START*R1*R2*END)"},
		{"wideParallel", "IN>A IN>B IN>C IN>D IN>E A>OUT B>OUT C>OUT D>OUT E>OUT", "(IN*(1-(1-A)*(1-B)*(1-C)*(1-D)*(1-E))*OUT)"},
		{"wide", wideSpec, "(IN*P*O*(1-(1-A)*(1-B)*(1-C)*(1-D)*(1-E))*(1-(1-F)*(1-G)*(1-H)*(1-I))*(1-(1-J)*(1-K)*(1-L))*Q*R*(1-(1-M)*(1-N))*OUT)*(B*(1-(1-F)*(1-G)*(1-H)*(1-I))*J)*(C*(1-(1-F)*(1-G)*(1-H)*(1-I))*J)*(D*(1-(1-F)*(1-G)*(1-H)*(1-I))*J)*(E*(1-(1-F)*(1-G)*(1-H)*(1-I))*J)*(G*(1-(1-J)*(1-K)*(1-L))*Q)*(H*(1-(1-J)*(1-K)*(1-L))*Q)*(I*(1-(1-J)*(1-K)*(1-L))*Q)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, GenerateFormula(edgesFromSpec(c.spec)))
		})
	}
}

func TestStripVirtualCodesLeavesOnlyRealBlocks(t *testing.T) {
	assert.Equal(t, "((1-(1-C14)*(1-C15))*C16)", StripVirtualCodes("(IN*(1-(1-C14)*(1-C15))*C16)*OUT"))
	assert.Equal(t, "(C1*C2)", StripVirtualCodes("(INH-00000005*C1*C2*OUTH-00000005)"))
	assert.Equal(t, "(HSA1*HSB1)", StripSystemVirtualCodes("(INH-00000010*HSA1*HSB1*OUTH-00000010)"))
	assert.Equal(t, "(IN00000010*HSA1)", StripSystemVirtualCodes("(IN00000010*HSA1)"))
	assert.Equal(t, "(R1*R2)", StripVirtualCodes("(START*R1*R2*END)"))
}

func TestVirtualCodeDetection(t *testing.T) {
	assert.True(t, IsVirtualCode("IN"))
	assert.True(t, IsVirtualCode("outH-00000001"))
	assert.True(t, IsVirtualCode("START"))
	assert.False(t, IsVirtualCode("C14"))
	assert.False(t, IsVirtualCode(""))
	assert.True(t, IsVirtualNodeId("node-start"))
	assert.True(t, IsVirtualNumberedCode("IN00000076"))
	assert.False(t, IsVirtualNumberedCode("INH-00000076"))
	assert.False(t, IsVirtualNumberedCode("OUT"))
}
