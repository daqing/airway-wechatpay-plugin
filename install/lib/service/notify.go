package service

import (
	"context"
	"net/http"

	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"

	"github.com/daqing/airway-wechatpay-plugin/install/lib/client"
)

// HandlePaymentNotify verifies, decrypts and applies a WeChat Pay payment
// notification. Any error should be answered with a failure code so WeChat
// Pay retries the notification.
func HandlePaymentNotify(ctx context.Context, req *http.Request) error {
	handler, err := client.NotifyHandler()
	if err != nil {
		return err
	}

	txn := new(payments.Transaction)
	notifyReq, err := handler.ParseNotifyRequest(ctx, req, txn)
	if err != nil {
		return err
	}

	_, err = applyTransaction(ctx, txn, notifyReq.Resource.Plaintext)
	return err
}

// HandleRefundNotify verifies, decrypts and applies a WeChat Pay refund
// notification.
func HandleRefundNotify(ctx context.Context, req *http.Request) error {
	handler, err := client.NotifyHandler()
	if err != nil {
		return err
	}

	refund := new(refunddomestic.Refund)
	notifyReq, err := handler.ParseNotifyRequest(ctx, req, refund)
	if err != nil {
		return err
	}

	_, err = applyRefundNotify(ctx, refund, notifyReq.Resource.Plaintext)
	return err
}
