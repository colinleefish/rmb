package model

import (
	"time"

	"github.com/colinleefish/mem9/internal/db/pgarray"
	"github.com/google/uuid"
)

type Memory struct {
	ID              uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	URI             string     `gorm:"column:uri;type:text;not null;index"`
	Category        string     `gorm:"column:category;type:text;not null;index"`
	Slug            *string    `gorm:"column:slug;type:text"`
	Version         int        `gorm:"column:version;not null;default:1"`
	SupersededAt    *time.Time `gorm:"column:superseded_at;type:timestamptz"`
	Abstract        *string    `gorm:"column:abstract;type:text"`
	Body            *string    `gorm:"column:body;type:text"`
	SourceSceneURIs pgarray.TextArray `gorm:"column:source_scene_uris;type:text[];not null"`
	// SourceCorrectionURIs are the active human corrections baked into Body at
	// distill time. Used by the T3 provenance gate to re-distill when corrections
	// change. See docs/corrections.md.
	SourceCorrectionURIs pgarray.TextArray `gorm:"column:source_correction_uris;type:text[];not null"`
	CreatedAt       time.Time  `gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;type:timestamptz;not null"`
}

func (Memory) TableName() string { return "memories" }
