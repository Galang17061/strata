package domain

type ThresholdView struct {
	Threshold *float64 `json:"threshold"`
}

type ThresholdUpdate struct {
	Threshold *float64 `json:"threshold"`
}

type BatchRecalculateRequest struct {
	RunningHours *float64 `json:"runningHours"`
}

type BatchRecalculateResult struct {
	Components       int      `json:"components"`
	Unfitted         []string `json:"unfitted"`
	ReliabilityTotal *float64 `json:"reliabilityTotal"`
}
