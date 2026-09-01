package optimize

import (
	"errors"
	"strconv"
	"strings"
)

type tokenKind int

const (
	tokenNumber tokenKind = iota
	tokenName
	tokenOperator
	tokenOpen
	tokenClose
)

type token struct {
	kind     tokenKind
	number   float64
	name     string
	operator byte
}

type CompiledFormula struct {
	program []token
	Names   []string
}

func isNameRune(r byte) bool {
	return r == '_' || r == '-' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func tokenize(formula string) ([]token, error) {
	tokens := []token{}
	index := 0
	for index < len(formula) {
		char := formula[index]
		switch {
		case char == ' ' || char == '\t':
			index++
		case char == '(':
			tokens = append(tokens, token{kind: tokenOpen})
			index++
		case char == ')':
			tokens = append(tokens, token{kind: tokenClose})
			index++
		case char == '+' || char == '-' || char == '*' || char == '/':
			tokens = append(tokens, token{kind: tokenOperator, operator: char})
			index++
		case char >= '0' && char <= '9' || char == '.':
			end := index
			for end < len(formula) && (formula[end] >= '0' && formula[end] <= '9' || formula[end] == '.') {
				end++
			}
			value, err := strconv.ParseFloat(formula[index:end], 64)
			if err != nil {
				return nil, errors.New("Invalid number in formula: " + formula[index:end])
			}
			tokens = append(tokens, token{kind: tokenNumber, number: value})
			index = end
		case isNameRune(char):
			end := index
			for end < len(formula) && isNameRune(formula[end]) {
				end++
			}
			tokens = append(tokens, token{kind: tokenName, name: formula[index:end]})
			index = end
		default:
			return nil, errors.New("Invalid character in formula: " + string(char))
		}
	}
	return tokens, nil
}

func precedence(operator byte) int {
	if operator == '*' || operator == '/' {
		return 2
	}
	return 1
}

func Compile(formula string) (*CompiledFormula, error) {
	trimmed := strings.TrimSpace(formula)
	if trimmed == "" {
		return nil, errors.New("Formula is empty")
	}
	tokens, err := tokenize(trimmed)
	if err != nil {
		return nil, err
	}
	program := []token{}
	stack := []token{}
	names := []string{}
	seen := map[string]bool{}
	expectValue := true
	for _, current := range tokens {
		switch current.kind {
		case tokenNumber:
			program = append(program, current)
			expectValue = false
		case tokenName:
			program = append(program, current)
			if !seen[current.name] {
				seen[current.name] = true
				names = append(names, current.name)
			}
			expectValue = false
		case tokenOperator:
			if expectValue && current.operator == '-' {
				program = append(program, token{kind: tokenNumber, number: 0})
			}
			for len(stack) > 0 && stack[len(stack)-1].kind == tokenOperator && precedence(stack[len(stack)-1].operator) >= precedence(current.operator) {
				program = append(program, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			stack = append(stack, current)
			expectValue = true
		case tokenOpen:
			stack = append(stack, current)
			expectValue = true
		case tokenClose:
			for len(stack) > 0 && stack[len(stack)-1].kind != tokenOpen {
				program = append(program, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			if len(stack) == 0 {
				return nil, errors.New("Unbalanced parentheses in formula")
			}
			stack = stack[:len(stack)-1]
			expectValue = false
		}
	}
	for len(stack) > 0 {
		top := stack[len(stack)-1]
		if top.kind == tokenOpen {
			return nil, errors.New("Unbalanced parentheses in formula")
		}
		program = append(program, top)
		stack = stack[:len(stack)-1]
	}
	if len(program) == 0 {
		return nil, errors.New("Formula is empty")
	}
	return &CompiledFormula{program: program, Names: names}, nil
}

func (f *CompiledFormula) Eval(values map[string]float64) (float64, error) {
	stack := make([]float64, 0, len(f.program))
	for _, current := range f.program {
		switch current.kind {
		case tokenNumber:
			stack = append(stack, current.number)
		case tokenName:
			value, known := values[current.name]
			if !known {
				return 0, errors.New("No value for " + current.name)
			}
			stack = append(stack, value)
		case tokenOperator:
			if len(stack) < 2 {
				return 0, errors.New("Malformed formula")
			}
			right := stack[len(stack)-1]
			left := stack[len(stack)-2]
			stack = stack[:len(stack)-2]
			switch current.operator {
			case '+':
				stack = append(stack, left+right)
			case '-':
				stack = append(stack, left-right)
			case '*':
				stack = append(stack, left*right)
			case '/':
				stack = append(stack, left/right)
			}
		}
	}
	if len(stack) != 1 {
		return 0, errors.New("Malformed formula")
	}
	return stack[0], nil
}
