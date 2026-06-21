package model

import "time"

type TaskGroup struct {
	ID        int       `gorm:"primaryKey;autoIncrement"`
	Name      string    `gorm:"type:varchar(255);not null"`
	Status    string    `gorm:"type:varchar(32);not null;default:DRAFT"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (TaskGroup) TableName() string {
	return "task_group"
}
