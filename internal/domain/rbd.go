package domain

type SystemView struct {
	RbdSystemId *string `db:"rbd_system_id" json:"rbdSystemId"`
	SystemName  *string `db:"system_name" json:"systemName"`
	ProjectId   string  `db:"project_id" json:"projectId"`
	ProjectName string  `db:"project_name" json:"projectName"`
	DrawingName *string `db:"drawing_name" json:"drawingName"`
}

type SystemCreate struct {
	ProjectId  *string     `json:"projectId"`
	SystemName *string     `json:"systemName"`
	Hierarchy  []TreeInput `json:"hierarchy"`
}

type TreeInput struct {
	HierarchyId    *string          `json:"hierarchyId"`
	Name           *string          `json:"name"`
	ConnectionType *string          `json:"connectionType"`
	FormulaCode    *string          `json:"formulaCode"`
	Level          *int             `json:"level"`
	Hierarchy      []TreeInput      `json:"hierarchy"`
	Components     []ComponentInput `json:"components"`
}

type ComponentInput struct {
	SystemComponentId *string `json:"systemComponentId"`
	FormulaCode       *string `json:"formulaCode"`
	ComponentName     *string `json:"componentName"`
	Vendor            *string `json:"vendor"`
	TotalComponent    *int    `json:"totalComponent"`
	ActiveComponent   *int    `json:"activeComponent"`
	ConnectionType    *string `json:"connectionType"`
}

type SystemUpdate struct {
	RbdSystemId *string     `json:"rbdSystemId"`
	ProjectId   *string     `json:"projectId"`
	SystemName  *string     `json:"systemName"`
	Hierarchy   []TreeInput `json:"hierarchy"`
}

type SystemTree struct {
	RbdSystemId string     `json:"rbdSystemId"`
	ProjectId   string     `json:"projectId"`
	ProjectName string     `json:"projectName"`
	SystemName  string     `json:"systemName"`
	Hierarchy   []TreeNode `json:"hierarchy"`
}

type TreeNode struct {
	HierarchyId    string          `json:"hierarchyId"`
	Name           string          `json:"name"`
	ConnectionType *string         `json:"connectionType"`
	FormulaCode    *string         `json:"formulaCode"`
	Level          int             `json:"level"`
	Hierarchy      []TreeNode      `json:"hierarchy"`
	Components     []ComponentNode `json:"components"`
}

type ComponentNode struct {
	SystemComponentId string   `json:"systemComponentId"`
	FormulaCode       *string  `json:"formulaCode"`
	ComponentName     *string  `json:"componentName"`
	VendorName        *string  `json:"vendorName"`
	TotalComponent    *int     `json:"totalComponent"`
	ActiveComponent   *int     `json:"activeComponent"`
	ConnectionType    *string  `json:"connectionType"`
	TargetEdges       []string `json:"targetEdges"`
}

type ComponentInputParameters struct {
	ComponentName        string  `json:"componentName"`
	VendorName           *string `json:"vendorName"`
	FailureRate          *Number `json:"failureRate"`
	RunningHours         *Number `json:"runningHours"`
	FormulaCode          *string `json:"formulaCode"`
	ComponentReliability *Number `json:"componentReliability"`
}

type ComponentInputOutputParameters struct {
	SystemComponentId    string  `json:"systemComponentId"`
	ComponentName        string  `json:"componentName"`
	VendorName           *string `json:"vendorName"`
	FailureRate          *Number `json:"failureRate"`
	RunningHours         *Number `json:"runningHours"`
	FormulaCode          *string `json:"formulaCode"`
	Cost                 *string `json:"cost"`
	ActiveComponent      *int    `json:"activeComponent"`
	TotalComponent       *int    `json:"totalComponent"`
	SerialNumber         *string `json:"serialNumber"`
	DistributionType     *string `json:"distributionType"`
	ShapeParameter       *Number `json:"shapeParameter"`
	ScaleParameter       *Number `json:"scaleParameter"`
	ComponentReliability *Number `json:"componentReliability"`
	Mtbf                 *Number `json:"mtbf"`
}

type MasterComponentWithVendor struct {
	ComponentId      string  `db:"component_id"`
	ComponentName    string  `db:"component_name"`
	VendorId         string  `db:"vendor_id"`
	FailureRate      *Number `db:"failure_rate"`
	Cost             *string `db:"cost"`
	Compatibility    *string `db:"compatibility"`
	SerialNumber     *string `db:"serial_number"`
	ManufacturerName *string `db:"manufacturer_name"`
}
