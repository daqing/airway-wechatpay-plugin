// Package client owns the wechatpay-go core client singleton and the
// notify handler used to verify and decrypt WeChat Pay callbacks.
package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"

	"github.com/daqing/airway-wechatpay-plugin/install/lib/config"
)

var (
	mu           sync.RWMutex
	cfg          config.Config
	coreClient   *core.Client
	notifyHandle *notify.Handler
	notifyBase   string
)

// Setup initializes the SDK from the merchant config. mountPath is the
// plugin's mount path, used to derive the callback URLs sent to WeChat Pay.
// Platform certificates are downloaded and refreshed automatically by the
// SDK's certificate downloader.
func Setup(ctx context.Context, c config.Config, mountPath string) error {
	privateKey, err := utils.LoadPrivateKeyWithPath(c.PrivateKeyPath)
	if err != nil {
		return fmt.Errorf("wechatpay: load merchant private key: %w", err)
	}

	sdkClient, err := core.NewClient(
		ctx,
		option.WithWechatPayAutoAuthCipher(c.MchID, c.CertSerialNo, privateKey, c.APIv3Key),
	)
	if err != nil {
		return fmt.Errorf("wechatpay: build client: %w", err)
	}

	verifier := verifiers.NewSHA256WithRSAVerifier(
		downloader.MgrInstance().GetCertificateVisitor(c.MchID),
	)

	mu.Lock()
	defer mu.Unlock()

	cfg = c
	coreClient = sdkClient
	notifyHandle = notify.NewNotifyHandler(c.APIv3Key, verifier)
	notifyBase = strings.TrimSuffix(c.NotifyURL, "/") + mountPath

	return nil
}

// Core returns the initialized wechatpay-go client.
func Core() (*core.Client, error) {
	mu.RLock()
	defer mu.RUnlock()

	if coreClient == nil {
		return nil, errors.New("wechatpay: client is not initialized — is the plugin compiled in and booted?")
	}

	return coreClient, nil
}

// Config returns the merchant config passed to Setup.
func Config() (config.Config, error) {
	mu.RLock()
	defer mu.RUnlock()

	if coreClient == nil {
		return config.Config{}, errors.New("wechatpay: client is not initialized — is the plugin compiled in and booted?")
	}

	return cfg, nil
}

// NotifyHandler returns the handler that verifies and decrypts WeChat Pay
// callback requests.
func NotifyHandler() (*notify.Handler, error) {
	mu.RLock()
	defer mu.RUnlock()

	if notifyHandle == nil {
		return nil, errors.New("wechatpay: notify handler is not initialized — is the plugin compiled in and booted?")
	}

	return notifyHandle, nil
}

// PaymentNotifyURL is the callback URL registered on payment orders.
func PaymentNotifyURL() string {
	mu.RLock()
	defer mu.RUnlock()
	return notifyBase + "/notify"
}

// RefundNotifyURL is the callback URL registered on refund requests.
func RefundNotifyURL() string {
	mu.RLock()
	defer mu.RUnlock()
	return notifyBase + "/notify/refund"
}
