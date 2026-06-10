package model

import (
	"time"

	"github.com/colinleefish/mypast/internal/db/pgarray"
	"github.com/google/uuid"
)

// Assertion kinds. v1 exposes correct + forget; the rest are reserved so the
// schema check constraint need not change when they land.
const (
	AssertionKindCorrect = "correct"
	AssertionKindAssert  = "assert"
	AssertionKindForget  = "forget"
	AssertionKindSplit   = "split"
	AssertionKindAlias   = "alias"
)

// Assertion is an append-only, human-authored statement that overlays distilled
// memory. Unlike memories it is not keyed/versioned by target: a row is retired
// only by setting SupersededAt on that specific assertion. See docs/corrections.md.
type Assertion struct {
	ID           uuid.UUID         `gorm:"column:id;type:uuid;primaryKey"`
	URI          string            `gorm:"column:uri;type:text;not null;uniqueIndex"`
	Author       string            `gorm:"column:author;type:text;not null;default:human"`
	Kind         string            `gorm:"column:kind;type:text;not null"`
	TargetURIs   pgarray.TextArray `gorm:"column:target_uris;type:text[];not null"`
	Statement    *string           `gorm:"column:statement;type:text"`
	Payload      *string           `gorm:"column:payload;type:jsonb"`
	SupersededAt *time.Time        `gorm:"column:superseded_at;type:timestamptz"`
	CreatedAt    time.Time         `gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt    time.Time         `gorm:"column:updated_at;type:timestamptz;not null"`
}

func (Assertion) TableName() string { return "assertions" }
