package domain

type ScpListItem struct {
	ComponentName  *string `db:"component_name" json:"componentName"`
	TotalComponent *int    `db:"total_component" json:"totalComponent"`
	TotalActive    *int    `db:"total_active" json:"totalActive"`
}

type ComponentDetail struct {
	SystemComponentId  string    `json:"systemComponentId"`
	RbdSystemId        *string   `json:"rbdSystemId"`
	ParentId           *string   `json:"parentId"`
	ComponentName      string    `json:"componentName"`
	ComponentTagNumber *string   `json:"componentTagNumber"`
	Active             *int      `json:"active"`
	Vendor             *string   `json:"vendor"`
	FormulaCode        *string   `json:"formulaCode"`
	DistributionType   *string   `json:"distributionType"`
	FailureRate        *Number   `json:"failureRate"`
	RunningHours       *Number   `json:"runningHours"`
	ScaleParameter     *Number   `json:"scaleParameter"`
	ShapeParameter     *Number   `json:"shapeParameter"`
	ConnectionType     *string   `json:"connectionType"`
	ConnectionToId     *string   `json:"connectionToId"`
	PositionX          *string   `json:"positionX"`
	PositionY          *string   `json:"positionY"`
	SourcePosition     *string   `json:"sourcePosition"`
	TargetPosition     *string   `json:"targetPosition"`
	IdNode             *string   `json:"idNode"`
	ReliabilityValue   *Number   `json:"reliabilityValue"`
	ActiveComponent    *int      `json:"activeComponent"`
	TotalComponent     *int      `json:"totalComponent"`
	Regresi            *Number   `json:"regresi"`
	Mtbf               *Number   `json:"mtbf"`
	AllowedFailures    *int      `json:"allowedFailures"`
	CreatedAt          *DateTime `json:"createdAt"`
	UpdatedAt          *DateTime `json:"updatedAt"`
	CreatedBy          *string   `json:"createdBy"`
	UpdatedBy          *string   `json:"updatedBy"`
}

func ComponentDetailOf(c SystemComponentProperties) ComponentDetail {
	return ComponentDetail{
		SystemComponentId:  c.SystemComponentId,
		RbdSystemId:        c.RbdSystemId,
		ParentId:           c.ParentId,
		ComponentName:      c.ComponentName,
		ComponentTagNumber: c.ComponentTagNumber,
		Active:             c.Active,
		Vendor:             c.Vendor,
		FormulaCode:        c.FormulaCode,
		DistributionType:   c.DistributionType,
		FailureRate:        c.FailureRate,
		RunningHours:       c.RunningHours,
		ScaleParameter:     c.ScaleParameter,
		ShapeParameter:     c.ShapeParameter,
		ConnectionType:     c.ConnectionType,
		ConnectionToId:     c.ConnectionToId,
		PositionX:          c.PositionX,
		PositionY:          c.PositionY,
		SourcePosition:     c.SourcePosition,
		TargetPosition:     c.TargetPosition,
		IdNode:             c.IdNode,
		ReliabilityValue:   c.ReliabilityValue,
		ActiveComponent:    c.ActiveComponent,
		TotalComponent:     c.TotalComponent,
		Regresi:            c.Regresi,
		Mtbf:               c.Mtbf,
		AllowedFailures:    c.AllowedFailures,
		CreatedAt:          c.CreatedAt,
		UpdatedAt:          c.UpdatedAt,
		CreatedBy:          c.CreatedBy,
		UpdatedBy:          c.UpdatedBy,
	}
}

type ComponentSimpleCreate struct {
	ParentId           *string `json:"parentId"`
	ComponentName      *string `json:"componentName"`
	ComponentTagNumber *string `json:"componentTagNumber"`
	Vendor             *string `json:"vendor"`
}

type ComponentDetailUpdate struct {
	ParentId           *string `json:"parentId"`
	ComponentName      *string `json:"componentName"`
	ComponentTagNumber *string `json:"componentTagNumber"`
	Vendor             *string `json:"vendor"`
	FormulaCode        *string `json:"formulaCode"`
	DistributionType   *string `json:"distributionType"`
	FailureRate        *Number `json:"failureRate"`
	RunningHours       *Number `json:"runningHours"`
	ScaleParameter     *Number `json:"scaleParameter"`
	ShapeParameter     *Number `json:"shapeParameter"`
	ConnectionType     *string `json:"connectionType"`
	ConnectionToId     *string `json:"connectionToId"`
	PositionX          *string `json:"positionX"`
	PositionY          *string `json:"positionY"`
	SourcePosition     *string `json:"sourcePosition"`
	TargetPosition     *string `json:"targetPosition"`
	IdNode             *string `json:"idNode"`
	ReliabilityValue   *Number `json:"reliabilityValue"`
	Active             *int    `json:"active"`
	ActiveComponent    *int    `json:"activeComponent"`
	TotalComponent     *int    `json:"totalComponent"`
	Regresi            *Number `json:"regresi"`
	Mtbf               *Number `json:"mtbf"`
	AllowedFailures    *int    `json:"allowedFailures"`
}

type DrawingNodeInput struct {
	ConnectionType *string `json:"connectionType"`
	PositionX      *string `json:"positionX"`
	PositionY      *string `json:"positionY"`
	IdNode         *string `json:"idNode"`
}

type EdgeInput struct {
	IdEdge   *string `json:"idEdge"`
	SourceId *string `json:"sourceId"`
	TargetId *string `json:"targetId"`
}

type ComponentNodeView struct {
	SystemComponentId string  `json:"systemComponentId"`
	ConnectionType    *string `json:"connectionType"`
	PositionX         *string `json:"positionX"`
	PositionY         *string `json:"positionY"`
	IdNode            *string `json:"idNode"`
	ComponentName     *string `json:"componentName"`
	VendorName        *string `json:"vendorName"`
	ActiveComponent   *int    `json:"activeComponent"`
}

type HierarchyNodeView struct {
	HierarchyId     string  `json:"hierarchyId"`
	ConnectionType  *string `json:"connectionType"`
	PositionX       *string `json:"positionX"`
	PositionY       *string `json:"positionY"`
	IdNode          *string `json:"idNode"`
	ComponentName   *string `json:"componentName"`
	VendorName      *string `json:"vendorName"`
	ActiveComponent *int    `json:"activeComponent"`
}

type EdgeView struct {
	IdEdge   *string `json:"idEdge"`
	SourceId *string `json:"sourceId"`
	TargetId *string `json:"targetId"`
}

type MonitoredComponent struct {
	ComponentName    *string `json:"componentName"`
	Vendor           *string `json:"vendor"`
	DistributionType *string `json:"distributionType"`
	FailureRate      *Number `json:"failureRate"`
	RunningHours     *Number `json:"runningHours"`
	ScaleParameter   *Number `json:"scaleParameter"`
	ShapeParameter   *Number `json:"shapeParameter"`
}

type CalculationRow struct {
	ComponentName    *string  `json:"componentName"`
	ConnectionType   *string  `json:"connectionType"`
	ReliabilityValue *float64 `json:"reliabilityValue"`
}

type CalculationSummary struct {
	DataCalculation  []CalculationRow `json:"dataCalculation"`
	ReliabilityTotal *float64         `json:"reliabilityTotal"`
}

type ParameterSuggestion struct {
	Source      string   `json:"source"`
	Events      int      `json:"events"`
	Components  int      `json:"components"`
	TotalHours  float64  `json:"totalHours"`
	FailureRate *float64 `json:"failureRate"`
	Mtbf        *float64 `json:"mtbf"`
	Beta        *float64 `json:"beta"`
	Eta         *float64 `json:"eta"`
}
