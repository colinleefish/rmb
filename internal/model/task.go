package model

import (
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID        uuid.UUID  `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	Kind      string     `gorm:"column:kind;type:text;not null" json:"kind"`
	Status    string     `gorm:"column:status;type:text;not null;default:pending" json:"status"`
	Progress  int        `gorm:"column:progress;not null;default:0" json:"progress"`
	ResultURI *string    `gorm:"column:result_uri;type:text" json:"result_uri,omitempty"`
	Error     *string    `gorm:"column:error;type:text" json:"error,omitempty"`
	SessionID *uuid.UUID `gorm:"column:session_id;type:uuid;index" json:"session_id,omitempty"`
	CreatedAt time.Time  `gorm:"column:created_at;type:timestamptz;not null" json:"created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at;type:timestamptz;not null" json:"updated_at"`
}

func (Task) TableName() string { return "tasks" }
