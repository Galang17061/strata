package domain

type ThresholdView struct {
	Threshold *float64 `json:"threshold"`
}

type ThresholdUpdate struct {
	Threshold *float64 `json:"threshold"`
}
