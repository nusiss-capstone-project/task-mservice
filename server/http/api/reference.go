package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nusiss-capstone-project/task-mservice/server/http/data"
	"github.com/nusiss-capstone-project/task-mservice/server/service"
)

// ListDataMetrics returns configured data metrics.
//
// @Summary List DataMetric
// @Description List data metrics used for task condition configuration.
// @Tags Reference
// @Produce json
// @Success 200 {object} data.BaseResponse{data=[]data.DataMetricVO}
// @Failure 500 {object} data.BaseResponse
// @Router /task-ms/v1/data-metrics [get]
func ListDataMetrics(c *gin.Context) {
	ret, err := service.GetReferenceService().ListDataMetrics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, data.BaseResponse{ErrMsg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: ret})
}

// ListDataMetricOperators returns configured metric operators.
//
// @Summary List DataMetricOperator
// @Description List metric operators used for task condition configuration.
// @Tags Reference
// @Produce json
// @Success 200 {object} data.BaseResponse{data=[]data.MetricOperatorVO}
// @Failure 500 {object} data.BaseResponse
// @Router /task-ms/v1/data-metric-operators [get]
func ListDataMetricOperators(c *gin.Context) {
	ret, err := service.GetReferenceService().ListMetricOperators(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, data.BaseResponse{ErrMsg: err.Error()})
		return
	}
	c.JSON(http.StatusOK, data.BaseResponse{Data: ret})
}
