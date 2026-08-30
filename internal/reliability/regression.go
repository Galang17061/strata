package reliability

import (
	"errors"
	"math"
)

func sum(values []float64) float64 {
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total
}

func average(values []float64) float64 {
	return sum(values) / float64(len(values))
}

func SlopeWeibull(x, y []float64) float64 {
	n := float64(len(x))
	sumX := sum(x)
	sumY := sum(y)
	sumXY := 0.0
	sumXSquare := 0.0
	for index := range x {
		sumXY += x[index] * y[index]
		sumXSquare += x[index] * x[index]
	}
	return (n*sumXY - sumX*sumY) / (n*sumXSquare - sumX*sumX)
}

func InterceptWeibull(x, y []float64) float64 {
	n := float64(len(x))
	slope := SlopeWeibull(x, y)
	return (sum(y) - slope*sum(x)) / n
}

func SlopeExponential(yValues, xValues []float64) (float64, error) {
	if len(yValues) != len(xValues) || len(yValues) == 0 {
		return 0, errors.New("Invalid input data for slope calculation.")
	}
	xAvg := average(xValues)
	yAvg := average(yValues)
	numerator := 0.0
	denominator := 0.0
	for index := range xValues {
		numerator += (xValues[index] - xAvg) * (yValues[index] - yAvg)
		denominator += math.Pow(xValues[index]-xAvg, 2)
	}
	return numerator / denominator, nil
}

func RSquared(yValues, xValues []float64) (float64, error) {
	if len(yValues) != len(xValues) || len(yValues) == 0 {
		return 0, errors.New("Invalid input data for R-squared calculation.")
	}
	slope, err := SlopeExponential(yValues, xValues)
	if err != nil {
		return 0, err
	}
	intercept := average(yValues) - slope*average(xValues)
	ssTotal := 0.0
	ssResidual := 0.0
	yAvg := average(yValues)
	for index := range yValues {
		ssTotal += math.Pow(yValues[index]-yAvg, 2)
		ssResidual += math.Pow(yValues[index]-(slope*xValues[index]+intercept), 2)
	}
	return 1 - (ssResidual / ssTotal), nil
}

func Finite(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return value
}

func MedianRank(n, total int) float64 {
	return (float64(n) - 0.3) / (float64(total) + 0.4)
}

func WeibullPlotPoint(runningHours int, frequency float64) (float64, float64) {
	hours := 0.001
	if runningHours > 0 {
		hours = float64(runningHours)
	}
	x := math.Log(hours)
	innerLog := 1 / (1 - frequency)
	y := 0.0
	if innerLog > 0 {
		y = math.Log(math.Log(innerLog))
	}
	return Finite(x), Finite(y)
}
