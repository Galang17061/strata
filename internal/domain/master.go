package domain

type FormFileInfo struct {
	ContentDisposition string              `json:"contentDisposition"`
	ContentType        string              `json:"contentType"`
	Headers            map[string][]string `json:"headers"`
	Length             int64               `json:"length"`
	Name               string              `json:"name"`
	FileName           string              `json:"fileName"`
}

type ManufacturerForm struct {
	ManufacturerName string        `json:"manufacturerName"`
	LogoImage        *FormFileInfo `json:"logoImage"`
	ValidUntil       *DateTime     `json:"validUntil"`
}

type MasterComponentView struct {
	ComponentId      string   `db:"component_id" json:"componentId"`
	ComponentName    string   `db:"component_name" json:"componentName"`
	VendorId         string   `db:"vendor_id" json:"vendorId"`
	ManufacturerName string   `db:"manufacturer_name" json:"manufacturerName"`
	Cost             *string  `db:"cost" json:"cost"`
	Compatibility    *string  `db:"compatibility" json:"compatibility"`
	FailureRate      *Number  `db:"failure_rate" json:"failureRate"`
	CreatedAt        DateTime `db:"created_at" json:"createdAt"`
	UpdatedAt        DateTime `db:"updated_at" json:"updatedAt"`
	CreatedBy        *string  `db:"created_by" json:"createdBy"`
	UpdatedBy        *string  `db:"updated_by" json:"updatedBy"`
	SerialNumber     *string  `db:"serial_number" json:"serialNumber"`
}

type MasterComponentCreate struct {
	ComponentName string  `json:"componentName"`
	VendorId      string  `json:"vendorId"`
	Cost          *string `json:"cost"`
	Compatibility *string `json:"compatibility"`
	FailureRate   Number  `json:"failureRate"`
	SerialNumber  *string `json:"serialNumber"`
}

type MasterComponentUpdate struct {
	ComponentName string  `json:"componentName"`
	VendorId      string  `json:"vendorId"`
	FailureRate   Number  `json:"failureRate"`
	Cost          *string `json:"cost"`
	Compatibility *string `json:"compatibility"`
	SerialNumber  *string `json:"serialNumber"`
}

type ImportResult struct {
	Success      bool     `json:"success"`
	SuccessCount int      `json:"successCount"`
	UpdatedCount int      `json:"updatedCount"`
	FailedRows   []string `json:"failedRows"`
}

type MasterProjectCreate struct {
	ProjectId      *string `json:"projectId"`
	ProjectName    string  `json:"projectName"`
	HierarchyDepth *int    `json:"hierarchyDepth"`
}

type MasterProjectUpdate struct {
	ProjectName string `json:"projectName"`
}

type MasterProjectRbd struct {
	RbdSystemId      string   `db:"rbd_system_id" json:"rbdSystemId"`
	ProjectId        string   `db:"project_id" json:"projectId"`
	ProjectName      string   `db:"project_name" json:"projectName"`
	DrawingName      *string  `db:"drawing_name" json:"drawingName"`
	SystemName       *string  `db:"system_name" json:"systemName"`
	ReliabilityTotal *Number  `db:"reliability_total" json:"reliabilityTotal"`
	CreatedAt        DateTime `db:"created_at" json:"createdAt"`
	UpdatedAt        DateTime `db:"updated_at" json:"updatedAt"`
}

type ComponentNameOnly struct {
	ComponentName string `json:"componentName"`
}

type HierarchySummary struct {
	Level            int                 `json:"level"`
	SubSystemName    *string             `json:"subSystemName"`
	SystemComponents []ComponentNameOnly `json:"systemComponents"`
}

type RbdSystemSummary struct {
	SystemName  *string            `json:"systemName"`
	Hierarchies []HierarchySummary `json:"hierarchies"`
}

type MasterProjectDetail struct {
	ProjectName string             `json:"projectName"`
	RbdSystems  []RbdSystemSummary `json:"rbdSystems"`
}
