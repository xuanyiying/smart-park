package biz

import (
	"time"
)

// Config holds the business logic configuration.
type Config struct {
	// Lock configuration
	LockTTL time.Duration

	// Device configuration
	DeviceOnlineThreshold time.Duration

	// Messages
	Messages MessagesConfig
}

// MessagesConfig holds display messages.
type MessagesConfig struct {
	Welcome        string
	MonthlyWelcome string
	VIPWelcome     string
	DuplicateEntry string
	NoEntryRecord  string
	PleasePay      string
	FreePass       string
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		LockTTL:               10 * time.Second,
		DeviceOnlineThreshold: 5 * time.Minute,
		Messages: MessagesConfig{
			Welcome:        "欢迎光临",
			MonthlyWelcome: "月卡车，欢迎光临",
			VIPWelcome:     "VIP 车辆，欢迎光临",
			DuplicateEntry: "车辆已在场内，请勿重复入场",
			NoEntryRecord:  "未找到入场记录",
			PleasePay:      "请缴费",
			FreePass:       "免费放行",
		},
	}
}
