package model

import "time"

const (
	TaskConditionExecutionProgressStatusInit       = "Init"
	TaskConditionExecutionProgressStatusInProgress = "InProgress"
	TaskConditionExecutionProgressStatusComplete   = "Complete"
	TaskConditionExecutionProgressStatusExpired    = "Expired"
)

type TaskConditionExecutionProgress struct {
	ID                      int       `gorm:"primaryKey"`
	UserID                  int       `gorm:"not null"`
	TaskExecutionProgressID int       `gorm:"not null"`
	TaskID                  int       `gorm:"not null"`
	TaskConditionID         int       `gorm:"not null"`
	CurrentValue            string     `gorm:"not null"`
	Status                  string     `gorm:"not null"`
	LastEventTime           *time.Time `gorm:"column:last_event_time"`
	CreatedAt               time.Time  `gorm:"autoCreateTime"`
	UpdatedAt               time.Time `gorm:"autoUpdateTime"`
}

func (TaskConditionExecutionProgress) TableName() string {
	return "task_condition_execution_progress"
}
