// Package conf provides configuration structures for the application.
package conf

import (
	"time"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
)

// ServerConfig holds the HTTP/gRPC server configuration.
type ServerConfig struct {
	Port    int            `json:"port"`
	HTTP    *HTTPConfig    `json:"http"`
	GRPC    *GRPCConfig    `json:"grpc"`
	Timeout time.Duration  `json:"timeout"`
}

// HTTPConfig holds the HTTP server configuration.
type HTTPConfig struct {
	Network string `json:"network"`
	Addr    string `json:"addr"`
	Timeout int    `json:"timeout"`
}

// GRPCConfig holds the gRPC server configuration.
type GRPCConfig struct {
	Network string `json:"network"`
	Addr    string `json:"addr"`
	Timeout int    `json:"timeout"`
}

// DatabaseConfig holds the database configuration.
type DatabaseConfig struct {
	Driver string `json:"driver"`
	Source string `json:"source"`
}

// RedisConfig holds the Redis configuration.
type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

// MQConfig holds the message queue configuration.
type MQConfig struct {
	Type  string            `json:"type"`
	Redis *RedisConfig      `json:"redis"`
	Extra map[string]string `json:"extra"`
}

// LogConfig holds the logging configuration.
type LogConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
}

// OTelConfig holds the OpenTelemetry configuration.
type OTelConfig struct {
	Endpoint   string `json:"endpoint"`
	ServiceName string `json:"serviceName"`
}

// WechatConfig holds the WeChat payment configuration.
type WechatConfig struct {
	AppID     string `json:"appid"`
	MchID     string `json:"mchid"`
	Key       string `json:"key"`
	NotifyURL string `json:"notifyUrl"`
}

// AlipayConfig holds the Alipay configuration.
type AlipayConfig struct {
	AppID      string `json:"appid"`
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
	NotifyURL  string `json:"notifyUrl"`
}

// Config holds the complete application configuration.
type Config struct {
	Server   *ServerConfig   `json:"server"`
	Database *DatabaseConfig `json:"database"`
	Redis    *RedisConfig    `json:"redis"`
	MQ       *MQConfig       `json:"mq"`
	Log      *LogConfig      `json:"log"`
	OTel     *OTelConfig     `json:"otel"`
	Wechat   *WechatConfig   `json:"wechat"`
	Alipay   *AlipayConfig   `json:"alipay"`
	Raw      map[string]interface{} `json:"-"`
}

// LoadConfig loads configuration from the given path.
func LoadConfig(path string) (*Config, error) {
	c := config.New(
		config.WithSource(
			file.NewSource(path),
		),
	)
	if err := c.Load(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := c.Scan(&cfg); err != nil {
		return nil, err
	}

	// Store raw values for custom parsing
	cfg.Raw = make(map[string]interface{})
	if err := c.Scan(&cfg.Raw); err != nil {
		// Ignore error, Raw is optional
	}

	return &cfg, nil
}
