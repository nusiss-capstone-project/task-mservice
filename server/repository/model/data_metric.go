package model

import "time"

type DataMetric struct {
	ID         int       `gorm:"primaryKey;autoIncrement"`
	Code       string    `gorm:"type:varchar(128);not null;uniqueIndex"`
	DataSource string    `gorm:"type:varchar(255);not null"`
	Config     string    `gorm:"type:json"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (DataMetric) TableName() string {
	return "data_metric"
}
