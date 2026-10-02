# github.com/daqing/airway-wechatpay-plugin

An [Airway](https://github.com/daqing/airway) plugin that integrates
[WeChat Pay](https://pay.weixin.qq.com/) (APIv3, direct merchant) into a host
application — built on the official
[wechatpay-go](https://github.com/wechatpay-apiv3/wechatpay-go) SDK.

Features:

- **Native prepay** (scan-to-pay): create an order, get a `code_url`, render
  it as a QR code.
- **Order query and close** against WeChat Pay, synced into a local
  `wechatpay_orders` table.
- **Refunds**, recorded in a local `wechatpay_refunds` table.
- **Payment and refund notify endpoints** with signature verification and
  decryption; successful payments/refunds invoke a host-registered callback,
  and a callback error makes WeChat Pay retry the notification.
- Local order/refund persistence via Airway's Go-code migrations (MySQL and
  SQLite both work — nothing to install).
- Automatic WeChat Pay platform-certificate download and rotation.

## Install

In the host project:

```bash
go run . plugin:install github.com/daqing/airway-wechatpay-plugin
go run . db:migrate
```

`plugin:install` adds a blank import of this module to the host's
`plugins.go` and runs `go get`. The tables are created by the plugin's
Go-code migrations, which `db:migrate` picks up automatically.

## Configure

The plugin reads its merchant settings from the environment at boot and
fails fast when any is missing:

| Variable | Meaning |
| --- | --- |
| `WECHATPAY_MCH_ID` | Direct merchant ID (商户号) |
| `WECHATPAY_APP_ID` | AppID of the linked mini-program / official account / app |
| `WECHATPAY_PRIVATE_KEY_PATH` | Path to the merchant private key PEM file |
| `WECHATPAY_CERT_SERIAL_NO` | Merchant certificate serial number |
| `WECHATPAY_API_V3_KEY` | APIv3 key (decrypts notify payloads) |
| `WECHATPAY_NOTIFY_URL` | Public base URL of the host, e.g. `https://pay.example.com` |

The plugin appends its own mount path when registering callback URLs with
WeChat Pay: payments notify `POST <WECHATPAY_NOTIFY_URL>/api/v1/wechatpay/notify`,
refunds notify `POST <WECHATPAY_NOTIFY_URL>/api/v1/wechatpay/notify/refund`.
Both endpoints must stay publicly reachable (no auth middleware) — WeChat
Pay's servers call them. If you re-mount the plugin's routes under a
different prefix, adjust accordingly.

## Use from the host application

```go
import wechatpay "github.com/daqing/airway-wechatpay-plugin"

// Create a Native order (amounts are in cents).
order, err := wechatpay.PrepayNative(ctx, wechatpay.PrepayParams{
    Description: "Pro plan, one month",
    Amount:      9900,
    Attach:      "user_id=42",
})
// Render order.CodeURL as a QR code for the buyer to scan.

// Act on payment success — register once, typically in an init or boot hook.
wechatpay.OnPaid(func(ctx context.Context, order *wechatpay.Order) error {
    // Activate the purchase. Returning an error makes WeChat Pay retry
    // the notification, so handler failures are never lost.
    return activateOrder(ctx, order.OutTradeNo)
})

wechatpay.OnRefunded(func(ctx context.Context, refund *wechatpay.Refund) error {
    return markRefunded(ctx, refund.OutRefundNo)
})

// Other operations.
order, err = wechatpay.QueryOrder(ctx, order.OutTradeNo) // sync state from WeChat Pay
order, err = wechatpay.CloseOrder(ctx, order.OutTradeNo) // close an unpaid order
refund, err := wechatpay.CreateRefund(ctx, wechatpay.RefundParams{
    OutTradeNo: order.OutTradeNo,
    Amount:     9900, // partial refunds allowed, up to the order total
    Reason:     "customer request",
})
```

`OutTradeNo` / `OutRefundNo` may be supplied in the params when the host
wants its own numbering scheme; random ones are generated otherwise.
Duplicate numbers are rejected, and duplicate notifications for an
already-paid order (or an already-successful refund) are acknowledged
without re-invoking the host handlers.

The models are also available in `airway repl` as `WechatpayOrder` and
`WechatpayRefund`.

## Repository layout

Content reaches the host in two ways: code under `install/lib/` is compiled
into the host binary through the module import, while files under
`install/host/` and `install/deps/` are copied into the host project by
`plugin:install`. This plugin ships everything as compiled-in code — the
migrations are Go code registered with `schema.RegisterChange`, so no SQL
files need to be installed.

See https://github.com/daqing/airway/blob/main/docs/plugin.md for the full
plugin guide.
