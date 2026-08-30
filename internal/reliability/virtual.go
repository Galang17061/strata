package reliability

import (
	"regexp"
	"strings"
)

var (
	virtualCodePattern   = regexp.MustCompile(`(?i)\b(IN|OUT|START|END)[A-Z]*-?\d*\b`)
	virtualSystemPattern = regexp.MustCompile(`(?i)\b(IN|OUT)[A-Z]*-\d{8}\b`)
	doubleStar           = regexp.MustCompile(`\*\s*\*`)
	edgeStars            = regexp.MustCompile(`^\*+|\*+$`)
	openStar             = regexp.MustCompile(`\(\s*\*`)
	starClose            = regexp.MustCompile(`\*\s*\)`)
	emptyGroup           = regexp.MustCompile(`\(\s*\)`)
)

func hasPrefixFold(text, prefix string) bool {
	return len(text) >= len(prefix) && strings.EqualFold(text[:len(prefix)], prefix)
}

func IsVirtualCode(code string) bool {
	if code == "" {
		return false
	}
	return strings.EqualFold(code, "IN") ||
		strings.EqualFold(code, "OUT") ||
		(hasPrefixFold(code, "IN") && len(code) > 2) ||
		(hasPrefixFold(code, "OUT") && len(code) > 3) ||
		hasPrefixFold(code, "START") ||
		hasPrefixFold(code, "END")
}

func IsVirtualNodeId(nodeId string) bool {
	return IsVirtualCode(nodeId) || hasPrefixFold(nodeId, "node-start") || hasPrefixFold(nodeId, "node-end")
}

func IsVirtualNumberedCode(code string) bool {
	if code == "" {
		return false
	}
	if hasPrefixFold(code, "IN") && len(code) > 2 && isDigit(code[2]) {
		return true
	}
	if hasPrefixFold(code, "OUT") && len(code) > 3 && isDigit(code[3]) {
		return true
	}
	return false
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func cleanStars(formula string) string {
	cleaned := doubleStar.ReplaceAllString(formula, "*")
	cleaned = edgeStars.ReplaceAllString(cleaned, "")
	cleaned = openStar.ReplaceAllString(cleaned, "(")
	cleaned = starClose.ReplaceAllString(cleaned, ")")
	cleaned = emptyGroup.ReplaceAllString(cleaned, "")
	return strings.TrimSpace(cleaned)
}

func StripVirtualCodes(formula string) string {
	if formula == "" {
		return formula
	}
	return cleanStars(virtualCodePattern.ReplaceAllString(formula, ""))
}

func StripSystemVirtualCodes(formula string) string {
	if formula == "" {
		return formula
	}
	return cleanStars(virtualSystemPattern.ReplaceAllString(formula, ""))
}
