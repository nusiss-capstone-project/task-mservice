package data

// UserTaskProgressVO is a user's progress view of a task in a group.
type UserTaskProgressVO struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}
