package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type Validator struct {
	requiredVars []string
}

func NewValidator() *Validator {
	return &Validator{
		requiredVars: []string{
			"DB_HOST",
			"DB_PASSWORD",
			"JWT_SECRET_KEY",
			"WECHAT_APP_ID",
			"WECHAT_MCH_ID",
			"ALIPAY_APP_ID",
		},
	}
}

func (v *Validator) Validate() error {
	var missing []string

	for _, varName := range v.requiredVars {
		value := os.Getenv(varName)
		if value == "" || strings.Contains(value, "your_") {
			missing = append(missing, varName)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing or invalid environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

func (v *Validator) ValidateDatabaseConfig() error {
	password := os.Getenv("DB_PASSWORD")
	if password == "postgres" || password == "password" {
		return errors.New("database password is too weak")
	}
	return nil
}

func (v *Validator) ValidateJWTSecret() error {
	secret := os.Getenv("JWT_SECRET_KEY")
	if len(secret) < 32 {
		return errors.New("JWT secret key must be at least 32 characters")
	}
	return nil
}

func (v *Validator) ValidateWechatConfig() error {
	appID := os.Getenv("WECHAT_APP_ID")
	mchID := os.Getenv("WECHAT_MCH_ID")

	if appID == "" || strings.Contains(appID, "your_") {
		return errors.New("invalid WeChat App ID")
	}

	if mchID == "" || strings.Contains(mchID, "your_") {
		return errors.New("invalid WeChat Merchant ID")
	}

	return nil
}

func (v *Validator) ValidateAlipayConfig() error {
	appID := os.Getenv("ALIPAY_APP_ID")

	if appID == "" || strings.Contains(appID, "your_") {
		return errors.New("invalid Alipay App ID")
	}

	return nil
}
