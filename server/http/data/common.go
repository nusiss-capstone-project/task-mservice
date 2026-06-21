package data

type BaseResponse struct {
	Code   int         `json:"code"`
	ErrMsg string      `json:"err_msg,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}

const (
	ErrServerError = "server error"

	ErrTaskGroupNil                   = "task group is nil"
	ErrTaskGroupNameRequired          = "task group name is required"
	ErrTaskGroupNotFound              = "task group not found"
	ErrPublishedTaskGroupCannotModify = "published task group cannot be modified"
	ErrTaskGroupPublishedCannotModify = "task group is published and cannot be modified"

	ErrTaskNil                     = "task is nil"
	ErrTaskNotFound                = "task not found"
	ErrPublishedTaskCannotModify   = "published task cannot be modified"
	ErrTaskNameRequired            = "task name is required"
	ErrAtLeastOneConditionRequired = "at least one condition is required"

	ErrInvalidInput = "invalid input"
)
