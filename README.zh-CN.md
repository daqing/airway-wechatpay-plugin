# github.com/daqing/airway-wechatpay-plugin

一个 [Airway](https://github.com/daqing/airway) 插件,基于微信支付官方
[wechatpay-go](https://github.com/wechatpay-apiv3/wechatpay-go) SDK,
为宿主应用接入微信支付(APIv3,直连商户模式)。

功能:

- **Native 扫码支付**:创建订单、获取 `code_url`,生成二维码供用户扫码。
- **查单与关单**:主动向微信查询/关闭订单,并同步到本地
  `wechatpay_orders` 表。
- **退款**:退款记录保存在本地 `wechatpay_refunds` 表。
- **支付/退款回调端点**:验签 + 解密;支付/退款成功后同步调用宿主注册的
  回调函数,回调返回错误时插件向微信返回失败码,触发微信重试通知。
- 本地订单/退款持久化使用 Airway 的 Go 代码迁移(MySQL 和 SQLite 都支持,
  无需安装 SQL 文件)。
- 微信平台证书由 SDK 自动下载和轮换。

## 安装

在宿主项目中:

```bash
go run . plugin:install github.com/daqing/airway-wechatpay-plugin
go run . db:migrate
```

`plugin:install` 会把本模块的空白导入加入宿主的 `plugins.go` 并执行
`go get`。数据表由插件的 Go 代码迁移创建,`db:migrate` 会自动识别。

## 配置

插件在启动时从环境变量读取商户配置,缺失任一项都会直接报错:

| 变量 | 含义 |
| --- | --- |
| `WECHATPAY_MCH_ID` | 直连商户号 |
| `WECHATPAY_APP_ID` | 关联的小程序/公众号/应用的 AppID |
| `WECHATPAY_PRIVATE_KEY_PATH` | 商户私钥 PEM 文件路径 |
| `WECHATPAY_CERT_SERIAL_NO` | 商户证书序列号 |
| `WECHATPAY_API_V3_KEY` | APIv3 密钥(用于解密回调报文) |
| `WECHATPAY_NOTIFY_URL` | 宿主的公网基础地址,如 `https://pay.example.com` |

插件向微信注册回调地址时会拼上自己的挂载路径:支付回调为
`POST <WECHATPAY_NOTIFY_URL>/api/v1/wechatpay/notify`,退款回调为
`POST <WECHATPAY_NOTIFY_URL>/api/v1/wechatpay/notify/refund`。这两个端点必须
保持公网可访问且不带鉴权——调用方是微信服务器。如果宿主把插件路由挂载到了
其他前缀,需要相应调整。

## 在宿主应用中使用

```go
import wechatpay "github.com/daqing/airway-wechatpay-plugin"

// 创建 Native 订单(金额单位为分)。
order, err := wechatpay.PrepayNative(ctx, wechatpay.PrepayParams{
    Description: "专业版,一个月",
    Amount:      9900,
    Attach:      "user_id=42",
})
// 把 order.CodeURL 渲染成二维码给用户扫码。

// 支付成功后的业务处理——只注册一次,一般在 init 或启动钩子中。
wechatpay.OnPaid(func(ctx context.Context, order *wechatpay.Order) error {
    // 发货/开通权益。返回 error 会让微信重试通知,处理失败不会丢单。
    return activateOrder(ctx, order.OutTradeNo)
})

wechatpay.OnRefunded(func(ctx context.Context, refund *wechatpay.Refund) error {
    return markRefunded(ctx, refund.OutRefundNo)
})

// 其他操作。
order, err = wechatpay.QueryOrder(ctx, order.OutTradeNo) // 从微信同步订单状态
order, err = wechatpay.CloseOrder(ctx, order.OutTradeNo) // 关闭未支付订单
refund, err := wechatpay.CreateRefund(ctx, wechatpay.RefundParams{
    OutTradeNo: order.OutTradeNo,
    Amount:     9900, // 支持部分退款,不超过订单总额
    Reason:     "客户申请退款",
})
```

`OutTradeNo` / `OutRefundNo` 可以在参数中显式传入(宿主想用自己的单号体系时);
不传则由插件生成随机单号。重复单号会被拒绝;已支付订单(或已成功的退款)
收到重复通知时会直接应答成功,不会再次触发宿主回调。

模型同时暴露在 `airway repl` 中,名为 `WechatpayOrder` 和 `WechatpayRefund`。

## 仓库结构

内容通过两种方式到达宿主:`install/lib/` 下的代码通过模块导入编译进宿主
二进制;`install/host/` 和 `install/deps/` 下的文件由 `plugin:install` 拷贝进
宿主项目。本插件的所有内容都是编译进二进制的代码——迁移是用
`schema.RegisterChange` 注册的 Go 代码,无需安装 SQL 文件。

完整的插件开发指南见
https://github.com/daqing/airway/blob/main/docs/plugin.md 。
