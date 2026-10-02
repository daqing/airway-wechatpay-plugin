package service

import (
	"context"
	stdsql "database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/sql"
	"github.com/daqing/airway/lib/utils"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"

	"github.com/daqing/airway-wechatpay-plugin/install/lib/client"
	"github.com/daqing/airway-wechatpay-plugin/install/lib/models"
)

// RefundParams describes a refund against a paid order. Amounts are in
// cents (分).
type RefundParams struct {
	// OutTradeNo identifies the order to refund.
	OutTradeNo string
	// OutRefundNo is the merchant refund number; a random one is generated
	// when empty.
	OutRefundNo string
	// Amount is the refund amount in cents; it must not exceed the order
	// total. Partial refunds are allowed.
	Amount int64
	// Reason shows in the refund message sent to the buyer.
	Reason string
}

// Refund applies for a refund at WeChat Pay and records it locally. The
// refund result arrives asynchronously at the refund notify route.
func Refund(ctx context.Context, p RefundParams) (*models.Refund, error) {
	coreClient, err := client.Core()
	if err != nil {
		return nil, err
	}

	order, err := findOrder(p.OutTradeNo)
	if err != nil {
		return nil, err
	}
	if order.TradeState != models.TradeStateSuccess && order.TradeState != models.TradeStateRefund {
		return nil, fmt.Errorf("wechatpay: order %s is %s, only paid orders can be refunded", p.OutTradeNo, order.TradeState)
	}
	if p.Amount <= 0 {
		return nil, errors.New("wechatpay: RefundParams.Amount must be positive (cents)")
	}
	if p.Amount > order.Amount {
		return nil, fmt.Errorf("wechatpay: refund amount %d exceeds order amount %d", p.Amount, order.Amount)
	}

	outRefundNo := p.OutRefundNo
	if outRefundNo == "" {
		outRefundNo = "R" + utils.RandomHex(15)
	}

	existing, err := repo.FindOneBy[models.Refund](sql.H{"out_refund_no": outRefundNo})
	if err != nil && !errors.Is(err, stdsql.ErrNoRows) {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("wechatpay: refund %s already exists", outRefundNo)
	}

	svc := refunddomestic.RefundsApiService{Client: coreClient}
	resp, _, err := svc.Create(ctx, refunddomestic.CreateRequest{
		OutTradeNo:  core.String(p.OutTradeNo),
		OutRefundNo: core.String(outRefundNo),
		Reason:      core.String(p.Reason),
		NotifyUrl:   core.String(client.RefundNotifyURL()),
		Amount: &refunddomestic.AmountReq{
			Refund:   core.Int64(p.Amount),
			Total:    core.Int64(order.Amount),
			Currency: core.String(order.Currency),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("wechatpay: create refund %s for order %s: %w", outRefundNo, p.OutTradeNo, err)
	}

	status := models.RefundStatusProcessing
	if resp.Status != nil {
		status = string(*resp.Status)
	}

	refundID := ""
	if resp.RefundId != nil {
		refundID = *resp.RefundId
	}

	refund, err := repo.CreateFrom[models.Refund](sql.H{
		"out_refund_no": outRefundNo,
		"out_trade_no":  p.OutTradeNo,
		"refund_id":     refundID,
		"amount":        p.Amount,
		"total":         order.Amount,
		"reason":        p.Reason,
		"status":        status,
	})
	if err != nil {
		return nil, fmt.Errorf("wechatpay: insert refund %s: %w", outRefundNo, err)
	}

	if order.TradeState != models.TradeStateRefund {
		if err := repo.UpdateByID[models.Order](order.ID, sql.H{
			"trade_state": models.TradeStateRefund,
			"updated_at":  time.Now(),
		}); err != nil {
			return nil, err
		}
	}

	return refund, nil
}

// applyRefundNotify syncs a decrypted refund notify into the local refund
// and invokes the registered RefundedHandler on SUCCESS.
func applyRefundNotify(ctx context.Context, rn *refunddomestic.Refund, notifyRaw string) (*models.Refund, error) {
	outRefundNo := ""
	if rn.OutRefundNo != nil {
		outRefundNo = *rn.OutRefundNo
	}

	refund, err := repo.FindOneBy[models.Refund](sql.H{"out_refund_no": outRefundNo})
	if errors.Is(err, stdsql.ErrNoRows) {
		return nil, fmt.Errorf("wechatpay: unknown refund %s", outRefundNo)
	}
	if err != nil {
		return nil, err
	}

	if refund.Status == models.RefundStatusSuccess {
		return refund, nil
	}

	vals := sql.H{
		"notify_raw": notifyRaw,
		"updated_at": time.Now(),
	}

	status := ""
	if rn.Status != nil {
		status = string(*rn.Status)
		vals["status"] = status
	}
	if rn.RefundId != nil && *rn.RefundId != "" {
		vals["refund_id"] = *rn.RefundId
	}
	if rn.SuccessTime != nil {
		vals["refunded_at"] = *rn.SuccessTime
	}

	if err := repo.UpdateByID[models.Refund](refund.ID, vals); err != nil {
		return nil, err
	}

	refund, err = repo.FindByID[models.Refund](refund.ID)
	if err != nil {
		return nil, err
	}

	if status == models.RefundStatusSuccess && refundedHandler != nil {
		if err := refundedHandler(ctx, refund); err != nil {
			return nil, fmt.Errorf("wechatpay: OnRefunded handler for refund %s: %w", outRefundNo, err)
		}
	}

	return refund, nil
}
