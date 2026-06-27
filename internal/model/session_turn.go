package model

import (
	"time"

	"github.com/google/uuid"
)

type SessionTurn struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	SessionID     uuid.UUID  `gorm:"type:uuid;not null;index:idx_session_turns_session_created,priority:1" json:"session_id"`
	MessagesJSONL string     `gorm:"column:messages_jsonl;type:text;not null" json:"messages_jsonl"`
	T1ExtractedAt *time.Time `gorm:"column:t1_extracted_at;type:timestamptz" json:"t1_extracted_at,omitempty"`
	CreatedAt     time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (SessionTurn) TableName() string {
	return "session_turns"
}
