package domain

type PlotCreate struct {
	ReliabilityPlotId *string `json:"reliabilityPlotId"`
	SystemComponentId *string `json:"systemComponentId"`
	TimeT             int     `json:"timeT"`
	ReliabilityComp   Number  `json:"reliabilityComp"`
}

type PlotUpdate struct {
	SystemComponentId *string `json:"systemComponentId"`
	TimeT             int     `json:"timeT"`
	ReliabilityComp   int     `json:"reliabilityComp"`
}
