package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nusiss-capstone-project/task-mservice/server/http/data"
	"github.com/nusiss-capstone-project/task-mservice/server/service"
)

// SaveTaskGroup creates or updates a task group.
//
// @Summary Save TaskGroup
// @Description Create a new task group or update an existing draft task group.
// @Tags TaskGroup
// @Accept json
// @Produce json
// @Param body body data.TaskGroupVO true "Task group"
// @Success 200 {object} data.BaseResponse{data=data.TaskGroupVO}
// @Failure 400 {object} data.BaseResponse
// @Failure 404 {object} data.BaseResponse
// @Failure 500 {object} data.BaseResponse
// @Router /task-ms/v1/task-groups [post]
func SaveTaskGroup(c *gin.Context) {
	req := &data.TaskGroupVO{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, data.BaseResponse{ErrMsg: err.Error()})
		return
	}
	ret, err := service.GetTaskGroupService().SaveTaskGroup(c.Request.Context(), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: ret})
}

// ListTaskGroups returns all task groups.
//
// @Summary List TaskGroup
// @Description List all task groups.
// @Tags TaskGroup
// @Produce json
// @Success 200 {object} data.BaseResponse{data=[]data.TaskGroupVO}
// @Failure 500 {object} data.BaseResponse
// @Router /task-ms/v1/task-groups [get]
func ListTaskGroups(c *gin.Context) {
	ret, err := service.GetTaskGroupService().ListTaskGroups(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, data.BaseResponse{ErrMsg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: ret})
}

// PublishTaskGroup publishes a task group.
//
// @Summary Publish TaskGroup
// @Description Publish a draft task group.
// @Tags TaskGroup
// @Accept json
// @Produce json
// @Param task_group_id path int true "Task group ID"
// @Param body body data.PublishStatusVO false "Publish status"
// @Success 200 {object} data.BaseResponse{data=data.PublishStatusVO}
// @Failure 400 {object} data.BaseResponse
// @Failure 404 {object} data.BaseResponse
// @Failure 500 {object} data.BaseResponse
// @Router /task-ms/v1/task-groups/{task_group_id} [patch]
func PublishTaskGroup(c *gin.Context) {
	groupID, err := strconv.Atoi(c.Param("task_group_id"))
	if err != nil || groupID <= 0 {
		c.JSON(http.StatusBadRequest, data.BaseResponse{ErrMsg: "invalid task group id"})
		return
	}
	ret, err := service.GetTaskGroupService().PublishTaskGroup(c.Request.Context(), groupID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: ret})
}

func writeServiceError(c *gin.Context, err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "not found"):
		c.JSON(http.StatusNotFound, data.BaseResponse{ErrMsg: msg})
	case strings.Contains(msg, "cannot be modified"),
		strings.Contains(msg, "is required"),
		strings.Contains(msg, "invalid"),
		strings.Contains(msg, "is published"):
		c.JSON(http.StatusBadRequest, data.BaseResponse{ErrMsg: msg})
	default:
		c.JSON(http.StatusInternalServerError, data.BaseResponse{ErrMsg: msg})
	}
}
