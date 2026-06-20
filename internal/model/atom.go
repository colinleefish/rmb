package model

import (
	"time"

	"github.com/colinleefish/rmb/internal/db/pgarray"
	"github.com/google/uuid"
)

type Atom struct {
	URI            string    `gorm:"column:uri;type:text;primaryKey"`
	SessionID      uuid.UUID `gorm:"column:session_id;type:uuid;not null;index"`
	Category       string    `gorm:"column:category;type:text;not null;index"`
	Priority       int       `gorm:"column:priority;not null;default:50"`
	SceneName      *string   `gorm:"column:scene_name;type:text"`
	Slug           *string   `gorm:"column:slug;type:text"`
	Content        string    `gorm:"column:content;type:text;not null"`
	SourceTurnIDs  pgarray.UUIDArray `gorm:"column:source_turn_ids;type:uuid[];not null"`
	CreatedAt      time.Time `gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt      time.Time `gorm:"column:updated_at;type:timestamptz;not null"`
}

func (Atom) TableName() string { return "atoms" }
