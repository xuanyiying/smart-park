package mqtt

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
)

// tokenWaitTimeout bounds how long any broker round-trip (connect, publish,
// subscribe) may block. Without it a hung broker pins the calling request thread
// forever.
const tokenWaitTimeout = 10 * time.Second

type CommandType string

const (
	CommandOpenGate     CommandType = "open_gate"
	CommandCloseGate    CommandType = "close_gate"
	CommandRestart      CommandType = "restart"
	CommandUpdateConfig CommandType = "update_config"
)

type Command struct {
	CommandID string            `json:"command_id"`
	DeviceID  string            `json:"device_id"`
	Command   CommandType       `json:"command"`
	Params    map[string]string `json:"params,omitempty"`
	Timestamp int64             `json:"timestamp"`
	Priority  int               `json:"priority"`
}

type CommandResult struct {
	CommandID string `json:"command_id"`
	DeviceID  string `json:"device_id"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

type Client interface {
	PublishCommand(ctx context.Context, cmd *Command) error
	Subscribe(topic string, handler func(*Command)) error
	Unsubscribe(topic string) error
	Connect() error
	Disconnect() error
	IsConnected() bool
}

type Config struct {
	Broker   string
	Port     int
	ClientID string
	Username string
	Password string
	Topics   []string
	// TLS enables TLS transport (or is implied by a "tls://" broker scheme).
	TLS bool
	// TLSSkipVerify disables certificate verification (development only).
	TLSSkipVerify bool
}

type MQTTClient struct {
	client    mqtt.Client
	config    *Config
	connected bool
	mu        sync.RWMutex

	// subscriptions stores the wrapped handlers so they can be replayed after a
	// reconnect: with a persistent session paho restores broker-side subscriptions,
	// but resubscribing explicitly keeps delivery correct across all failure modes.
	subscriptions map[string]func(*Command)
	results       chan *CommandResult
}

func NewMQTTClient(cfg *Config) *MQTTClient {
	clientID := cfg.ClientID
	if clientID == "" {
		clientID = fmt.Sprintf("smart-park-vehicle-%s", uuid.New().String()[:8])
	}

	scheme := "tcp"
	if cfg.TLS {
		scheme = "ssl"
	}
	broker := fmt.Sprintf("%s://%s:%d", scheme, cfg.Broker, cfg.Port)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetUsername(cfg.Username)
	opts.SetPassword(cfg.Password)
	opts.SetAutoReconnect(true)
	// 持久会话（CleanSession=false）：设备离线期间 QoS1 指令由 broker 暂存，
	// 恢复连接后投递，而不是静默丢弃。生产部署应固定 ClientID 以复用会话。
	opts.SetCleanSession(false)
	opts.SetConnectTimeout(tokenWaitTimeout)

	// 遗嘱消息：进程异常退出时 broker 会在状态主题上发布 offline，供监控告警。
	willTopic := fmt.Sprintf("smart-park/service/%s/status", clientID)
	opts.SetWill(willTopic, `{"status":"offline"}`, 1, true)

	if cfg.TLS {
		opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: cfg.TLSSkipVerify}) //nolint:gosec // TLSSkipVerify is an explicit operator choice
	}

	c := &MQTTClient{
		client:        mqtt.NewClient(opts),
		config:        cfg,
		results:       make(chan *CommandResult, 100),
		subscriptions: make(map[string]func(*Command)),
	}

	// 重连后重建订阅关系，否则自动重连只是一条"假活"的连接。
	opts.OnConnect = func(client mqtt.Client) {
		c.mu.Lock()
		subs := make(map[string]func(*Command), len(c.subscriptions))
		for topic, handler := range c.subscriptions {
			subs[topic] = handler
		}
		c.mu.Unlock()

		for topic, handler := range subs {
			if t := client.Subscribe(topic, 1, c.wrapHandler(handler)); t.WaitTimeout(tokenWaitTimeout) && t.Error() != nil {
				continue
			}
		}
	}

	return c
}

// wrapHandler decodes the command payload once and funnels it to the business handler.
func (c *MQTTClient) wrapHandler(handler func(*Command)) func(mqtt.Client, mqtt.Message) {
	return func(client mqtt.Client, msg mqtt.Message) {
		var cmd Command
		if err := json.Unmarshal(msg.Payload(), &cmd); err != nil {
			return
		}
		handler(&cmd)
	}
}

func (c *MQTTClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if token := c.client.Connect(); !token.WaitTimeout(tokenWaitTimeout) || token.Error() != nil {
		if token.Error() != nil {
			return token.Error()
		}
		return fmt.Errorf("connect timed out after %v", tokenWaitTimeout)
	}

	c.connected = true
	return nil
}

func (c *MQTTClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		c.client.Disconnect(250)
		c.connected = false
		// 不关闭 results channel：并发的 PublishCommand goroutine 向其写入时
		// 会导致 panic。让 GC 回收即可。
	}
	return nil
}

func (c *MQTTClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

func (c *MQTTClient) PublishCommand(ctx context.Context, cmd *Command) error {
	if !c.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	if cmd.CommandID == "" {
		cmd.CommandID = uuid.New().String()
	}
	if cmd.Timestamp == 0 {
		cmd.Timestamp = time.Now().Unix()
	}

	payload, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	topic := fmt.Sprintf("smart-park/device/%s/command", cmd.DeviceID)

	// 尊重调用方 context：请求已取消时不再等 broker。
	if err := ctx.Err(); err != nil {
		return err
	}

	token := c.client.Publish(topic, 1, false, payload)
	if !token.WaitTimeout(tokenWaitTimeout) {
		return fmt.Errorf("publish timed out after %v", tokenWaitTimeout)
	}
	if token.Error() != nil {
		return fmt.Errorf("failed to publish command: %w", token.Error())
	}

	return nil
}

func (c *MQTTClient) Subscribe(topic string, handler func(*Command)) error {
	if !c.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	token := c.client.Subscribe(topic, 1, c.wrapHandler(handler))
	if !token.WaitTimeout(tokenWaitTimeout) {
		return fmt.Errorf("subscribe timed out after %v", tokenWaitTimeout)
	}
	if token.Error() != nil {
		return fmt.Errorf("failed to subscribe: %w", token.Error())
	}

	c.mu.Lock()
	c.subscriptions[topic] = handler
	c.mu.Unlock()
	return nil
}

func (c *MQTTClient) Unsubscribe(topic string) error {
	token := c.client.Unsubscribe(topic)
	if !token.WaitTimeout(tokenWaitTimeout) {
		return fmt.Errorf("unsubscribe timed out after %v", tokenWaitTimeout)
	}
	if token.Error() != nil {
		return fmt.Errorf("failed to unsubscribe: %w", token.Error())
	}

	c.mu.Lock()
	delete(c.subscriptions, topic)
	c.mu.Unlock()
	return nil
}

func (c *MQTTClient) Results() <-chan *CommandResult {
	return c.results
}

type MockMQTTClient struct {
	connected bool
	mu        sync.RWMutex
	results   chan *CommandResult
}

func NewMockMQTTClient() *MockMQTTClient {
	return &MockMQTTClient{
		results: make(chan *CommandResult, 100),
	}
}

func (c *MockMQTTClient) Connect() error {
	c.mu.Lock()
	c.connected = true
	c.mu.Unlock()
	return nil
}

func (c *MockMQTTClient) Disconnect() error {
	c.mu.Lock()
	c.connected = false
	c.mu.Unlock()
	close(c.results)
	return nil
}

func (c *MockMQTTClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

func (c *MockMQTTClient) PublishCommand(ctx context.Context, cmd *Command) error {
	if !c.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	if cmd.CommandID == "" {
		cmd.CommandID = uuid.New().String()
	}
	if cmd.Timestamp == 0 {
		cmd.Timestamp = time.Now().Unix()
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		c.results <- &CommandResult{
			CommandID: cmd.CommandID,
			DeviceID:  cmd.DeviceID,
			Status:    "delivered",
			Timestamp: time.Now().Unix(),
		}
	}()

	return nil
}

func (c *MockMQTTClient) Subscribe(topic string, handler func(*Command)) error {
	return nil
}

func (c *MockMQTTClient) Unsubscribe(topic string) error {
	return nil
}

func (c *MockMQTTClient) Results() <-chan *CommandResult {
	return c.results
}
