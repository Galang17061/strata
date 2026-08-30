package domain

import "encoding/json"

type HierarchyCalculation struct {
	HierarchyId           string                 `json:"hierarchyId"`
	HierarchyName         *string                `json:"hierarchyName"`
	Level                 int                    `json:"level"`
	FormulaCode           *string                `json:"formulaCode"`
	Formula               *string                `json:"formula"`
	CalculatedReliability *Number                `json:"calculatedReliability"`
	ReliabilityLookup     *OrderedMap            `json:"reliabilityLookup"`
	Children              []HierarchyCalculation `json:"children"`
}

type SystemReliabilityTotal struct {
	RbdSystemId      string      `json:"rbdSystemId"`
	SystemName       *string     `json:"systemName"`
	ReliabilityTotal *Number     `json:"reliabilityTotal"`
	Formula          *string     `json:"formula"`
	Level            any         `json:"level"`
	HierarchyLookup  *OrderedMap `json:"hierarchyLookup"`
	ComponentLookup  *OrderedMap `json:"componentLookup"`
}

type HistoryView struct {
	HistoryId             string          `json:"historyId"`
	RbdSystemId           string          `json:"rbdSystemId"`
	HierarchyId           string          `json:"hierarchyId"`
	HierarchyName         *string         `json:"hierarchyName"`
	HierarchyLevel        *int            `json:"hierarchyLevel"`
	FormulaCode           *string         `json:"formulaCode"`
	Formula               *string         `json:"formula"`
	CalculatedReliability *Number         `json:"calculatedReliability"`
	ReliabilityLookup     json.RawMessage `json:"reliabilityLookup"`
	ComponentDetails      json.RawMessage `json:"componentDetails"`
	RunningHours          *Number         `json:"runningHours"`
	CalculationTimestamp  DateTime        `json:"calculationTimestamp"`
	CalculatedBy          *string         `json:"calculatedBy"`
}

type ComponentHistoryDetail struct {
	FormulaCode         *string `json:"formulaCode"`
	ComponentName       string  `json:"componentName"`
	Vendor              *string `json:"vendor"`
	Type                string  `json:"type"`
	ConnectionType      *string `json:"connectionType"`
	ActiveComponent     *int    `json:"activeComponent"`
	TotalComponent      *int    `json:"totalComponent"`
	BaseReliability     *Number `json:"baseReliability"`
	AdjustedReliability *Number `json:"adjustedReliability"`
	FailureRate         *Number `json:"failureRate"`
}

type HierarchyHistoryDetail struct {
	FormulaCode   *string `json:"formulaCode"`
	ComponentName *string `json:"componentName"`
	Vendor        *string `json:"vendor"`
	Type          string  `json:"type"`
}

type PlotPoint struct {
	DrawingName *string     `json:"drawingName"`
	Time        int         `json:"time"`
	Components  *OrderedMap `json:"components"`
}

type PlotComponentValue struct {
	CompName *string `json:"comp_name"`
	RComp    *Number `json:"r_comp"`
}
