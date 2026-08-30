package reliability

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

type operand struct {
	integer   int64
	number    decimal.Decimal
	isDecimal bool
}

func (o operand) toDecimal() decimal.Decimal {
	if o.isDecimal {
		return o.number
	}
	return decimal.NewFromInt(o.integer)
}

type parser struct {
	text string
	pos  int
}

func Evaluate(formula string) (decimal.Decimal, error) {
	p := &parser{text: formula}
	p.skipSpaces()
	if p.pos >= len(p.text) {
		return decimal.Zero, errors.New("Object cannot be cast from DBNull to other types.")
	}
	value, err := p.expression()
	if err != nil {
		return decimal.Zero, err
	}
	p.skipSpaces()
	if p.pos < len(p.text) {
		return decimal.Zero, fmt.Errorf("The expression contains invalid token at position %d.", p.pos)
	}
	return value.toDecimal(), nil
}

func PrepareFormula(formula string, balanceParentheses bool) string {
	formula = strings.ReplaceAll(formula, "R_OUT", "1.0")
	formula = strings.ReplaceAll(formula, "START", "1.0")
	formula = strings.ReplaceAll(formula, "END", "1.0")
	formula = strings.TrimSpace(formula)
	for len(formula) > 0 && strings.ContainsRune("+-*/", rune(formula[len(formula)-1])) {
		formula = formula[:len(formula)-1]
	}
	if balanceParentheses {
		open := strings.Count(formula, "(")
		closed := strings.Count(formula, ")")
		if open > closed {
			formula += strings.Repeat(")", open-closed)
		}
	}
	return formula
}

func (p *parser) skipSpaces() {
	for p.pos < len(p.text) && (p.text[p.pos] == ' ' || p.text[p.pos] == '\t') {
		p.pos++
	}
}

func (p *parser) expression() (operand, error) {
	left, err := p.term()
	if err != nil {
		return operand{}, err
	}
	for {
		p.skipSpaces()
		if p.pos >= len(p.text) || (p.text[p.pos] != '+' && p.text[p.pos] != '-') {
			return left, nil
		}
		op := p.text[p.pos]
		p.pos++
		right, err := p.term()
		if err != nil {
			return operand{}, err
		}
		left = apply(op, left, right)
	}
}

func (p *parser) term() (operand, error) {
	left, err := p.factor()
	if err != nil {
		return operand{}, err
	}
	for {
		p.skipSpaces()
		if p.pos >= len(p.text) || (p.text[p.pos] != '*' && p.text[p.pos] != '/') {
			return left, nil
		}
		op := p.text[p.pos]
		p.pos++
		right, err := p.factor()
		if err != nil {
			return operand{}, err
		}
		if op == '/' {
			if right.toDecimal().IsZero() {
				return operand{}, errors.New("Attempted to divide by zero.")
			}
			if !left.isDecimal && !right.isDecimal {
				quotient, err := DivideIntegers(left.integer, right.integer)
				if err != nil {
					return operand{}, err
				}
				left = operand{number: quotient, isDecimal: true}
				continue
			}
			quotient, err := Div(left.toDecimal(), right.toDecimal())
			if err != nil {
				return operand{}, err
			}
			left = operand{number: quotient, isDecimal: true}
			continue
		}
		left = apply(op, left, right)
	}
}

func apply(op byte, left, right operand) operand {
	if !left.isDecimal && !right.isDecimal {
		switch op {
		case '+':
			return operand{integer: left.integer + right.integer}
		case '-':
			return operand{integer: left.integer - right.integer}
		default:
			return operand{integer: left.integer * right.integer}
		}
	}
	a, b := left.toDecimal(), right.toDecimal()
	switch op {
	case '+':
		return operand{number: Add(a, b), isDecimal: true}
	case '-':
		return operand{number: Sub(a, b), isDecimal: true}
	default:
		return operand{number: Mul(a, b), isDecimal: true}
	}
}

func (p *parser) factor() (operand, error) {
	p.skipSpaces()
	if p.pos >= len(p.text) {
		return operand{}, errors.New("The expression has too many closing parentheses or is missing an operand.")
	}
	switch c := p.text[p.pos]; {
	case c == '(':
		p.pos++
		value, err := p.expression()
		if err != nil {
			return operand{}, err
		}
		p.skipSpaces()
		if p.pos >= len(p.text) || p.text[p.pos] != ')' {
			return operand{}, errors.New("The expression is missing the closing parenthesis.")
		}
		p.pos++
		return value, nil
	case c == '-':
		p.pos++
		value, err := p.factor()
		if err != nil {
			return operand{}, err
		}
		if value.isDecimal {
			return operand{number: value.number.Neg(), isDecimal: true}, nil
		}
		return operand{integer: -value.integer}, nil
	case c == '+':
		p.pos++
		return p.factor()
	case c >= '0' && c <= '9' || c == '.':
		return p.number()
	default:
		start := p.pos
		for p.pos < len(p.text) && (isIdentifierChar(p.text[p.pos])) {
			p.pos++
		}
		if start == p.pos {
			p.pos++
		}
		return operand{}, fmt.Errorf("Cannot find column [%s].", p.text[start:p.pos])
	}
}

func isIdentifierChar(c byte) bool {
	return c == '_' || c == '-' && false || c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z'
}

func (p *parser) number() (operand, error) {
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
	if hasExponent {
		value, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return operand{}, fmt.Errorf("Cannot interpret token '%s'.", text)
		}
		parsed, err := FromFloat(value)
		if err != nil {
			return operand{}, err
		}
		return operand{number: parsed, isDecimal: true}, nil
	}
	if hasDot {
		parsed, err := decimal.NewFromString(text)
		if err != nil {
			return operand{}, fmt.Errorf("Cannot interpret token '%s'.", text)
		}
		return operand{number: parsed, isDecimal: true}, nil
	}
	integer, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		parsed, err := decimal.NewFromString(text)
		if err != nil {
			return operand{}, fmt.Errorf("Cannot interpret token '%s'.", text)
		}
		return operand{number: parsed, isDecimal: true}, nil
	}
	return operand{integer: integer}, nil
}
