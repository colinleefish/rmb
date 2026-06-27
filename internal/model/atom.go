package model

import (
	"time"

	"github.com/colinleefish/rmb/internal/db/pgarray"
	"github.com/google/uuid"
)

type Atom struct {
	URI           string            `gorm:"column:uri;type:text;primaryKey" json:"uri"`
	SessionID     uuid.UUID         `gorm:"column:session_id;type:uuid;not null;index" json:"session_id"`
	Category      string            `gorm:"column:category;type:text;not null;index" json:"category"`
	Priority      int               `gorm:"column:priority;not null;default:50" json:"priority"`
	SceneName     *string           `gorm:"column:scene_name;type:text" json:"scene_name,omitempty"`
	Slug          *string           `gorm:"column:slug;type:text" json:"slug,omitempty"`
	Content       string            `gorm:"column:content;type:text;not null" json:"content"`
	SourceTurnIDs pgarray.UUIDArray `gorm:"column:source_turn_ids;type:uuid[];not null" json:"source_turn_ids"`
	CreatedAt     time.Time         `gorm:"column:created_at;type:timestamptz;not null" json:"created_at"`
	UpdatedAt     time.Time         `gorm:"column:updated_at;type:timestamptz;not null" json:"updated_at"`
}

func (Atom) TableName() string { return "atoms" }
