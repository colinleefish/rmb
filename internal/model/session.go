package model

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	SessionKey string    `gorm:"type:text;not null;uniqueIndex" json:"session_key"`
	Status     string    `gorm:"type:text;not null;default:active" json:"status"`
	Abstract   *string   `gorm:"type:text" json:"abstract,omitempty"`
	CreatedAt  time.Time `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt  time.Time `gorm:"type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (Session) TableName() string {
	return "sessions"
}
