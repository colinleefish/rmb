package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	SessionTurnStatusNotSummarized = "not_summarized"
	SessionTurnStatusSummarizing   = "summarizing"
	SessionTurnStatusSummarized    = "summarized"
	SessionTurnStatusFailed        = "failed"
)

type SessionTurn struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey"`
	SessionID          uuid.UUID  `gorm:"type:uuid;not null;index:idx_session_turns_status_created,priority:2;index:idx_session_turns_session_created,priority:1"`
	TurnStatus         string     `gorm:"type:text;not null;default:not_summarized;index:idx_session_turns_status_created,priority:1;check:turn_status IN ('not_summarized','summarizing','summarized','failed')"`
	SummarizeStartedAt *time.Time `gorm:"type:timestamptz"`
	MessagesJSONL      string     `gorm:"column:messages_jsonl;type:text;not null"`
	T1ExtractedAt      *time.Time `gorm:"column:t1_extracted_at;type:timestamptz"`
	CreatedAt          time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt          time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

func (SessionTurn) TableName() string {
	return "session_turns"
}
