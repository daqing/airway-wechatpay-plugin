package wechatpay_api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/daqing/airway-wechatpay-plugin/install/lib/service"
)

// NotifyAction answers WeChat Pay payment notifications. WeChat Pay expects
// {"code":"SUCCESS"} on success and a non-2xx status with {"code":"FAIL"}
// otherwise, in which case it retries the notification.
func NotifyAction(c *gin.Context) {
	if err := service.HandlePaymentNotify(c.Request.Context(), c.Request); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "FAIL", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "success"})
}

// RefundNotifyAction answers WeChat Pay refund notifications.
func RefundNotifyAction(c *gin.Context) {
	if err := service.HandleRefundNotify(c.Request.Context(), c.Request); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "FAIL", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": "SUCCESS", "message": "success"})
}
