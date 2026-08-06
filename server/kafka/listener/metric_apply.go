package listener

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/nusiss-capstone-project/task-mservice/server/repository/dao"
	"github.com/nusiss-capstone-project/task-mservice/server/service"
)

const (
	metricKycPassed           = "kyc_passed"
	metricPaymentMethodAdded  = "payment_method_added"
	metricTradeCountAccum     = "trade_count_accum"
	metricTradeAmount         = "trade_amount"
	metricTradeAmountAccum    = "trade_amount_accum"
	metricDepositAmount       = "deposit_amount"
	metricDepositAmountAccum  = "deposit_amount_accum"

	kycStatusPassed   = "PASSED"
	paymentStatusOK   = "pay_succeed"
)

func applyMetricEvent(
	ctx context.Context,
	userID int,
	metricCode, metricValue string,
	eventTime time.Time,
	bizID string,
) error {
	metric, err := dao.GetDataMetricDao().GetByCode(ctx, metricCode)
	if err != nil {
		return fmt.Errorf("get data metric %s: %w", metricCode, err)
	}
	if metric == nil {
		return fmt.Errorf("data metric not found: %s", metricCode)
	}
	if err := service.GetUserTaskProgressService().UpdateUserTaskProgress(
		ctx, userID, metric.ID, metricValue, eventTime, bizID,
	); err != nil {
		return fmt.Errorf("update user task progress metric=%s: %w", metricCode, err)
	}
	return nil
}

func eventTimeFromUnixSeconds(sec int64) time.Time {
	return time.Unix(sec, 0).UTC()
}

func orderBizID(orderID int64) string {
	return strconv.FormatInt(orderID, 10)
}
