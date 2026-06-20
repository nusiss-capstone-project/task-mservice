package data

type TaskGroupVO struct {
	ID     int    `json:"id,omitempty"`
	Name   string `json:"name" binding:"required"`
	Status string `json:"status,omitempty"`
}

type PublishStatusVO struct {
	ID     int    `json:"id" binding:"required"`
	Status string `json:"status" binding:"required"`
}
