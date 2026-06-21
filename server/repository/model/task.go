package model

import "time"

type Task struct {
	ID                    int        `gorm:"primaryKey;autoIncrement"`
	TaskGroupID           int        `gorm:"not null;index"`
	Name                  string     `gorm:"type:varchar(255);not null"`
	Status                string     `gorm:"type:varchar(32);not null;default:DRAFT"`
	ConditionExpressions  string     `gorm:"type:text"`
	StartTime             *time.Time `gorm:"type:datetime"`
	EndTime               *time.Time `gorm:"type:datetime"`
	CreatedAt             time.Time  `gorm:"autoCreateTime"`
	UpdatedAt             time.Time  `gorm:"autoUpdateTime"`
}

func (Task) TableName() string {
	return "task"
}
