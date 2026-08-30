package domain

type HierarchyView struct {
	HierarchyId      string          `json:"hierarchyId"`
	RbdSystemId      *string         `json:"rbdSystemId"`
	ParentId         string          `json:"parentId"`
	Level            int             `json:"level"`
	SubSystemName    *string         `json:"subSystemName"`
	Formula          *string         `json:"formula"`
	FormulaCode      *string         `json:"formulaCode"`
	ConnectionType   *string         `json:"connectionType"`
	RealibilityValue *Number         `json:"realibilityValue"`
	RunningHours     *int            `json:"runningHours"`
	PositionX        *Number         `json:"positionX"`
	PositionY        *Number         `json:"positionY"`
	SourceId         *string         `json:"sourceId"`
	TargetId         *string         `json:"targetId"`
	Children         []HierarchyView `json:"children"`
}

func HierarchyViewOf(h Hierarchy) HierarchyView {
	return HierarchyView{
		HierarchyId:      h.HierarchyId,
		RbdSystemId:      h.RbdSystemId,
		ParentId:         h.ParentId,
		Level:            h.Level,
		SubSystemName:    h.SubSystemName,
		Formula:          h.Formula,
		FormulaCode:      h.FormulaCode,
		ConnectionType:   h.ConnectionType,
		RealibilityValue: h.RealibilityValue,
		RunningHours:     h.RunningHours,
		PositionX:        h.PositionX,
		PositionY:        h.PositionY,
		SourceId:         h.SourceId,
		TargetId:         h.TargetId,
	}
}

type HierarchyCreate struct {
	RbdSystemId    *string `json:"rbdSystemId"`
	ParentId       *string `json:"parentId"`
	Level          *int    `json:"level"`
	SubSystemName  *string `json:"subSystemName"`
	Formula        *string `json:"formula"`
	FormulaCode    *string `json:"formulaCode"`
	ConnectionType *string `json:"connectionType"`
	RunningHours   *int    `json:"runningHours"`
	PositionX      *Number `json:"positionX"`
	PositionY      *Number `json:"positionY"`
	SourceId       *string `json:"sourceId"`
	TargetId       *string `json:"targetId"`
}

type HierarchyUpdate struct {
	SubSystemName    *string `json:"subSystemName"`
	Formula          *string `json:"formula"`
	FormulaCode      *string `json:"formulaCode"`
	ConnectionType   *string `json:"connectionType"`
	RealibilityValue *Number `json:"realibilityValue"`
	RunningHours     *int    `json:"runningHours"`
	PositionX        *Number `json:"positionX"`
	PositionY        *Number `json:"positionY"`
	SourceId         *string `json:"sourceId"`
	TargetId         *string `json:"targetId"`
}
