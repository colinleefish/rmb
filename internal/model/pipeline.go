package model

const (
	PipelineStatusIdle    = "idle"
	PipelineStatusPending = "pending"
	PipelineStatusRunning = "running"
	PipelineStatusFailed  = "failed"

	TaskKindT1       = "t1"
	TaskStatusPending  = "pending"
	TaskStatusRunning  = "running"
	TaskStatusDone     = "done"
	TaskStatusFailed   = "failed"

	AtomCategoryProfile     = "profile"
	AtomCategoryPreferences = "preferences"
	AtomCategoryEntities    = "entities"
	AtomCategoryEvents      = "events"
)

var ValidAtomCategories = map[string]struct{}{
	AtomCategoryProfile:     {},
	AtomCategoryPreferences: {},
	AtomCategoryEntities:    {},
	AtomCategoryEvents:      {},
}
