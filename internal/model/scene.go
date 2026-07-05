package model

import (
	"time"

	"github.com/colinleefish/rmb/internal/db/pgarray"
	"github.com/colinleefish/rmb/internal/uri"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Scene struct {
	ID          uuid.UUID         `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	SessionID   uuid.UUID         `gorm:"column:session_id;type:uuid;not null;index" json:"session_id"`
	DisplayName *string           `gorm:"column:display_name;type:text" json:"display_name,omitempty"`
	Abstract    *string           `gorm:"column:abstract;type:text" json:"abstract,omitempty"`
	Body        *string           `gorm:"column:body;type:text" json:"body,omitempty"`
	SourceAtoms pgarray.UUIDArray `gorm:"column:source_atoms;type:uuid[];not null" json:"source_atoms"`
	CreatedAt   time.Time         `gorm:"column:created_at;type:timestamptz;not null" json:"created_at"`
	UpdatedAt   time.Time         `gorm:"column:updated_at;type:timestamptz;not null" json:"updated_at"`
	URI         string            `gorm:"-" json:"uri,omitempty"`
}

func (Scene) TableName() string { return "scenes" }

func (s *Scene) AfterFind(*gorm.DB) error {
	if s.ID != uuid.Nil {
		s.URI = uri.BuildScene(s.ID.String())
	}
	return nil
}
