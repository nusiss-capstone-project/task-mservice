package data

type TaskConditionVO struct {
	No          int    `json:"no,omitempty"`
	MetricID    int    `json:"metric_id" binding:"required"`
	OperatorID  int    `json:"operator_id" binding:"required"`
	MetricValue string `json:"metric_value" binding:"required"`
}

type TaskVO struct {
	ID           int               `json:"id,omitempty"`
	Name         string            `json:"name" binding:"required"`
	TaskGroupID  int               `json:"task_group_id,omitempty"`
	Status       string            `json:"status,omitempty"`
	Conditions   []TaskConditionVO `json:"conditions"`
	Expression   string            `json:"expression"`
	StartTime    *DateTime         `json:"start_time,omitempty"`
	EndTime      *DateTime         `json:"end_time,omitempty"`
}
