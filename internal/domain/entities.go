package domain

type MasterManufacturer struct {
	VendorId         string    `db:"vendor_id" json:"vendorId"`
	ManufacturerName string    `db:"manufacturer_name" json:"manufacturerName"`
	LogoImage        *string   `db:"logo_image" json:"logoImage"`
	ValidUntil       *DateTime `db:"valid_until" json:"validUntil"`
	CreatedAt        DateTime  `db:"created_at" json:"createdAt"`
	UpdatedAt        DateTime  `db:"updated_at" json:"updatedAt"`
	CreatedBy        *string   `db:"created_by" json:"createdBy"`
	UpdatedBy        *string   `db:"updated_by" json:"updatedBy"`
}

type MasterComponent struct {
	ComponentId   string   `db:"component_id" json:"componentId"`
	ComponentName string   `db:"component_name" json:"componentName"`
	VendorId      string   `db:"vendor_id" json:"vendorId"`
	FailureRate   *Number  `db:"failure_rate" json:"failureRate"`
	Cost          *string  `db:"cost" json:"cost"`
	Compatibility *string  `db:"compatibility" json:"compatibility"`
	CreatedAt     DateTime `db:"created_at" json:"createdAt"`
	UpdatedAt     DateTime `db:"updated_at" json:"updatedAt"`
	CreatedBy     *string  `db:"created_by" json:"createdBy"`
	UpdatedBy     *string  `db:"updated_by" json:"updatedBy"`
	SerialNumber  *string  `db:"serial_number" json:"serialNumber"`
}

type MasterProject struct {
	ProjectId      string   `db:"project_id" json:"projectId"`
	ProjectName    string   `db:"project_name" json:"projectName"`
	HierarchyDepth int      `db:"hierarchy_depth" json:"hierarchyDepth"`
	CreatedAt      DateTime `db:"created_at" json:"createdAt"`
	UpdatedAt      DateTime `db:"updated_at" json:"updatedAt"`
	CreatedBy      *string  `db:"created_by" json:"createdBy"`
	UpdatedBy      *string  `db:"updated_by" json:"updatedBy"`
}

type RbdSystemDrawing struct {
	RbdSystemId      string         `db:"rbd_system_id" json:"rbdSystemId"`
	ProjectId        string         `db:"project_id" json:"projectId"`
	DrawingName      *string        `db:"drawing_name" json:"drawingName"`
	SystemName       *string        `db:"system_name" json:"systemName"`
	RunningHours     *string        `db:"running_hours" json:"runningHours"`
	ReliabilityTotal *Number        `db:"reliability_total" json:"reliabilityTotal"`
	Formula          *string        `db:"formula" json:"formula"`
	CreatedAt        DateTime       `db:"created_at" json:"createdAt"`
	UpdatedAt        DateTime       `db:"updated_at" json:"updatedAt"`
	CreatedBy        *string        `db:"created_by" json:"createdBy"`
	UpdatedBy        *string        `db:"updated_by" json:"updatedBy"`
	Project          *MasterProject `db:"-" json:"project"`
}

type SystemComponentProperties struct {
	SystemComponentId  string            `db:"system_component_id" json:"systemComponentId"`
	RbdSystemId        *string           `db:"rbd_system_id" json:"rbdSystemId"`
	ParentId           *string           `db:"parent_id" json:"parentId"`
	ComponentName      string            `db:"component_name" json:"componentName"`
	ComponentTagNumber *string           `db:"component_tag_number" json:"componentTagNumber"`
	Active             *int              `db:"active" json:"active"`
	Vendor             *string           `db:"vendor" json:"vendor"`
	FormulaCode        *string           `db:"formula_code" json:"formulaCode"`
	DistributionType   *string           `db:"distribution_type" json:"distributionType"`
	FailureRate        *Number           `db:"failure_rate" json:"failureRate"`
	RunningHours       *Number           `db:"running_hours" json:"runningHours"`
	ScaleParameter     *Number           `db:"scale_parameter" json:"scaleParameter"`
	ShapeParameter     *Number           `db:"shape_parameter" json:"shapeParameter"`
	ConnectionType     *string           `db:"connection_type" json:"connectionType"`
	ConnectionToId     *string           `db:"connection_to_id" json:"connectionToId"`
	PositionX          *string           `db:"position_x" json:"positionX"`
	PositionY          *string           `db:"position_y" json:"positionY"`
	SourcePosition     *string           `db:"source_position" json:"sourcePosition"`
	TargetPosition     *string           `db:"target_position" json:"targetPosition"`
	IdNode             *string           `db:"id_node" json:"idNode"`
	ReliabilityValue   *Number           `db:"reliability_value" json:"reliabilityValue"`
	ActiveComponent    *int              `db:"active_component" json:"activeComponent"`
	TotalComponent     *int              `db:"total_component" json:"totalComponent"`
	Regresi            *Number           `db:"regresi" json:"regresi"`
	Mtbf               *Number           `db:"mtbf" json:"mtbf"`
	AllowedFailures    *int              `db:"allowed_failures" json:"allowedFailures"`
	CreatedAt          *DateTime         `db:"created_at" json:"createdAt"`
	UpdatedAt          *DateTime         `db:"updated_at" json:"updatedAt"`
	CreatedBy          *string           `db:"created_by" json:"createdBy"`
	UpdatedBy          *string           `db:"updated_by" json:"updatedBy"`
	RbdSystemDrawing   *RbdSystemDrawing `db:"-" json:"rbdSystemDrawing"`
}

type SystemComponentDrawing struct {
	IdEdge   string  `db:"id_edge" json:"idEdge"`
	SourceId *string `db:"source_id" json:"sourceId"`
	TargetId *string `db:"target_id" json:"targetId"`
}

type Hierarchy struct {
	HierarchyId      string  `db:"hierarchy_id" json:"hierarchyId"`
	RbdSystemId      *string `db:"rbd_system_id" json:"rbdSystemId"`
	ParentId         string  `db:"parent_id" json:"parentId"`
	Level            int     `db:"level" json:"level"`
	SubSystemName    *string `db:"sub_system_name" json:"subSystemName"`
	Formula          *string `db:"formula" json:"formula"`
	FormulaCode      *string `db:"formula_code" json:"formulaCode"`
	ConnectionType   *string `db:"connection_type" json:"connectionType"`
	RealibilityValue *Number `db:"realibility_value" json:"realibilityValue"`
	RunningHours     *int    `db:"running_hours" json:"runningHours"`
	PositionX        *Number `db:"position_x" json:"positionX"`
	PositionY        *Number `db:"position_y" json:"positionY"`
	SourceId         *string `db:"source_id" json:"sourceId"`
	TargetId         *string `db:"target_id" json:"targetId"`
}

type FailureEventHistory struct {
	FailureEventId            string                     `db:"failure_event_id" json:"failureEventId"`
	SystemComponentId         string                     `db:"system_component_id" json:"systemComponentId"`
	FailureDate               DateTime                   `db:"failure_date" json:"failureDate"`
	FailureNumber             *int                       `db:"failure_number" json:"failureNumber"`
	RunningHours              int                        `db:"running_hours" json:"runningHours"`
	CreatedAt                 DateTime                   `db:"created_at" json:"createdAt"`
	UpdatedAt                 DateTime                   `db:"updated_at" json:"updatedAt"`
	CreatedBy                 *string                    `db:"created_by" json:"createdBy"`
	UpdatedBy                 *string                    `db:"updated_by" json:"updatedBy"`
	SystemComponentProperties *SystemComponentProperties `db:"-" json:"systemComponentProperties"`
}

type WeibullParameter struct {
	WeibullParameterId        string                     `db:"weibull_parameter_id" json:"weibullParameterId"`
	SystemComponentId         string                     `db:"system_component_id" json:"systemComponentId"`
	FailureEventHours         int                        `db:"failure_event_hours" json:"failureEventHours"`
	N                         int                        `db:"n" json:"n"`
	FreqF                     Number                     `db:"freq_f" json:"freqF"`
	X                         Number                     `db:"x" json:"x"`
	Y                         Number                     `db:"y" json:"y"`
	CreatedAt                 DateTime                   `db:"created_at" json:"createdAt"`
	UpdatedAt                 DateTime                   `db:"updated_at" json:"updatedAt"`
	CreatedBy                 *string                    `db:"created_by" json:"createdBy"`
	UpdatedBy                 *string                    `db:"updated_by" json:"updatedBy"`
	SystemComponentProperties *SystemComponentProperties `db:"-" json:"systemComponentProperties"`
}

type ExponentialParameter struct {
	ExponentialParameterId string   `db:"exponential_parameter_id" json:"exponentialParameterId"`
	SystemComponentId      string   `db:"system_component_id" json:"systemComponentId"`
	FailureEventHours      int      `db:"failure_event_hours" json:"failureEventHours"`
	N                      int      `db:"n" json:"n"`
	Fregf                  Number   `db:"fregf" json:"fregf"`
	FTMedianRank           Number   `db:"f_t_median_rank" json:"ftMedianRank"`
	RT                     Number   `db:"r_t" json:"rt"`
	InRt                   *int     `db:"inRt" json:"inRt"`
	CreatedAt              DateTime `db:"created_at" json:"createdAt"`
	UpdatedAt              DateTime `db:"updated_at" json:"updatedAt"`
	CreatedBy              *string  `db:"created_by" json:"createdBy"`
	UpdatedBy              *string  `db:"updated_by" json:"updatedBy"`
}

type ReliabilityPlotComponent struct {
	ReliabilityPlotId         string                     `db:"reliability_plot_id" json:"reliabilityPlotId"`
	SystemComponentId         string                     `db:"system_component_id" json:"systemComponentId"`
	TimeT                     int                        `db:"time_t" json:"timeT"`
	ReliabilityComp           *Number                    `db:"reliability_comp" json:"reliabilityComp"`
	CreatedAt                 *DateTime                  `db:"created_at" json:"createdAt"`
	UpdatedAt                 *DateTime                  `db:"updated_at" json:"updatedAt"`
	CreatedBy                 *string                    `db:"created_by" json:"createdBy"`
	UpdatedBy                 *string                    `db:"updated_by" json:"updatedBy"`
	SystemComponentProperties *SystemComponentProperties `db:"-" json:"systemComponentProperties"`
}

type ReliabilityHistory struct {
	HistoryId             string   `db:"history_id" json:"historyId"`
	RbdSystemId           string   `db:"rbd_system_id" json:"rbdSystemId"`
	HierarchyId           string   `db:"hierarchy_id" json:"hierarchyId"`
	HierarchyName         *string  `db:"hierarchy_name" json:"hierarchyName"`
	HierarchyLevel        *int     `db:"hierarchy_level" json:"hierarchyLevel"`
	FormulaCode           *string  `db:"formula_code" json:"formulaCode"`
	Formula               *string  `db:"formula" json:"formula"`
	CalculatedReliability *Number  `db:"calculated_reliability" json:"calculatedReliability"`
	ReliabilityLookup     *string  `db:"reliability_lookup" json:"reliabilityLookup"`
	ComponentDetails      *string  `db:"component_details" json:"componentDetails"`
	RunningHours          *Number  `db:"running_hours" json:"runningHours"`
	CalculationTimestamp  DateTime `db:"calculation_timestamp" json:"calculationTimestamp"`
	CalculatedBy          *string  `db:"calculated_by" json:"calculatedBy"`
}

func StringPtr(value string) *string {
	return &value
}

func IntPtr(value int) *int {
	return &value
}

func Deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func DerefInt(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}
