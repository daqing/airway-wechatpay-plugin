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
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"

	"github.com/daqing/airway-wechatpay-plugin/install/lib/client"
	"github.com/daqing/airway-wechatpay-plugin/install/lib/models"
)

// PrepayParams describes a Native payment order. Amounts are in cents (分).
type PrepayParams struct {
	// OutTradeNo is the merchant order number; a random one is generated
	// when empty.
	OutTradeNo string
	// Description shows on the WeChat Pay payment page.
	Description string
	// Amount is the order total in cents.
	Amount int64
	// Currency defaults to CNY.
	Currency string
	// Attach is opaque data echoed back in payment notifies.
	Attach string
	// TimeExpire, when non-zero, sets the order's expiration time.
	TimeExpire time.Time
}

// PrepayNative creates a Native payment order and returns it with CodeURL
// set — render that URL as a QR code for the buyer to scan.
func PrepayNative(ctx context.Context, p PrepayParams) (*models.Order, error) {
	cfg, err := client.Config()
	if err != nil {
		return nil, err
	}

	coreClient, err := client.Core()
	if err != nil {
		return nil, err
	}

	if p.Description == "" {
		return nil, errors.New("wechatpay: PrepayParams.Description is required")
	}
	if p.Amount <= 0 {
		return nil, errors.New("wechatpay: PrepayParams.Amount must be positive (cents)")
	}

	outTradeNo := p.OutTradeNo
	if outTradeNo == "" {
		outTradeNo = utils.RandomHex(16)
	}

	currency := p.Currency
	if currency == "" {
		currency = "CNY"
	}

	existing, err := repo.FindOneBy[models.Order](sql.H{"out_trade_no": outTradeNo})
	if err != nil && !errors.Is(err, stdsql.ErrNoRows) {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("wechatpay: order %s already exists", outTradeNo)
	}

	order, err := repo.CreateFrom[models.Order](sql.H{
		"out_trade_no": outTradeNo,
		"appid":        cfg.AppID,
		"mchid":        cfg.MchID,
		"description":  p.Description,
		"attach":       p.Attach,
		"amount":       p.Amount,
		"currency":     currency,
		"trade_state":  models.TradeStateNotPay,
	})
	if err != nil {
		return nil, fmt.Errorf("wechatpay: insert order: %w", err)
	}

	req := native.PrepayRequest{
		Appid:       core.String(cfg.AppID),
		Mchid:       core.String(cfg.MchID),
		Description: core.String(p.Description),
		OutTradeNo:  core.String(outTradeNo),
		NotifyUrl:   core.String(client.PaymentNotifyURL()),
		Amount: &native.Amount{
			Total:    core.Int64(p.Amount),
			Currency: core.String(currency),
		},
	}
	if p.Attach != "" {
		req.Attach = core.String(p.Attach)
	}
	if !p.TimeExpire.IsZero() {
		req.TimeExpire = core.Time(p.TimeExpire)
	}

	svc := native.NativeApiService{Client: coreClient}
	resp, _, err := svc.Prepay(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("wechatpay: prepay order %s: %w", outTradeNo, err)
	}

	codeURL := ""
	if resp.CodeUrl != nil {
		codeURL = *resp.CodeUrl
	}

	if err := repo.UpdateByID[models.Order](order.ID, sql.H{
		"code_url":   codeURL,
		"updated_at": time.Now(),
	}); err != nil {
		return nil, fmt.Errorf("wechatpay: save code_url for order %s: %w", outTradeNo, err)
	}

	order.CodeURL = codeURL
	return order, nil
}

// QueryOrder asks WeChat Pay for the order's current state and syncs it into
// the local record.
func QueryOrder(ctx context.Context, outTradeNo string) (*models.Order, error) {
	cfg, err := client.Config()
	if err != nil {
		return nil, err
	}

	coreClient, err := client.Core()
	if err != nil {
		return nil, err
	}

	svc := native.NativeApiService{Client: coreClient}
	txn, _, err := svc.QueryOrderByOutTradeNo(ctx, native.QueryOrderByOutTradeNoRequest{
		OutTradeNo: core.String(outTradeNo),
		Mchid:      core.String(cfg.MchID),
	})
	if err != nil {
		return nil, fmt.Errorf("wechatpay: query order %s: %w", outTradeNo, err)
	}

	return applyTransaction(ctx, txn, "")
}

// CloseOrder closes an unpaid order at WeChat Pay and marks it CLOSED
// locally.
func CloseOrder(ctx context.Context, outTradeNo string) (*models.Order, error) {
	cfg, err := client.Config()
	if err != nil {
		return nil, err
	}

	coreClient, err := client.Core()
	if err != nil {
		return nil, err
	}

	order, err := findOrder(outTradeNo)
	if err != nil {
		return nil, err
	}
	if order.TradeState == models.TradeStateSuccess {
		return nil, fmt.Errorf("wechatpay: order %s is already paid and cannot be closed", outTradeNo)
	}

	svc := native.NativeApiService{Client: coreClient}
	if _, err := svc.CloseOrder(ctx, native.CloseOrderRequest{
		OutTradeNo: core.String(outTradeNo),
		Mchid:      core.String(cfg.MchID),
	}); err != nil {
		return nil, fmt.Errorf("wechatpay: close order %s: %w", outTradeNo, err)
	}

	if err := repo.UpdateByID[models.Order](order.ID, sql.H{
		"trade_state": models.TradeStateClosed,
		"updated_at":  time.Now(),
	}); err != nil {
		return nil, err
	}

	return repo.FindByID[models.Order](order.ID)
}

func findOrder(outTradeNo string) (*models.Order, error) {
	order, err := repo.FindOneBy[models.Order](sql.H{"out_trade_no": outTradeNo})
	if errors.Is(err, stdsql.ErrNoRows) {
		return nil, fmt.Errorf("wechatpay: unknown order %s", outTradeNo)
	}
	if err != nil {
		return nil, err
	}
	return order, nil
}

// applyTransaction syncs a WeChat Pay transaction into the local order.
// notifyRaw is the decrypted callback payload, empty for active queries.
// When the notify flips the order to SUCCESS, the registered PaidHandler
// runs synchronously; its error propagates so WeChat Pay retries.
func applyTransaction(ctx context.Context, txn *payments.Transaction, notifyRaw string) (*models.Order, error) {
	outTradeNo := ""
	if txn.OutTradeNo != nil {
		outTradeNo = *txn.OutTradeNo
	}

	order, err := findOrder(outTradeNo)
	if err != nil {
		return nil, err
	}

	fromNotify := notifyRaw != ""
	if fromNotify && order.TradeState == models.TradeStateSuccess {
		// Duplicate notification for an already-paid order: answer success
		// without touching state or invoking the host handler again.
		return order, nil
	}

	vals := sql.H{"updated_at": time.Now()}

	tradeState := ""
	if txn.TradeState != nil {
		tradeState = *txn.TradeState
		vals["trade_state"] = tradeState
	}
	if txn.TransactionId != nil && *txn.TransactionId != "" {
		vals["transaction_id"] = *txn.TransactionId
	}
	if txn.SuccessTime != nil && *txn.SuccessTime != "" {
		if paidAt, err := time.Parse(time.RFC3339, *txn.SuccessTime); err == nil {
			vals["paid_at"] = paidAt
		}
	}
	if notifyRaw != "" {
		vals["notify_raw"] = notifyRaw
	}

	if err := repo.UpdateByID[models.Order](order.ID, vals); err != nil {
		return nil, err
	}

	order, err = repo.FindByID[models.Order](order.ID)
	if err != nil {
		return nil, err
	}

	if fromNotify && tradeState == models.TradeStateSuccess && paidHandler != nil {
		if err := paidHandler(ctx, order); err != nil {
			return nil, fmt.Errorf("wechatpay: OnPaid handler for order %s: %w", outTradeNo, err)
		}
	}

	return order, nil
}
