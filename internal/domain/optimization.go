package domain

type OptimizationLock struct {
	SystemComponentId *string `json:"systemComponentId"`
	ComponentId       *string `json:"componentId"`
}

type OptimizationPreviewRequest struct {
	RbdSystemId          *string            `json:"rbdSystemId"`
	Mode                 *int               `json:"mode"`
	MaxBudget            *float64           `json:"maxBudget"`
	TargetReliability    *float64           `json:"targetReliability"`
	WeightCost           *float64           `json:"weightCost"`
	WeightReliability    *float64           `json:"weightReliability"`
	RunningHours         *float64           `json:"runningHours"`
	PopulationSize       *int               `json:"populationSize"`
	MaxGenerations       *int               `json:"maxGenerations"`
	CrossoverProbability *float64           `json:"crossoverProbability"`
	MutationProbability  *float64           `json:"mutationProbability"`
	Seed                 *int64             `json:"seed"`
	Locks                []OptimizationLock `json:"locks"`
}

type OptimizationScoreRequest struct {
	RbdSystemId  *string            `json:"rbdSystemId"`
	RunningHours *float64           `json:"runningHours"`
	Choices      []OptimizationLock `json:"choices"`
}

type OptimizationCandidateView struct {
	ComponentId string  `json:"componentId"`
	VendorId    string  `json:"vendorId"`
	VendorName  string  `json:"vendorName"`
	FailureRate float64 `json:"failureRate"`
	UnitCost    float64 `json:"unitCost"`
	Reliability float64 `json:"reliability"`
}

type OptimizationSlotView struct {
	SystemComponentId string                      `json:"systemComponentId"`
	ComponentName     string                      `json:"componentName"`
	FormulaCode       string                      `json:"formulaCode"`
	Units             int                         `json:"units"`
	ConnectionType    *string                     `json:"connectionType"`
	CurrentIndex      int                         `json:"currentIndex"`
	ProposedIndex     int                         `json:"proposedIndex"`
	Locked            bool                        `json:"locked"`
	Candidates        []OptimizationCandidateView `json:"candidates"`
}

type OptimizationFixedSlotView struct {
	SystemComponentId string  `json:"systemComponentId"`
	ComponentName     string  `json:"componentName"`
	VendorName        *string `json:"vendorName"`
	Reason            string  `json:"reason"`
	Reliability       float64 `json:"reliability"`
	Cost              float64 `json:"cost"`
}

type OptimizationTotals struct {
	Reliability         float64 `json:"reliability"`
	Cost                float64 `json:"cost"`
	BaselineReliability float64 `json:"baselineReliability"`
	BaselineCost        float64 `json:"baselineCost"`
}

type OptimizationPreviewResult struct {
	RbdSystemId string                      `json:"rbdSystemId"`
	SystemName  *string                     `json:"systemName"`
	Mode        int                         `json:"mode"`
	Slots       []OptimizationSlotView      `json:"slots"`
	FixedSlots  []OptimizationFixedSlotView `json:"fixedSlots"`
	Totals      OptimizationTotals          `json:"totals"`
	Feasible    bool                        `json:"feasible"`
	History     any                         `json:"history"`
	Generations int                         `json:"generations"`
	Seed        int64                       `json:"seed"`
	ExecutionMs int64                       `json:"executionMs"`
}

type OptimizationScoreResult struct {
	Totals   OptimizationTotals `json:"totals"`
	Feasible *bool              `json:"feasible"`
}

type OptimizationApplyRequest struct {
	RbdSystemId          *string            `json:"rbdSystemId"`
	ProjectName          *string            `json:"projectName"`
	SystemName           *string            `json:"systemName"`
	RunningHours         *float64           `json:"runningHours"`
	Choices              []OptimizationLock `json:"choices"`
	Mode                 *int               `json:"mode"`
	MaxBudget            *float64           `json:"maxBudget"`
	TargetReliability    *float64           `json:"targetReliability"`
	WeightCost           *float64           `json:"weightCost"`
	WeightReliability    *float64           `json:"weightReliability"`
	PopulationSize       *int               `json:"populationSize"`
	MaxGenerations       *int               `json:"maxGenerations"`
	CrossoverProbability *float64           `json:"crossoverProbability"`
	MutationProbability  *float64           `json:"mutationProbability"`
	Seed                 *int64             `json:"seed"`
}

type OptimizationApplyResult struct {
	ProjectId   string `json:"projectId"`
	RbdSystemId string `json:"rbdSystemId"`
}

type OptimizationRun struct {
	OptimizationRunId    string   `db:"optimization_run_id" json:"optimizationRunId"`
	RbdSystemId          string   `db:"rbd_system_id" json:"rbdSystemId"`
	Mode                 int      `db:"mode" json:"mode"`
	MaxBudget            *float64 `db:"max_budget" json:"maxBudget"`
	TargetReliability    *float64 `db:"target_reliability" json:"targetReliability"`
	WeightCost           *float64 `db:"weight_cost" json:"weightCost"`
	WeightReliability    *float64 `db:"weight_reliability" json:"weightReliability"`
	RunningHours         *float64 `db:"running_hours" json:"runningHours"`
	PopulationSize       *int     `db:"population_size" json:"populationSize"`
	MaxGenerations       *int     `db:"max_generations" json:"maxGenerations"`
	CrossoverProbability *float64 `db:"crossover_probability" json:"crossoverProbability"`
	MutationProbability  *float64 `db:"mutation_probability" json:"mutationProbability"`
	Seed                 *int64   `db:"seed" json:"seed"`
	Choices              *string  `db:"choices" json:"choices"`
	ResultProjectId      *string  `db:"result_project_id" json:"resultProjectId"`
	ResultRbdSystemId    *string  `db:"result_rbd_system_id" json:"resultRbdSystemId"`
	CreatedAt            DateTime `db:"created_at" json:"createdAt"`
	CreatedBy            *string  `db:"created_by" json:"createdBy"`
}
