package models

import (
	"time"

	"github.com/daqing/airway/lib/sql"
)

// Trade states reported by WeChat Pay.
const (
	TradeStateNotPay   = "NOTPAY"
	TradeStateSuccess  = "SUCCESS"
	TradeStateClosed   = "CLOSED"
	TradeStateRefund   = "REFUND"
	TradeStatePayError = "PAYERROR"
)

// Order is a WeChat Pay payment order. Amounts are in cents (分).
type Order struct {
	ID            sql.IdType `db:"id" json:"id"`
	OutTradeNo    string     `db:"out_trade_no" json:"out_trade_no"`
	Appid         string     `db:"appid" json:"appid"`
	Mchid         string     `db:"mchid" json:"mchid"`
	Description   string     `db:"description" json:"description"`
	Attach        string     `db:"attach" json:"attach"`
	Amount        int64      `db:"amount" json:"amount"`
	Currency      string     `db:"currency" json:"currency"`
	CodeURL       string     `db:"code_url" json:"code_url"`
	TradeState    string     `db:"trade_state" json:"trade_state"`
	TransactionID string     `db:"transaction_id" json:"transaction_id"`
	PaidAt        *time.Time `db:"paid_at" json:"paid_at"`
	NotifyRaw     *string    `db:"notify_raw" json:"notify_raw"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at" json:"updated_at"`
}

func (Order) TableName() string { return "wechatpay_orders" }
