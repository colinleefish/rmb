package model

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	SessionKey string    `gorm:"type:text;not null;uniqueIndex" json:"session_key"`
	ScopeKey   *string   `gorm:"type:text;index" json:"scope_key,omitempty"`
	Title      *string   `gorm:"type:text" json:"title,omitempty"`
	Status     string    `gorm:"type:text;not null;default:active" json:"status"`
	Abstract   *string   `gorm:"type:text" json:"abstract,omitempty"`
	CreatedAt  time.Time `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt  time.Time `gorm:"type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (Session) TableName() string {
	return "sessions"
}
