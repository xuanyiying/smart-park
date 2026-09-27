package config

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

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
	// EntryExit tunes the vehicle entry/exit pipeline (distributed lock TTL, plate
	// recognition confidence threshold, device online threshold).
	EntryExit EntryExitConfig `mapstructure:"entry_exit"`
	// Holidays lists statutory holiday dates ("2006-01-02") that activate the
	// billing engine's "holiday" condition. Empty means no day is a holiday.
	Holidays []string `mapstructure:"holidays"`
	Billing   *BillingConfig  `mapstructure:"billing"`
	Vehicle   *VehicleConfig  `mapstructure:"vehicle"`
	Payment   *PaymentConfig  `mapstructure:"payment"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	Routes    []RouteConfig   `mapstructure:"routes"`
	Etcd      EtcdConfig      `mapstructure:"etcd"`
	Otel      OtelConfig      `mapstructure:"otel"`
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
	// PublicKeyPath points to the WeChat Pay *platform certificate* (PEM, CERTIFICATE block).
	// It is required to verify APIv3 callback signatures.
	PublicKeyPath string `mapstructure:"public_key_path"`
	// APIv3Key is the 32-char APIv3 secret used to AES-256-GCM decrypt callback resources.
	APIv3Key  string `mapstructure:"api_v3_key"`
	NotifyURL string `mapstructure:"notify_url"`
}

type AlipayConfig struct {
	AppID        string `mapstructure:"app_id"`
	PrivateKey   string `mapstructure:"private_key"`
	PublicKey    string `mapstructure:"public_key"`
	NotifyURL    string `mapstructure:"notify_url"`
	IsProduction bool   `mapstructure:"is_production"`
	// SignType selects the callback signature algorithm: "RSA" (SHA1) or "RSA2" (SHA256).
	// Defaults to RSA2 when empty.
	SignType string `mapstructure:"sign_type"`
}

type MQTTConfig struct {
	Broker   string `mapstructure:"broker"`
	Port     int    `mapstructure:"port"`
	ClientID string `mapstructure:"client_id"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	// TLS enables TLS transport to the broker; TLSSkipVerify disables certificate
	// verification (development only).
	TLS           bool `mapstructure:"tls"`
	TLSSkipVerify bool `mapstructure:"tls_skip_verify"`
}

// EntryExitConfig holds tunables for the vehicle entry/exit pipeline. Durations are
// expressed as Go duration strings ("10s", "1m") and zero values fall back to the
// defaults defined in the business layer.
type EntryExitConfig struct {
	LockTTL               string  `mapstructure:"lock_ttl"`
	DeviceOnlineThreshold string  `mapstructure:"device_online_threshold"`
	MinConfidence         float64 `mapstructure:"min_confidence"`
	// Messages overrides driver-facing display messages by key (e.g. "welcome",
	// "lot_full", "blacklisted"). Unset keys fall back to the built-in defaults.
	Messages map[string]string `mapstructure:"messages"`
	// SeedLotID opts into provisioning demo lanes/devices for a specific parking lot.
	// Empty disables seeding entirely, which is the production default.
	SeedLotID string `mapstructure:"seed_lot_id"`
}

type BillingConfig struct {
	Endpoint string `mapstructure:"endpoint"`
	Timeout  int    `mapstructure:"timeout"`
	// SeedLotID opts into provisioning starting billing rules for a specific parking lot.
	// Empty disables seeding entirely, which is the production default.
	SeedLotID string `mapstructure:"seed_lot_id"`
}

type VehicleConfig struct {
	Endpoint string `mapstructure:"endpoint"`
	Timeout  int    `mapstructure:"timeout"`
}

type PaymentConfig struct {
	Endpoint string `mapstructure:"endpoint"`
	Timeout  int    `mapstructure:"timeout"`
	// SweepInterval controls how often pending orders are swept: expired ones are closed,
	// and gateway-confirmed payments that lost their callback get settled. Zero or
	// negative falls back to the service default of one minute.
	SweepInterval time.Duration `mapstructure:"sweep_interval"`
}

type JWTConfig struct {
	Secret         string        `mapstructure:"secret"`
	PublicKeyPath  string        `mapstructure:"public_key_path"`
	PrivateKeyPath string        `mapstructure:"private_key_path"`
	Expiry         int           `mapstructure:"expiry"`
	TokenDuration  time.Duration `mapstructure:"token_duration"`
	// SkipPaths lists routes that bypass authentication (health probes, login, etc.).
	// Only add paths that are genuinely meant to be public.
	SkipPaths []string `mapstructure:"skip_paths"`
}

type RouteConfig struct {
	Path   string `mapstructure:"path"`
	Target string `mapstructure:"target"`
}

type EtcdConfig struct {
	Endpoints []string `mapstructure:"endpoints"`
	Timeout   string   `mapstructure:"timeout"`
}

type OtelConfig struct {
	Endpoint    string `mapstructure:"endpoint"`
	ServiceName string `mapstructure:"serviceName"`
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

	cl := &ConfigLoader{
		v:      v,
		cfg:    &Config{},
		stopCh: make(chan struct{}),
	}

	cl.bindEnvVars()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
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

var envBindings = map[string]string{
	"database.source":         "SP_DATABASE_SOURCE",
	"redis.password":          "SP_REDIS_PASSWORD",
	"redis.addr":              "SP_REDIS_ADDR",
	"mq.redis.password":       "SP_MQ_REDIS_PASSWORD",
	"mq.nats.password":        "SP_MQ_NATS_PASSWORD",
	"mq.rocketmq.accessKey":   "SP_MQ_ROCKETMQ_ACCESS_KEY",
	"mq.rocketmq.secretKey":   "SP_MQ_ROCKETMQ_SECRET_KEY",
	"wechat.app_id":           "SP_WECHAT_APP_ID",
	"wechat.mch_id":           "SP_WECHAT_MCH_ID",
	"wechat.api_key":          "SP_WECHAT_API_KEY",
	"wechat.cert_serial_no":   "SP_WECHAT_CERT_SERIAL_NO",
	"wechat.private_key_path": "SP_WECHAT_PRIVATE_KEY_PATH",
	"wechat.public_key_path":  "SP_WECHAT_PUBLIC_KEY_PATH",
	"wechat.notify_url":       "SP_WECHAT_NOTIFY_URL",
	"alipay.app_id":           "SP_ALIPAY_APP_ID",
	"alipay.private_key":      "SP_ALIPAY_PRIVATE_KEY",
	"alipay.public_key":       "SP_ALIPAY_PUBLIC_KEY",
	"alipay.notify_url":       "SP_ALIPAY_NOTIFY_URL",
	"mqtt.password":           "SP_MQTT_PASSWORD",
	"jwt.secret":              "SP_JWT_SECRET",
	"jwt.public_key_path":     "SP_JWT_PUBLIC_KEY_PATH",
	"jwt.private_key_path":    "SP_JWT_PRIVATE_KEY_PATH",
}

var requiredProductionEnvVars = []string{
	"SP_DATABASE_SOURCE",
	"SP_REDIS_PASSWORD",
	"SP_WECHAT_API_KEY",
	"SP_ALIPAY_PRIVATE_KEY",
	"SP_JWT_SECRET",
}

func (cl *ConfigLoader) bindEnvVars() {
	for key, env := range envBindings {
		cl.v.BindEnv(key, env)
	}
}

func (c *Config) ValidateForProduction() error {
	var missing []string
	for _, env := range requiredProductionEnvVars {
		if os.Getenv(env) == "" {
			missing = append(missing, env)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("required production environment variables not set: %s", strings.Join(missing, ", "))
	}
	return nil
}

func decryptValue(value string) (string, error) {
	if !strings.HasPrefix(value, "ENC(") || !strings.HasSuffix(value, ")") {
		return value, nil
	}
	encKey := os.Getenv("SP_CONFIG_ENCRYPTION_KEY")
	if encKey == "" {
		return "", fmt.Errorf("SP_CONFIG_ENCRYPTION_KEY environment variable not set")
	}

	key, err := hex.DecodeString(encKey)
	if err != nil {
		return "", fmt.Errorf("invalid encryption key format: %w", err)
	}
	if len(key) != 32 {
		return "", fmt.Errorf("encryption key must be 32 bytes (64 hex chars), got %d bytes", len(key))
	}

	encoded := value[4 : len(value)-1]
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 ciphertext: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBody := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertextBody, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

func EncryptValue(plaintext string, hexKey string) (string, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return "", fmt.Errorf("invalid encryption key format: %w", err)
	}
	if len(key) != 32 {
		return "", fmt.Errorf("encryption key must be 32 bytes (64 hex chars), got %d bytes", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	for i := range nonce {
		nonce[i] = byte(i)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return "ENC(" + encoded + ")", nil
}

func (c *Config) DecryptConfig() error {
	return decryptConfigFields(reflect.ValueOf(c).Elem())
}

func decryptConfigFields(v reflect.Value) error {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := v.Type().Field(i)

		if !field.CanInterface() {
			continue
		}

		switch field.Kind() {
		case reflect.String:
			val := field.String()
			if strings.HasPrefix(val, "ENC(") && strings.HasSuffix(val, ")") {
				decrypted, err := decryptValue(val)
				if err != nil {
					return fmt.Errorf("failed to decrypt field %s: %w", fieldType.Name, err)
				}
				field.SetString(decrypted)
			}
		case reflect.Struct:
			if err := decryptConfigFields(field); err != nil {
				return err
			}
		case reflect.Ptr:
			if !field.IsNil() {
				if field.Elem().Kind() == reflect.Struct {
					if err := decryptConfigFields(field); err != nil {
						return err
					}
				}
			}
		case reflect.Slice:
			if fieldType.Type.Elem().Kind() == reflect.Struct {
				for j := 0; j < field.Len(); j++ {
					if err := decryptConfigFields(field.Index(j)); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func maskValue(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}

func maskConfigFields(src, dst reflect.Value) {
	if src.Kind() == reflect.Ptr {
		if src.IsNil() {
			return
		}
		src = src.Elem()
		dst = dst.Elem()
	}

	for i := 0; i < src.NumField(); i++ {
		srcField := src.Field(i)
		dstField := dst.Field(i)
		fieldType := src.Type().Field(i)

		if !srcField.CanInterface() || !dstField.CanSet() {
			continue
		}

		switch srcField.Kind() {
		case reflect.String:
			if isSensitiveField(fieldType.Name) && srcField.String() != "" {
				dstField.SetString(maskValue(srcField.String()))
			} else {
				dstField.SetString(srcField.String())
			}
		case reflect.Struct:
			maskConfigFields(srcField, dstField)
		case reflect.Ptr:
			if !srcField.IsNil() && srcField.Elem().Kind() == reflect.Struct {
				dstField.Set(reflect.New(srcField.Elem().Type()))
				maskConfigFields(srcField, dstField)
			}
		case reflect.Slice:
			if fieldType.Type.Elem().Kind() == reflect.Struct {
				dstField.Set(reflect.MakeSlice(fieldType.Type, srcField.Len(), srcField.Cap()))
				for j := 0; j < srcField.Len(); j++ {
					maskConfigFields(srcField.Index(j), dstField.Index(j))
				}
			} else {
				dstField.Set(srcField)
			}
		default:
			dstField.Set(srcField)
		}
	}
}

var sensitiveFieldNames = map[string]bool{
	"Password":      true,
	"APIKey":        true,
	"PrivateKey":    true,
	"PublicKey":     true,
	"Secret":        true,
	"SecretKey":     true,
	"AccessKey":     true,
	"CertSerialNo":  true,
	"Source":        true,
}

func isSensitiveField(name string) bool {
	return sensitiveFieldNames[name]
}

func (c *Config) Masked() *Config {
	masked := &Config{}
	maskConfigFields(reflect.ValueOf(c).Elem(), reflect.ValueOf(masked).Elem())
	return masked
}

func MustLoad(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}
	return cfg
}
