package model

import (
	"time"

	"github.com/colinleefish/rmb/internal/db/pgarray"
	"github.com/google/uuid"
)

type Scene struct {
	URI            string            `gorm:"column:uri;type:text;primaryKey" json:"uri"`
	SessionID      uuid.UUID         `gorm:"column:session_id;type:uuid;not null;index" json:"session_id"`
	DisplayName    *string           `gorm:"column:display_name;type:text" json:"display_name,omitempty"`
	Abstract       *string           `gorm:"column:abstract;type:text" json:"abstract,omitempty"`
	Body           *string           `gorm:"column:body;type:text" json:"body,omitempty"`
	SourceAtomURIs pgarray.TextArray `gorm:"column:source_atom_uris;type:text[];not null" json:"source_atom_uris"`
	CreatedAt      time.Time         `gorm:"column:created_at;type:timestamptz;not null" json:"created_at"`
	UpdatedAt      time.Time         `gorm:"column:updated_at;type:timestamptz;not null" json:"updated_at"`
}

func (Scene) TableName() string { return "scenes" }
