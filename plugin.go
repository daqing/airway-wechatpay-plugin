// Package wechatpayplugin implements the wechatpay Airway plugin.
//
// The plugin integrates WeChat Pay (微信支付 APIv3, direct merchant) into an
// Airway host application: Native prepay orders, order query/close, refunds,
// and verified payment/refund notify callbacks. Configure it with the
// WECHATPAY_* environment variables (see install/lib/config) and register
// business handlers with OnPaid / OnRefunded.
package wechatpayplugin

import (
	"context"

	"github.com/daqing/airway/lib/plugin"
	"github.com/gin-gonic/gin"

	"github.com/daqing/airway-wechatpay-plugin/install/lib/api/wechatpay_api"
	"github.com/daqing/airway-wechatpay-plugin/install/lib/client"
	"github.com/daqing/airway-wechatpay-plugin/install/lib/config"
	"github.com/daqing/airway-wechatpay-plugin/install/lib/models"
	"github.com/daqing/airway-wechatpay-plugin/install/lib/service"
)

// Plugin is the wechatpay feature module.
type Plugin struct{}

func (Plugin) Name() string      { return "wechatpay" }
func (Plugin) MountPath() string { return "/api/v1/wechatpay" }

func (Plugin) Routes(r *gin.RouterGroup) {
	wechatpay_api.Routes(r)
}

// Boot loads the WECHATPAY_* environment variables and initializes the
// WeChat Pay SDK client, including automatic platform-certificate updates.
func (p Plugin) Boot() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	return client.Setup(context.Background(), cfg, p.MountPath())
}

// REPLModels exposes the plugin's models to `airway repl`.
func (Plugin) REPLModels() map[string]any {
	return map[string]any{
		"WechatpayOrder":  models.Order{},
		"WechatpayRefund": models.Refund{},
	}
}

func init() {
	plugin.Register(Plugin{})
}

// Host-facing API. The host application creates payments through these
// functions instead of talking to wechatpay-go directly.

type (
	// Order is a WeChat Pay payment order.
	Order = models.Order
	// Refund is a WeChat Pay refund.
	Refund = models.Refund
	// PrepayParams describes a Native payment order (amounts in cents).
	PrepayParams = service.PrepayParams
	// RefundParams describes a refund against a paid order (amounts in cents).
	RefundParams = service.RefundParams
	// PaidHandler runs when an order is paid; its error triggers a WeChat
	// Pay notify retry.
	PaidHandler = service.PaidHandler
	// RefundedHandler runs when a refund succeeds.
	RefundedHandler = service.RefundedHandler
)

// Trade states and refund statuses reported by WeChat Pay.
const (
	TradeStateNotPay   = models.TradeStateNotPay
	TradeStateSuccess  = models.TradeStateSuccess
	TradeStateClosed   = models.TradeStateClosed
	TradeStateRefund   = models.TradeStateRefund
	TradeStatePayError = models.TradeStatePayError

	RefundStatusSuccess    = models.RefundStatusSuccess
	RefundStatusClosed     = models.RefundStatusClosed
	RefundStatusProcessing = models.RefundStatusProcessing
	RefundStatusAbnormal   = models.RefundStatusAbnormal
)

// PrepayNative creates a Native payment order and returns it with CodeURL
// set; render CodeURL as a QR code for the buyer to scan with WeChat.
func PrepayNative(ctx context.Context, p PrepayParams) (*Order, error) {
	return service.PrepayNative(ctx, p)
}

// QueryOrder asks WeChat Pay for the order's current state and syncs the
// local record.
func QueryOrder(ctx context.Context, outTradeNo string) (*Order, error) {
	return service.QueryOrder(ctx, outTradeNo)
}

// CloseOrder closes an unpaid order at WeChat Pay.
func CloseOrder(ctx context.Context, outTradeNo string) (*Order, error) {
	return service.CloseOrder(ctx, outTradeNo)
}

// CreateRefund applies for a refund against a paid order.
func CreateRefund(ctx context.Context, p RefundParams) (*Refund, error) {
	return service.Refund(ctx, p)
}

// OnPaid registers the host's payment-success handler (register once).
func OnPaid(h PaidHandler) {
	service.OnPaid(h)
}

// OnRefunded registers the host's refund-success handler (register once).
func OnRefunded(h RefundedHandler) {
	service.OnRefunded(h)
}
