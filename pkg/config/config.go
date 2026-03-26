package config

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	MQ        MQConfig        `mapstructure:"mq"`
	Log       LogConfig       `mapstructure:"log"`
	Telemetry TelemetryConfig `mapstructure:"telemetry"`
	Wechat    WechatConfig    `mapstructure:"wechat"`
	Alipay    AlipayConfig    `mapstructure:"alipay"`
	MQTT      MQTTConfig      `mapstructure:"mqtt"`
	Billing   *BillingConfig  `mapstructure:"billing"`
	Vehicle   *VehicleConfig  `mapstructure:"vehicle"`
	JWT       JWTConfig       `mapstructure:"jwt"`
}

type ServerConfig struct {
	Port    int    `mapstructure:"port"`
	Host    string `mapstructure:"host"`
	Timeout int    `mapstructure:"timeout"`
	Mode    string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Driver       string `mapstructure:"driver"`
	Source       string `mapstructure:"source"`
	MaxOpenConns int    `mapstructure:"maxOpenConns"`
	MaxIdleConns int    `mapstructure:"maxIdleConns"`
	ConnMaxLife  int    `mapstructure:"connMaxLife"`
}

type RedisConfig struct {
	Addr        string `mapstructure:"addr"`
	Password    string `mapstructure:"password"`
	DB          int    `mapstructure:"db"`
	PoolSize    int    `mapstructure:"poolSize"`
	MinIdleConn int    `mapstructure:"minIdleConn"`
}

type MQConfig struct {
	Type     string         `mapstructure:"type"`
	Redis    RedisMQConfig  `mapstructure:"redis"`
	NATS     NATSConfig     `mapstructure:"nats"`
	RocketMQ RocketMQConfig `mapstructure:"rocketmq"`
}

type RedisMQConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	Stream   string `mapstructure:"stream"`
}

type NATSConfig struct {
	URL      string `mapstructure:"url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type RocketMQConfig struct {
	NameServer string `mapstructure:"nameServer"`
	Group      string `mapstructure:"group"`
	AccessKey  string `mapstructure:"accessKey"`
	SecretKey  string `mapstructure:"secretKey"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	OutputPath string `mapstructure:"outputPath"`
	MaxSize    int    `mapstructure:"maxSize"`
	MaxBackups int    `mapstructure:"maxBackups"`
	MaxAge     int    `mapstructure:"maxAge"`
	Compress   bool   `mapstructure:"compress"`
}

type TelemetryConfig struct {
	Enabled     bool    `mapstructure:"enabled"`
	Endpoint    string  `mapstructure:"endpoint"`
	ServiceName string  `mapstructure:"serviceName"`
	SampleRate  float64 `mapstructure:"sampleRate"`
}

type WechatConfig struct {
	AppID          string `mapstructure:"app_id"`
	MchID          string `mapstructure:"mch_id"`
	APIKey         string `mapstructure:"api_key"`
	CertSerialNo   string `mapstructure:"cert_serial_no"`
	PrivateKeyPath string `mapstructure:"private_key_path"`
	PublicKeyPath  string `mapstructure:"public_key_path"`
	NotifyURL      string `mapstructure:"notify_url"`
}

type AlipayConfig struct {
	AppID        string `mapstructure:"app_id"`
	PrivateKey   string `mapstructure:"private_key"`
	PublicKey    string `mapstructure:"public_key"`
	NotifyURL    string `mapstructure:"notify_url"`
	IsProduction bool   `mapstructure:"is_production"`
}

type MQTTConfig struct {
	Broker   string `mapstructure:"broker"`
	Port     int    `mapstructure:"port"`
	ClientID string `mapstructure:"client_id"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type BillingConfig struct {
	Endpoint string `mapstructure:"endpoint"`
	Timeout  int    `mapstructure:"timeout"`
}

type VehicleConfig struct {
	Endpoint string `mapstructure:"endpoint"`
	Timeout  int    `mapstructure:"timeout"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Expiry int    `mapstructure:"expiry"`
}

type ConfigLoader struct {
	v        *viper.Viper
	cfg      *Config
	mu       sync.RWMutex
	onChange func(*Config)
	watcher  *fsnotify.Watcher
	stopCh   chan struct{}
}

func NewLoader(path string) (*ConfigLoader, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	cl := &ConfigLoader{
		v:      v,
		cfg:    &Config{},
		stopCh: make(chan struct{}),
	}

	if err := v.Unmarshal(cl.cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cl, nil
}

func (cl *ConfigLoader) Get() *Config {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return cl.cfg
}

func (cl *ConfigLoader) Reload() error {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	if err := cl.v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	newCfg := &Config{}
	if err := cl.v.Unmarshal(newCfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	cl.cfg = newCfg

	if cl.onChange != nil {
		cl.onChange(newCfg)
	}

	return nil
}

func (cl *ConfigLoader) Watch(onChange func(*Config)) error {
	cl.onChange = onChange

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	cl.watcher = watcher

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					if err := cl.Reload(); err != nil {
						fmt.Printf("config reload error: %v\n", err)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Printf("watcher error: %v\n", err)
			case <-cl.stopCh:
				return
			}
		}
	}()

	return watcher.Add(cl.v.ConfigFileUsed())
}

func (cl *ConfigLoader) Stop() {
	close(cl.stopCh)
	if cl.watcher != nil {
		cl.watcher.Close()
	}
}

func Load(path string) (*Config, error) {
	cl, err := NewLoader(path)
	if err != nil {
		return nil, err
	}
	return cl.Get(), nil
}

func LoadFromFile(file string) (*Config, error) {
	return Load(file)
}

func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if c.Database.Source == "" {
		return fmt.Errorf("database source is required")
	}
	if c.Redis.Addr == "" {
		return fmt.Errorf("redis addr is required")
	}
	return nil
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:    8080,
			Host:    "0.0.0.0",
			Timeout: 60,
			Mode:    "debug",
		},
		Database: DatabaseConfig{
			Driver:       "postgres",
			Source:       "host=localhost user=postgres password=postgres dbname=parking port=5432 sslmode=disable",
			MaxOpenConns: 100,
			MaxIdleConns: 10,
			ConnMaxLife:  3600,
		},
		Redis: RedisConfig{
			Addr:        "localhost:6379",
			Password:    "",
			DB:          0,
			PoolSize:    100,
			MinIdleConn: 10,
		},
		Log: LogConfig{
			Level:      "info",
			Format:     "json",
			OutputPath: "logs/app.log",
			MaxSize:    100,
			MaxBackups: 30,
			MaxAge:     7,
			Compress:   true,
		},
		Telemetry: TelemetryConfig{
			Enabled:    false,
			Endpoint:   "localhost:4317",
			SampleRate: 1.0,
		},
		Wechat: WechatConfig{},
		Alipay: AlipayConfig{},
	}
}

func MustLoad(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}
	return cfg
}
