// Package service implements the WeChat Pay business flows: Native prepay,
// order query/close, refunds, and the payment/refund notify callbacks.
package service

import (
	"context"

	"github.com/daqing/airway-wechatpay-plugin/install/lib/models"
)

// PaidHandler is invoked synchronously after a payment notify flips an order
// to SUCCESS. Returning an error makes the plugin answer WeChat Pay with a
// failure code, so WeChat Pay retries the notification.
type PaidHandler func(ctx context.Context, order *models.Order) error

// RefundedHandler is invoked synchronously after a refund notify flips a
// refund to SUCCESS. The retry semantics match PaidHandler.
type RefundedHandler func(ctx context.Context, refund *models.Refund) error

var (
	paidHandler     PaidHandler
	refundedHandler RefundedHandler
)

// OnPaid registers the host's payment-success handler. Registering twice
// panics, mirroring plugin.Register.
func OnPaid(h PaidHandler) {
	if paidHandler != nil {
		panic("wechatpay: OnPaid handler already registered")
	}
	paidHandler = h
}

// OnRefunded registers the host's refund-success handler.
func OnRefunded(h RefundedHandler) {
	if refundedHandler != nil {
		panic("wechatpay: OnRefunded handler already registered")
	}
	refundedHandler = h
}
