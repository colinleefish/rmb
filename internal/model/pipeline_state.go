package model

import (
	"time"

	"github.com/google/uuid"
)

type PipelineState struct {
	SessionID            uuid.UUID  `gorm:"column:session_id;type:uuid;primaryKey" json:"session_id"`
	T1Status             string     `gorm:"column:t1_status;type:text;not null;default:idle" json:"t1_status"`
	T1AdvancedAt         *time.Time `gorm:"column:t1_advanced_at;type:timestamptz" json:"t1_advanced_at,omitempty"`
	T1TurnsSinceAdvanced int        `gorm:"column:t1_turns_since_advanced;not null;default:0" json:"t1_turns_since_advanced"`
	T2Status             string     `gorm:"column:t2_status;type:text;not null;default:idle" json:"t2_status"`
	T2AdvancedAt         *time.Time `gorm:"column:t2_advanced_at;type:timestamptz" json:"t2_advanced_at,omitempty"`
	T3Status             string     `gorm:"column:t3_status;type:text;not null;default:idle" json:"t3_status"`
	T3AdvancedAt         *time.Time `gorm:"column:t3_advanced_at;type:timestamptz" json:"t3_advanced_at,omitempty"`
	WarmupThreshold      int        `gorm:"column:warmup_threshold;not null;default:2" json:"warmup_threshold"`
	CreatedAt            time.Time  `gorm:"column:created_at;type:timestamptz;not null" json:"created_at"`
	UpdatedAt            time.Time  `gorm:"column:updated_at;type:timestamptz;not null" json:"updated_at"`
}

func (PipelineState) TableName() string { return "pipeline_state" }
