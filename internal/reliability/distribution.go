package reliability

import (
	"errors"
	"math"

	"github.com/shopspring/decimal"
)

func ExponentialFromDecimals(failureRate, runningHours decimal.Decimal) decimal.Decimal {
	exponent := -ToFloat(Mul(failureRate, runningHours))
	return MustFromFloat(math.Exp(exponent))
}

func ExponentialFromFloats(failureRate, runningHours decimal.Decimal) decimal.Decimal {
	lambda := ToFloat(failureRate)
	t := ToFloat(runningHours)
	return MustFromFloat(math.Exp(-(lambda * t)))
}

func WeibullFromRatio(runningHours, scale, shape decimal.Decimal) (decimal.Decimal, error) {
	ratio, err := Div(runningHours, scale)
	if err != nil {
		return decimal.Zero, err
	}
	exponent := -math.Pow(ToFloat(ratio), ToFloat(shape))
	return FromFloat(math.Exp(exponent))
}

func WeibullFromFloats(runningHours, scale, shape decimal.Decimal) (decimal.Decimal, error) {
	t := ToFloat(runningHours)
	eta := ToFloat(scale)
	beta := ToFloat(shape)
	return FromFloat(math.Exp(-math.Pow(t/eta, beta)))
}

func WeibullFloat(t, eta, beta float64) float64 {
	return math.Exp(-math.Pow(t/eta, beta))
}

func SeriesOfIdentical(reliability decimal.Decimal, total int) decimal.Decimal {
	return MustFromFloat(math.Pow(ToFloat(reliability), float64(total)))
}

func ParallelOfIdentical(reliability decimal.Decimal, total int) decimal.Decimal {
	return Sub(decimal.NewFromInt(1), MustFromFloat(math.Pow(1-ToFloat(reliability), float64(total))))
}

func BinomialCoefficient(n, k int) int64 {
	if k > n {
		return 0
	}
	if k == 0 || k == n {
		return 1
	}
	if k > n-k {
		k = n - k
	}
	var result int64 = 1
	for i := 0; i < k; i++ {
		result *= int64(n - i)
		result /= int64(i + 1)
	}
	return result
}

func KOutOfN(reliability decimal.Decimal, k, n int) (decimal.Decimal, error) {
	if k > n {
		return decimal.Zero, errors.New("Active components (" + itoa(k) + ") cannot exceed total components (" + itoa(n) + ")")
	}
	if k <= 0 {
		return decimal.Zero, errors.New("Active components must be greater than 0")
	}
	r := ToFloat(reliability)
	system := 0.0
	for i := k; i <= n; i++ {
		term := float64(BinomialCoefficient(n, i)) * math.Pow(r, float64(i)) * math.Pow(1-r, float64(n-i))
		system += term
	}
	return FromFloat(system)
}
