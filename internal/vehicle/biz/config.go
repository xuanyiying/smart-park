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

	// Plate recognition configuration
	MinConfidence float64

	// Messages
	Messages MessagesConfig
}

// MessagesConfig holds display messages.
type MessagesConfig struct {
	Welcome        string
	MonthlyWelcome string
	VIPWelcome     string
	DuplicateEntry string
	DuplicateExit  string
	NoEntryRecord  string
	PleasePay      string
	FreePass       string
	ValidationError string
	SystemError    string
	FallbackMode   string
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		LockTTL:               10 * time.Second,
		DeviceOnlineThreshold: 5 * time.Minute,
		MinConfidence:         0.7,
		Messages: MessagesConfig{
			Welcome:         "欢迎光临",
			MonthlyWelcome:  "月卡车，欢迎光临",
			VIPWelcome:      "VIP 车辆，欢迎光临",
			DuplicateEntry:  "车辆已在场内，请勿重复入场",
			DuplicateExit:   "车辆正在出场中，请勿重复操作",
			NoEntryRecord:   "未找到入场记录",
			PleasePay:       "请缴费",
			FreePass:        "免费放行",
			ValidationError: "信息验证失败，请重试",
			SystemError:     "系统错误，请联系管理员",
			FallbackMode:    "系统维护中，人工处理",
		},
	}
}
