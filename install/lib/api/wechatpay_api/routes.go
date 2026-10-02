package wechatpay_api

import (
	"github.com/gin-gonic/gin"
)

// Routes registers the plugin's HTTP endpoints. The group is already mounted
// at the plugin's MountPath. Both routes must stay unauthenticated — WeChat
// Pay's servers call them.
func Routes(r *gin.RouterGroup) {
	r.POST("/notify", NotifyAction)
	r.POST("/notify/refund", RefundNotifyAction)
}
