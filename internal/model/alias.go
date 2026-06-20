package model

import (
	"time"

	"github.com/google/uuid"
)

// Alias is an append-only, human-authored statement that one memory URI
// (AliasURI) is the same entity as another (CanonicalURI). It lives outside the
// memories tier so it survives T3 re-distillation. A row is retired by setting
// SupersededAt on that specific alias. Topology is flat (depth 1): a canonical
// may not itself be an active alias. See docs/aliases.md.
type Alias struct {
	ID           uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	URI          string     `gorm:"column:uri;type:text;not null;uniqueIndex"`
	AliasURI     string     `gorm:"column:alias_uri;type:text;not null"`
	CanonicalURI string     `gorm:"column:canonical_uri;type:text;not null"`
	Note         *string    `gorm:"column:note;type:text"`
	SupersededAt *time.Time `gorm:"column:superseded_at;type:timestamptz"`
	CreatedAt    time.Time  `gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;type:timestamptz;not null"`
}

func (Alias) TableName() string { return "aliases" }
