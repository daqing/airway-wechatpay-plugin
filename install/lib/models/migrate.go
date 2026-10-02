package models

import "github.com/daqing/airway/lib/migrate/schema"

func init() {
	schema.RegisterChange("20261002100001", "create_wechatpay_orders", func(m *schema.Migrator) {
		m.CreateTable("wechatpay_orders", func(t *schema.Table) {
			t.ID()
			t.String("out_trade_no", 64).Null(false)
			t.String("appid", 32).Null(false)
			t.String("mchid", 32).Null(false)
			t.String("description", 127).Null(false)
			t.String("attach", 127).Null(false).Default("")
			t.BigInt("amount").Null(false)
			t.String("currency", 8).Null(false).Default("CNY")
			t.String("code_url", 512).Null(false).Default("")
			t.String("trade_state", 32).Null(false).Default(TradeStateNotPay)
			t.String("transaction_id", 64).Null(false).Default("")
			t.DateTime("paid_at").Null(true)
			t.Text("notify_raw").Null(true)
			t.Timestamps()

			t.UniqueIndex("out_trade_no")
			t.Index("transaction_id")
		})
	})

	schema.RegisterChange("20261002100002", "create_wechatpay_refunds", func(m *schema.Migrator) {
		m.CreateTable("wechatpay_refunds", func(t *schema.Table) {
			t.ID()
			t.String("out_refund_no", 64).Null(false)
			t.String("out_trade_no", 64).Null(false)
			t.String("refund_id", 64).Null(false).Default("")
			t.BigInt("amount").Null(false)
			t.BigInt("total").Null(false)
			t.String("reason", 127).Null(false).Default("")
			t.String("status", 32).Null(false).Default(RefundStatusProcessing)
			t.DateTime("refunded_at").Null(true)
			t.Text("notify_raw").Null(true)
			t.Timestamps()

			t.UniqueIndex("out_refund_no")
			t.Index("out_trade_no")
		})
	})
}
