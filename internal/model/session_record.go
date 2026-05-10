package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	SessionRecordStatusNotSummarized = "not_summarized"
	SessionRecordStatusSummarizing   = "summarizing"
	SessionRecordStatusSummarized    = "summarized"
	SessionRecordStatusFailed        = "failed"
)

type SessionRecord struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey"`
	SessionID          uuid.UUID  `gorm:"type:uuid;not null;index:idx_session_records_session_record,priority:1,unique;index:idx_session_records_status_created,priority:2"`
	RecordIndex        int        `gorm:"not null;check:record_index >= 0;index:idx_session_records_session_record,priority:2,unique"`
	RecordStatus       string     `gorm:"type:text;not null;default:not_summarized;index:idx_session_records_status_created,priority:1;check:record_status IN ('not_summarized','summarizing','summarized','failed')"`
	SummarizeStartedAt *time.Time `gorm:"type:timestamptz"`
	MessagesJSONL      string     `gorm:"type:text;not null"`
	CreatedAt          time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt          time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

func (SessionRecord) TableName() string {
	return "session_records"
}
