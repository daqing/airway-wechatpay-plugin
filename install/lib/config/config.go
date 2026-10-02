// Package config loads the WeChat Pay merchant settings from the
// environment, following Airway's env-var configuration convention.
package config

import (
	"fmt"
	"strings"

	"github.com/daqing/airway/lib/utils"
)

const (
	EnvMchID          = "WECHATPAY_MCH_ID"
	EnvAppID          = "WECHATPAY_APP_ID"
	EnvPrivateKeyPath = "WECHATPAY_PRIVATE_KEY_PATH"
	EnvCertSerialNo   = "WECHATPAY_CERT_SERIAL_NO"
	EnvAPIv3Key       = "WECHATPAY_API_V3_KEY"
	EnvNotifyURL      = "WECHATPAY_NOTIFY_URL"
)

// Config holds the direct-merchant (直连商户) settings WeChat Pay requires.
type Config struct {
	MchID          string
	AppID          string
	PrivateKeyPath string
	CertSerialNo   string
	APIv3Key       string
	// NotifyURL is the public base URL of the host application, e.g.
	// "https://pay.example.com". The plugin appends its own mount path and
	// notify routes when it registers callback URLs with WeChat Pay.
	NotifyURL string
}

// Load reads every WECHATPAY_* variable and fails listing all missing ones.
func Load() (Config, error) {
	var cfg Config
	var missing []string

	required := []struct {
		key string
		dst *string
	}{
		{EnvMchID, &cfg.MchID},
		{EnvAppID, &cfg.AppID},
		{EnvPrivateKeyPath, &cfg.PrivateKeyPath},
		{EnvCertSerialNo, &cfg.CertSerialNo},
		{EnvAPIv3Key, &cfg.APIv3Key},
		{EnvNotifyURL, &cfg.NotifyURL},
	}

	for _, r := range required {
		value, err := utils.GetEnv(r.key)
		if err != nil || strings.TrimSpace(value) == "" {
			missing = append(missing, r.key)
			continue
		}
		*r.dst = value
	}

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("wechatpay: missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}
