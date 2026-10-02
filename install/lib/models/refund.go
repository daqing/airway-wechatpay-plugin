package models

import (
	"time"

	"github.com/daqing/airway/lib/sql"
)

// Refund statuses reported by WeChat Pay.
const (
	RefundStatusSuccess    = "SUCCESS"
	RefundStatusClosed     = "CLOSED"
	RefundStatusProcessing = "PROCESSING"
	RefundStatusAbnormal   = "ABNORMAL"
)

// Refund is a WeChat Pay refund against an Order. Amounts are in cents (分).
type Refund struct {
	ID          sql.IdType `db:"id" json:"id"`
	OutRefundNo string     `db:"out_refund_no" json:"out_refund_no"`
	OutTradeNo  string     `db:"out_trade_no" json:"out_trade_no"`
	RefundID    string     `db:"refund_id" json:"refund_id"`
	Amount      int64      `db:"amount" json:"amount"`
	Total       int64      `db:"total" json:"total"`
	Reason      string     `db:"reason" json:"reason"`
	Status      string     `db:"status" json:"status"`
	RefundedAt  *time.Time `db:"refunded_at" json:"refunded_at"`
	NotifyRaw   *string    `db:"notify_raw" json:"notify_raw"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

func (Refund) TableName() string { return "wechatpay_refunds" }
