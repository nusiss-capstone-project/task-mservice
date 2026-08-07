package model

import "time"

// MetricEventDedup records processed metric events for idempotency (metric_code + biz_id).
type MetricEventDedup struct {
	ID         int64     `gorm:"primaryKey;autoIncrement"`
	MetricCode string    `gorm:"type:varchar(128);not null;uniqueIndex:uk_metric_biz"`
	BizID      string    `gorm:"type:varchar(128);not null;uniqueIndex:uk_metric_biz"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

func (MetricEventDedup) TableName() string {
	return "metric_event_dedup"
}
