package model

import (
	"time"

	"github.com/colinleefish/rmb/internal/db/pgarray"
	"github.com/google/uuid"
)

type Memory struct {
	ID                   uuid.UUID         `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	URI                  string            `gorm:"column:uri;type:text;not null;index" json:"uri"`
	Category             string            `gorm:"column:category;type:text;not null;index" json:"category"`
	Slug                 *string           `gorm:"column:slug;type:text" json:"slug,omitempty"`
	Version              int               `gorm:"column:version;not null;default:1" json:"version"`
	SupersededAt         *time.Time        `gorm:"column:superseded_at;type:timestamptz" json:"superseded_at,omitempty"`
	Abstract             *string           `gorm:"column:abstract;type:text" json:"abstract,omitempty"`
	Body                 *string           `gorm:"column:body;type:text" json:"body,omitempty"`
	SourceSceneURIs       pgarray.TextArray `gorm:"column:source_scene_uris;type:text[];not null" json:"source_scene_uris"`
	SourceCorrectionURIs pgarray.TextArray `gorm:"column:source_correction_uris;type:text[];not null" json:"source_correction_uris"`
	CreatedAt            time.Time         `gorm:"column:created_at;type:timestamptz;not null" json:"created_at"`
	UpdatedAt            time.Time         `gorm:"column:updated_at;type:timestamptz;not null" json:"updated_at"`
}

func (Memory) TableName() string { return "memories" }
