package model

import (
	"time"

	"github.com/colinleefish/rmb/internal/db/pgarray"
	"github.com/google/uuid"
)

type Scene struct {
	URI             string    `gorm:"column:uri;type:text;primaryKey"`
	SessionID       uuid.UUID `gorm:"column:session_id;type:uuid;not null;index"`
	DisplayName     *string   `gorm:"column:display_name;type:text"`
	Abstract        *string   `gorm:"column:abstract;type:text"`
	Body            *string   `gorm:"column:body;type:text"`
	SourceAtomURIs  pgarray.TextArray `gorm:"column:source_atom_uris;type:text[];not null"`
	CreatedAt       time.Time `gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt       time.Time `gorm:"column:updated_at;type:timestamptz;not null"`
}

func (Scene) TableName() string { return "scenes" }
