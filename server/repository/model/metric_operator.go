package model

import "time"

type MetricOperator struct {
	ID        int       `gorm:"primaryKey;autoIncrement"`
	Code      string    `gorm:"type:varchar(64);not null;uniqueIndex"`
	Display   string    `gorm:"type:varchar(128);not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (MetricOperator) TableName() string {
	return "metric_operator"
}
