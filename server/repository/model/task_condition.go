package model

import "time"

type TaskCondition struct {
	ID              int       `gorm:"primaryKey;autoIncrement"`
	TaskID          int       `gorm:"not null;index"`
	No              int       `gorm:"not null"`
	DataMetricID    int       `gorm:"not null"`
	DataOperatorID  int       `gorm:"not null"`
	ConditionValue  string    `gorm:"type:varchar(255);not null"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

func (TaskCondition) TableName() string {
	return "task_condition"
}
