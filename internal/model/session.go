package model

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	SessionKey   string    `gorm:"type:text;not null;uniqueIndex"`
	ScopeKey     *string   `gorm:"type:text;index"`
	Title        *string   `gorm:"type:text"`
	Status       string    `gorm:"type:text;not null;default:active"`
	OverviewText *string   `gorm:"type:text"`
	CreatedAt    time.Time `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt    time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (Session) TableName() string {
	return "sessions"
}
