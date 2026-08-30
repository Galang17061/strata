package rbd

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"

	"github.com/Galang17061/strata-api/internal/domain"
	"github.com/Galang17061/strata-api/internal/reliability"
)

var (
	randomLock sync.Mutex
	random     = rand.New(rand.NewSource(rand.Int63()))
)

const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randomLetters() string {
	randomLock.Lock()
	defer randomLock.Unlock()
	length := random.Intn(3) + 1
	out := make([]byte, length)
	for index := range out {
		out[index] = letters[random.Intn(len(letters))]
	}
	return string(out)
}

func nextCodeWithPrefix(prefix string, existing []string) string {
	suffix := randomLetters()
	full := prefix + suffix
	maxNumber := 0
	found := false
	for _, code := range existing {
		if len(code) > len(full) && strings.HasPrefix(code, full) {
			found = true
			if number, err := strconv.Atoi(code[len(full):]); err == nil && number > maxNumber {
				maxNumber = number
			}
		}
	}
	if !found {
		return full + "1"
	}
	return full + strconv.Itoa(maxNumber+1)
}

func nextCodeAvoiding(prefix string, taken func(code string) bool) string {
	full := prefix + randomLetters()
	counter := 1
	for taken(full + strconv.Itoa(counter)) {
		counter++
	}
	return full + strconv.Itoa(counter)
}

func nextSequenceId(ids []string, prefix string) string {
	maxNumber := 0
	for _, id := range ids {
		number := 0
		if strings.HasPrefix(id, prefix) {
			if parsed, err := strconv.Atoi(id[len(prefix):]); err == nil {
				number = parsed
			}
		} else if parsed, err := strconv.Atoi(id); err == nil {
			number = parsed
		}
		if number > maxNumber {
			maxNumber = number
		}
	}
	return fmt.Sprintf("%08d", maxNumber+1)
}

func counterAfter(lastId *string, prefix string) int {
	if lastId == nil {
		return 1
	}
	number, err := strconv.Atoi(strings.Replace(*lastId, prefix, "", 1))
	if err != nil {
		return 1
	}
	return number + 1
}

func formatId(prefix string, counter int) string {
	return fmt.Sprintf("%s%08d", prefix, counter)
}

func formatPosition(value int) string {
	return fmt.Sprintf("%d.00", value)
}

func positionNumber(value int) *domain.Number {
	return domain.NumberPtr(domain.NumberFromInt(int64(value)))
}

func exponentialAtThousandHours(failureRate *domain.Number) *domain.Number {
	if failureRate == nil {
		return nil
	}
	return domain.NumberPtr(domain.NewNumber(reliability.ExponentialFromDecimals(failureRate.Decimal, domain.NumberFromInt(1000).Decimal)))
}

func seriesOrParallelFormula(codes []string, connectionType *string) *string {
	if len(codes) == 0 {
		return nil
	}
	if connectionType == nil || strings.EqualFold(*connectionType, "series") {
		return domain.StringPtr(strings.Join(codes, "*"))
	}
	if strings.EqualFold(*connectionType, "parallel") {
		if len(codes) == 1 {
			return domain.StringPtr("1-(1-" + codes[0] + ")")
		}
		terms := make([]string, 0, len(codes))
		for _, code := range codes {
			terms = append(terms, "(1-"+code+")")
		}
		return domain.StringPtr("1-(" + strings.Join(terms, "*") + ")")
	}
	return domain.StringPtr(strings.Join(codes, "*"))
}

func componentFormula(components []domain.SystemComponentProperties, connectionType *string) *string {
	expanded := []string{}
	for _, component := range components {
		code := domain.Deref(component.FormulaCode)
		if code == "" || reliability.IsVirtualNumberedCode(code) {
			continue
		}
		repeat := 1
		if component.ActiveComponent != nil && *component.ActiveComponent > 1 {
			repeat = *component.ActiveComponent
		}
		for i := 0; i < repeat; i++ {
			expanded = append(expanded, code)
		}
	}
	return seriesOrParallelFormula(expanded, connectionType)
}

func hierarchyFormula(children []domain.Hierarchy, connectionType *string) *string {
	codes := []string{}
	for _, child := range children {
		code := domain.Deref(child.FormulaCode)
		if code == "" || reliability.IsVirtualNumberedCode(code) {
			continue
		}
		codes = append(codes, code)
	}
	return seriesOrParallelFormula(codes, connectionType)
}
