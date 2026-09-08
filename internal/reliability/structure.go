package reliability

import (
	"errors"
	"fmt"
	"math"
	"strconv"
)

type Structure struct {
	root  structureNode
	codes []string
	index map[string]int
}

type structureNode interface {
	value(slots []float64) float64
}

type constantNode float64

func (c constantNode) value([]float64) float64 {
	return float64(c)
}

type variableNode int

func (v variableNode) value(slots []float64) float64 {
	return slots[v]
}

type binaryNode struct {
	operator    byte
	left, right structureNode
}

func (b binaryNode) value(slots []float64) float64 {
	left := b.left.value(slots)
	right := b.right.value(slots)
	switch b.operator {
	case '+':
		return left + right
	case '-':
		return left - right
	case '*':
		return left * right
	}
	if right == 0 {
		return math.NaN()
	}
	return left / right
}

type negateNode struct {
	inner structureNode
}

func (n negateNode) value(slots []float64) float64 {
	return -n.inner.value(slots)
}

type structureParser struct {
	text  string
	pos   int
	codes []string
	index map[string]int
}

func CompileStructure(formula string) (*Structure, error) {
	prepared := PrepareFormula(formula, true)
	p := &structureParser{text: prepared, index: map[string]int{}}
	p.skipSpaces()
	if p.pos >= len(p.text) {
		return nil, errors.New("The formula is empty.")
	}
	root, err := p.expression()
	if err != nil {
		return nil, err
	}
	p.skipSpaces()
	if p.pos < len(p.text) {
		return nil, fmt.Errorf("The formula contains an invalid token at position %d.", p.pos)
	}
	return &Structure{root: root, codes: p.codes, index: p.index}, nil
}

func (s *Structure) Codes() []string {
	copied := make([]string, len(s.codes))
	copy(copied, s.codes)
	return copied
}

func (s *Structure) Slot(code string) (int, bool) {
	slot, found := s.index[code]
	return slot, found
}

func (s *Structure) Slots() int {
	return len(s.codes)
}

func (s *Structure) Value(slots []float64) float64 {
	return s.root.value(slots)
}

func (s *Structure) Standing(slots []float64) bool {
	value := s.root.value(slots)
	return !math.IsNaN(value) && value >= 0.5
}

func (p *structureParser) skipSpaces() {
	for p.pos < len(p.text) && (p.text[p.pos] == ' ' || p.text[p.pos] == '\t') {
		p.pos++
	}
}

func (p *structureParser) expression() (structureNode, error) {
	left, err := p.term()
	if err != nil {
		return nil, err
	}
	for {
		p.skipSpaces()
		if p.pos >= len(p.text) || (p.text[p.pos] != '+' && p.text[p.pos] != '-') {
			return left, nil
		}
		operator := p.text[p.pos]
		p.pos++
		right, err := p.term()
		if err != nil {
			return nil, err
		}
		left = binaryNode{operator: operator, left: left, right: right}
	}
}

func (p *structureParser) term() (structureNode, error) {
	left, err := p.factor()
	if err != nil {
		return nil, err
	}
	for {
		p.skipSpaces()
		if p.pos >= len(p.text) || (p.text[p.pos] != '*' && p.text[p.pos] != '/') {
			return left, nil
		}
		operator := p.text[p.pos]
		p.pos++
		right, err := p.factor()
		if err != nil {
			return nil, err
		}
		left = binaryNode{operator: operator, left: left, right: right}
	}
}

func (p *structureParser) factor() (structureNode, error) {
	p.skipSpaces()
	if p.pos >= len(p.text) {
		return nil, errors.New("The formula is missing an operand.")
	}
	switch c := p.text[p.pos]; {
	case c == '(':
		p.pos++
		inner, err := p.expression()
		if err != nil {
			return nil, err
		}
		p.skipSpaces()
		if p.pos >= len(p.text) || p.text[p.pos] != ')' {
			return nil, errors.New("The formula is missing the closing parenthesis.")
		}
		p.pos++
		return inner, nil
	case c == '-':
		p.pos++
		inner, err := p.factor()
		if err != nil {
			return nil, err
		}
		return negateNode{inner: inner}, nil
	case c == '+':
		p.pos++
		return p.factor()
	case c >= '0' && c <= '9' || c == '.':
		return p.number()
	case isIdentifierChar(c):
		return p.identifier()
	default:
		return nil, fmt.Errorf("The formula contains an invalid token at position %d.", p.pos)
	}
}

func (p *structureParser) identifier() (structureNode, error) {
	start := p.pos
	for p.pos < len(p.text) && isIdentifierChar(p.text[p.pos]) {
		p.pos++
	}
	code := p.text[start:p.pos]
	slot, found := p.index[code]
	if !found {
		slot = len(p.codes)
		p.index[code] = slot
		p.codes = append(p.codes, code)
	}
	return variableNode(slot), nil
}

func (p *structureParser) number() (structureNode, error) {
	start := p.pos
	hasDot := false
	hasExponent := false
	for p.pos < len(p.text) {
		c := p.text[p.pos]
		if c >= '0' && c <= '9' {
			p.pos++
			continue
		}
		if c == '.' && !hasDot && !hasExponent {
			hasDot = true
			p.pos++
			continue
		}
		if (c == 'e' || c == 'E') && !hasExponent {
			hasExponent = true
			p.pos++
			if p.pos < len(p.text) && (p.text[p.pos] == '+' || p.text[p.pos] == '-') {
				p.pos++
			}
			continue
		}
		break
	}
	text := p.text[start:p.pos]
	parsed, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return nil, fmt.Errorf("Cannot interpret token '%s'.", text)
	}
	return constantNode(parsed), nil
}
