package model

import (
	"time"

	"github.com/google/uuid"
)

type PipelineState struct {
	SessionID       uuid.UUID  `gorm:"column:session_id;type:uuid;primaryKey"`
	T1Status        string     `gorm:"column:t1_status;type:text;not null;default:idle"`
	T1AdvancedAt         *time.Time `gorm:"column:t1_advanced_at;type:timestamptz"`
	T1TurnsSinceAdvanced int        `gorm:"column:t1_turns_since_advanced;not null;default:0"`
	T2Status             string     `gorm:"column:t2_status;type:text;not null;default:idle"`
	T2AdvancedAt    *time.Time `gorm:"column:t2_advanced_at;type:timestamptz"`
	T3Status        string     `gorm:"column:t3_status;type:text;not null;default:idle"`
	T3AdvancedAt    *time.Time `gorm:"column:t3_advanced_at;type:timestamptz"`
	WarmupThreshold int        `gorm:"column:warmup_threshold;not null;default:2"`
	CreatedAt       time.Time  `gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;type:timestamptz;not null"`
}

func (PipelineState) TableName() string { return "pipeline_state" }
