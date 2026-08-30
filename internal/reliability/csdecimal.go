package reliability

import (
	"errors"
	"math"
	"math/big"
	"strconv"

	"github.com/shopspring/decimal"
)

const maxScale = 28

var (
	maxCoefficient = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 96), big.NewInt(1))
	bigTen         = big.NewInt(10)
)

func Fit(d decimal.Decimal) decimal.Decimal {
	coefficient := new(big.Int).Set(d.Coefficient())
	scale := int(-d.Exponent())
	if scale < 0 {
		coefficient.Mul(coefficient, pow10(-scale))
		scale = 0
	}
	for {
		drop := 0
		if scale > maxScale {
			drop = scale - maxScale
		}
		for new(big.Int).Abs(new(big.Int).Quo(coefficient, pow10(drop))).Cmp(maxCoefficient) > 0 {
			drop++
		}
		if drop == 0 {
			break
		}
		coefficient = roundHalfEven(coefficient, pow10(drop))
		scale -= drop
	}
	return decimal.NewFromBigInt(coefficient, int32(-scale))
}

func roundHalfEven(value, divisor *big.Int) *big.Int {
	quotient, remainder := new(big.Int).QuoRem(value, divisor, new(big.Int))
	negative := remainder.Sign() < 0
	remainder.Abs(remainder)
	doubled := new(big.Int).Lsh(remainder, 1)
	comparison := doubled.Cmp(divisor)
	if comparison > 0 || (comparison == 0 && quotient.Bit(0) == 1) {
		if negative {
			quotient.Sub(quotient, big.NewInt(1))
		} else {
			quotient.Add(quotient, big.NewInt(1))
		}
	}
	return quotient
}

func pow10(exponent int) *big.Int {
	return new(big.Int).Exp(bigTen, big.NewInt(int64(exponent)), nil)
}

func Mul(a, b decimal.Decimal) decimal.Decimal {
	return Fit(a.Mul(b))
}

func Add(a, b decimal.Decimal) decimal.Decimal {
	return Fit(a.Add(b))
}

func Sub(a, b decimal.Decimal) decimal.Decimal {
	return Fit(a.Sub(b))
}

func Div(a, b decimal.Decimal) (decimal.Decimal, error) {
	if b.IsZero() {
		return decimal.Zero, errors.New("Attempted to divide by zero.")
	}
	quotient := a.DivRound(b, 60)
	if quotient.Mul(b).Equal(a) {
		return stripTrailingZeros(quotient), nil
	}
	return Fit(a.DivRound(b, 40)), nil
}

func stripTrailingZeros(d decimal.Decimal) decimal.Decimal {
	coefficient := new(big.Int).Set(d.Coefficient())
	scale := int(-d.Exponent())
	for scale > 0 && new(big.Int).Rem(coefficient, bigTen).Sign() == 0 {
		coefficient.Quo(coefficient, bigTen)
		scale--
	}
	return decimal.NewFromBigInt(coefficient, int32(-scale))
}

func DivideIntegers(a, b int64) (decimal.Decimal, error) {
	if b == 0 {
		return decimal.Zero, errors.New("Attempted to divide by zero.")
	}
	quotient := float64(a) / float64(b)
	return decimal.NewFromString(strconv.FormatFloat(quotient, 'g', -1, 64))
}

func FromFloat(value float64) (decimal.Decimal, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return decimal.Zero, errors.New("Value was either too large or too small for a Decimal.")
	}
	if value == 0 {
		return decimal.Zero, nil
	}
	exponent := int((math.Float64bits(value)>>52)&0x7FF) - 1022
	if exponent < -94 {
		return decimal.Zero, nil
	}
	if exponent > 96 {
		return decimal.Zero, errors.New("Value was either too large or too small for a Decimal.")
	}
	negative := value < 0
	dbl := math.Abs(value)
	power := 14 - ((exponent * 19728) >> 16)
	if power >= 0 {
		if power > maxScale {
			power = maxScale
		}
		dbl *= math.Pow10(power)
	} else {
		if power != -1 || dbl >= 1e15 {
			dbl /= math.Pow10(-power)
		} else {
			power = 0
		}
	}
	if dbl < 1e14 && power < maxScale {
		dbl *= 10
		power++
	}
	mantissa := uint64(int64(dbl))
	fraction := dbl - float64(int64(mantissa))
	if fraction > 0.5 || (fraction == 0.5 && mantissa&1 != 0) {
		mantissa++
	}
	if mantissa == 0 {
		return decimal.Zero, nil
	}
	coefficient := new(big.Int).SetUint64(mantissa)
	scale := power
	if power < 0 {
		coefficient.Mul(coefficient, pow10(-power))
		scale = 0
	} else {
		limit := power
		if limit > 14 {
			limit = 14
		}
		for _, chunk := range []int{8, 4, 2, 1} {
			divisor := pow10(chunk)
			if limit >= chunk && new(big.Int).Rem(coefficient, divisor).Sign() == 0 {
				coefficient.Quo(coefficient, divisor)
				scale -= chunk
				limit -= chunk
			}
		}
	}
	if coefficient.Cmp(maxCoefficient) > 0 {
		return decimal.Zero, errors.New("Value was either too large or too small for a Decimal.")
	}
	if negative {
		coefficient.Neg(coefficient)
	}
	return decimal.NewFromBigInt(coefficient, int32(-scale)), nil
}

func ToFloat(d decimal.Decimal) float64 {
	coefficient := new(big.Float).SetInt(d.Coefficient())
	mantissa, _ := coefficient.Float64()
	scale := int(-d.Exponent())
	if scale <= 0 {
		return mantissa * math.Pow10(-scale)
	}
	return mantissa / math.Pow10(scale)
}

func MustFromFloat(value float64) decimal.Decimal {
	result, err := FromFloat(value)
	if err != nil {
		panic(err)
	}
	return result
}
