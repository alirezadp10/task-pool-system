package model

import (
	"task-pool-system.com/task-pool-system/pkg/constants"
	"time"
)

type Task struct {
	ID          string               `gorm:"primaryKey;size:36" json:"id"`
	Title       string               `gorm:"not null" json:"title"`
	Description string               `gorm:"not null" json:"description"`
	Status      constants.TaskStatus `gorm:"type:varchar(20);not null" json:"status"`
	Version     uint                 `gorm:"not null;default:1" json:"version"`
	CreatedAt   time.Time            `json:"created_at"`
	StartedAt   *time.Time           `json:"started_at,omitempty"`
	CompletedAt *time.Time           `json:"completed_at,omitempty"`
}
