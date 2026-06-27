package model

import "time"

const (
	TaskExecutionProgressStatusInit       = "Init"
	TaskExecutionProgressStatusInProgress = "InProgress"
	TaskExecutionProgressStatusComplete   = "Complete"
	TaskExecutionProgressStatusExpired    = "Expired"
)

type TaskExecutionProgress struct {
	ID        int       `gorm:"primaryKey"`
	TaskID    int       `gorm:"not null"`
	UserID    int       `gorm:"not null"`
	Status    string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (TaskExecutionProgress) TableName() string {
	return "task_execution_progress"
}
