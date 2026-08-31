package domain

type FailureEventCreate struct {
	FailureEventId    *string   `json:"failureEventId"`
	SystemComponentId *string   `json:"systemComponentId"`
	FailureDate       *DateTime `json:"failureDate"`
	RunningHours      int       `json:"runningHours"`
}

type CalculationUpdate struct {
	DistributionType *string `json:"distributionType"`
	FailureRate      *Number `json:"failureRate"`
	RunningHours     *Number `json:"runningHours"`
	ScaleParameter   *Number `json:"scaleParameter"`
	ShapeParameter   *Number `json:"shapeParameter"`
	ReliabilityValue *Number `json:"reliabilityValue"`
	Mtbf             *Number `json:"mtbf"`
	Regresi          *Number `json:"regresi"`
	AllowedFailures  *int    `json:"allowedFailures"`
}

type PoissonParameterView struct {
	PoissonParameterId string    `json:"poissonParameterId"`
	SystemComponentId  string    `json:"systemComponentId"`
	FailureTime        *Number   `json:"failureTime"`
	FailureRate        *Number   `json:"failureRate"`
	AllowedFailures    *int      `json:"allowedFailures"`
	CreatedAt          *DateTime `json:"createdAt"`
	UpdatedAt          *DateTime `json:"updatedAt"`
	TotalReliability   *Number   `json:"totalReliability"`
}

type WeibullParameterView struct {
	WeibullParameterId string    `json:"weibullParameterId"`
	SystemComponentId  string    `json:"systemComponentId"`
	FailureTime        *Number   `json:"failureTime"`
	ScaleParameter     *Number   `json:"scaleParameter"`
	ShapeParameter     *Number   `json:"shapeParameter"`
	CreatedAt          *DateTime `json:"createdAt"`
	UpdatedAt          *DateTime `json:"updatedAt"`
	TotalReliability   *Number   `json:"totalReliability"`
}
