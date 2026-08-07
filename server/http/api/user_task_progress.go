package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	commonauth "github.com/nusiss-capstone-project/identity-mservice/common/auth"
	"github.com/nusiss-capstone-project/task-mservice/server/http/data"
	"github.com/nusiss-capstone-project/task-mservice/server/service"
)

// AdminListUserTaskProgress lists a user's task progress in a task group (admin).
//
// @Summary Admin user task progress in group
// @Description List published tasks in a group with the given user's execution status.
// @Tags Task Progress
// @Produce json
// @Param task_group_id path int true "Task group ID"
// @Param user_id path int true "User ID"
// @Success 200 {object} data.BaseResponse{data=[]data.UserTaskProgressVO}
// @Failure 400 {object} data.BaseResponse
// @Failure 404 {object} data.BaseResponse
// @Failure 500 {object} data.BaseResponse
// @Router /task-ms/v1/admin/tasks/task_group/{task_group_id}/users/{user_id} [get]
func AdminListUserTaskProgress(c *gin.Context) {
	groupID, err := parsePathID(c, "task_group_id")
	if err != nil {
		return
	}
	userID, err := parsePathID(c, "user_id")
	if err != nil {
		return
	}
	ret, err := service.GetUserTaskProgressService().ListUserTaskProgressInGroup(c.Request.Context(), groupID, userID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: ret})
}

// WebListUserTaskProgress lists the current user's task progress in a task group.
//
// @Summary Web user task progress in group
// @Description List published tasks in a group with the authenticated user's execution status.
// @Tags Task Progress
// @Produce json
// @Param task_group_id path int true "Task group ID"
// @Success 200 {object} data.BaseResponse{data=[]data.UserTaskProgressVO}
// @Failure 400 {object} data.BaseResponse
// @Failure 401 {object} data.BaseResponse
// @Failure 404 {object} data.BaseResponse
// @Failure 500 {object} data.BaseResponse
// @Router /task-ms/v1/web/tasks/task_group/{task_group_id} [get]
func WebListUserTaskProgress(c *gin.Context) {
	groupID, err := parsePathID(c, "task_group_id")
	if err != nil {
		return
	}
	userID, ok := commonauth.GetUserID(c.Request.Context())
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, data.BaseResponse{ErrMsg: "unauthorized"})
		return
	}
	ret, err := service.GetUserTaskProgressService().ListUserTaskProgressInGroup(
		c.Request.Context(), groupID, int(userID),
	)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: ret})
}
