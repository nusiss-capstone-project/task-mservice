package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nusiss-capstone-project/task-mservice/server/http/data"
	"github.com/nusiss-capstone-project/task-mservice/server/service"
)

// CreateTask creates a task under a task group.
//
// @Summary Create Task
// @Description Create a new task with conditions in a draft task group.
// @Tags Task
// @Accept json
// @Produce json
// @Param task_group_id path int true "Task group ID"
// @Param body body data.TaskVO true "Task"
// @Success 200 {object} data.BaseResponse{data=data.TaskVO}
// @Failure 400 {object} data.BaseResponse
// @Failure 404 {object} data.BaseResponse
// @Failure 500 {object} data.BaseResponse
// @Router /task-ms/v1/task-group/{task_group_id}/tasks [post]
func CreateTask(c *gin.Context) {
	groupID, err := parsePathID(c, "task_group_id")
	if err != nil {
		return
	}
	req := &data.TaskVO{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, data.BaseResponse{ErrMsg: err.Error()})
		return
	}
	ret, err := service.GetTaskService().CreateTask(c.Request.Context(), groupID, req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: ret})
}

// SaveTask updates an existing task.
//
// @Summary Save Task
// @Description Update a draft task and its conditions.
// @Tags Task
// @Accept json
// @Produce json
// @Param task_group_id path int true "Task group ID"
// @Param task_id path int true "Task ID"
// @Param body body data.TaskVO true "Task"
// @Success 200 {object} data.BaseResponse{data=data.TaskVO}
// @Failure 400 {object} data.BaseResponse
// @Failure 404 {object} data.BaseResponse
// @Failure 500 {object} data.BaseResponse
// @Router /task-ms/v1/task-group/{task_group_id}/tasks/{task_id} [put]
func SaveTask(c *gin.Context) {
	groupID, err := parsePathID(c, "task_group_id")
	if err != nil {
		return
	}
	taskID, err := parsePathID(c, "task_id")
	if err != nil {
		return
	}
	req := &data.TaskVO{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, data.BaseResponse{ErrMsg: err.Error()})
		return
	}
	ret, err := service.GetTaskService().SaveTask(c.Request.Context(), groupID, taskID, req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: ret})
}

// ListTasksByGroup lists tasks under a task group.
//
// @Summary List Task by GroupId
// @Description List tasks belonging to a task group.
// @Tags Task
// @Produce json
// @Param task_group_id path int true "Task group ID"
// @Success 200 {object} data.BaseResponse{data=[]data.TaskVO}
// @Failure 400 {object} data.BaseResponse
// @Failure 404 {object} data.BaseResponse
// @Failure 500 {object} data.BaseResponse
// @Router /task-ms/v1/task-group/{task_group_id}/tasks [get]
func ListTasksByGroup(c *gin.Context) {
	groupID, err := parsePathID(c, "task_group_id")
	if err != nil {
		return
	}
	ret, err := service.GetTaskService().ListTasksByGroupID(c.Request.Context(), groupID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: ret})
}

// GetTaskDetail returns task detail with conditions.
//
// @Summary Get Task Detail
// @Description Get task detail including conditions.
// @Tags Task
// @Produce json
// @Param task_group_id path int true "Task group ID"
// @Param task_id path int true "Task ID"
// @Success 200 {object} data.BaseResponse{data=data.TaskVO}
// @Failure 400 {object} data.BaseResponse
// @Failure 404 {object} data.BaseResponse
// @Failure 500 {object} data.BaseResponse
// @Router /task-ms/v1/task-group/{task_group_id}/tasks/{task_id} [get]
func GetTaskDetail(c *gin.Context) {
	groupID, err := parsePathID(c, "task_group_id")
	if err != nil {
		return
	}
	taskID, err := parsePathID(c, "task_id")
	if err != nil {
		return
	}
	ret, err := service.GetTaskService().GetTaskDetail(c.Request.Context(), groupID, taskID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	if ret == nil {
		c.JSON(http.StatusNotFound, data.BaseResponse{ErrMsg: data.ErrTaskNotFound})
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: ret})
}

// PublishTask publishes a task.
//
// @Summary Publish Task
// @Description Publish a draft task.
// @Tags Task
// @Accept json
// @Produce json
// @Param task_id path int true "Task ID"
// @Param body body data.PublishStatusVO false "Publish status"
// @Success 200 {object} data.BaseResponse{data=data.PublishStatusVO}
// @Failure 400 {object} data.BaseResponse
// @Failure 404 {object} data.BaseResponse
// @Failure 500 {object} data.BaseResponse
// @Router /task-ms/v1/tasks/{task_id} [patch]
func PublishTask(c *gin.Context) {
	taskID, err := parsePathID(c, "task_id")
	if err != nil {
		return
	}
	ret, err := service.GetTaskService().PublishTask(c.Request.Context(), taskID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: ret})
}

func parsePathID(c *gin.Context, name string) (int, error) {
	id, err := strconv.Atoi(c.Param(name))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, data.BaseResponse{ErrMsg: data.ErrInvalidInput})
		return 0, err
	}
	return id, nil
}
