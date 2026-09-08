package reliability

import (
	"errors"
	"strings"
)

const flattenDepthCeiling = 32

func FlattenFormula(formula string, expand map[string]string) (string, error) {
	return flatten(formula, expand, map[string]bool{}, 0)
}

func flatten(formula string, expand map[string]string, walking map[string]bool, depth int) (string, error) {
	if depth > flattenDepthCeiling {
		return "", errors.New("The formula nests deeper than the flattener can follow.")
	}
	var built strings.Builder
	for i := 0; i < len(formula); {
		c := formula[i]
		if c >= '0' && c <= '9' || c == '.' {
			start := i
			for i < len(formula) && isIdentifierChar(formula[i]) || i < len(formula) && formula[i] == '.' {
				i++
			}
			built.WriteString(formula[start:i])
			continue
		}
		if !isIdentifierChar(c) {
			built.WriteByte(c)
			i++
			continue
		}
		start := i
		for i < len(formula) && isIdentifierChar(formula[i]) {
			i++
		}
		code := formula[start:i]
		inner, found := expand[code]
		if !found || strings.TrimSpace(inner) == "" {
			built.WriteString(code)
			continue
		}
		if walking[code] {
			return "", errors.New("The formula refers back to " + code + ", so it never ends.")
		}
		walking[code] = true
		expanded, err := flatten(inner, expand, walking, depth+1)
		delete(walking, code)
		if err != nil {
			return "", err
		}
		built.WriteString("(")
		built.WriteString(expanded)
		built.WriteString(")")
	}
	return built.String(), nil
}
