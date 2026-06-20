package model

import (
	"time"

	"github.com/colinleefish/rmb/internal/db/pgarray"
	"github.com/google/uuid"
)

// Correction is an append-only, human-authored statement that overlays distilled
// memory. Unlike memories it is not keyed/versioned by target: a row is retired
// only by setting SupersededAt on that specific correction. See docs/corrections.md.
type Correction struct {
	ID           uuid.UUID         `gorm:"column:id;type:uuid;primaryKey"`
	URI          string            `gorm:"column:uri;type:text;not null;uniqueIndex"`
	Author       string            `gorm:"column:author;type:text;not null;default:human"`
	TargetURIs   pgarray.TextArray `gorm:"column:target_uris;type:text[];not null"`
	Statement    *string           `gorm:"column:statement;type:text"`
	SupersededAt *time.Time        `gorm:"column:superseded_at;type:timestamptz"`
	CreatedAt    time.Time         `gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt    time.Time         `gorm:"column:updated_at;type:timestamptz;not null"`
}

func (Correction) TableName() string { return "corrections" }
