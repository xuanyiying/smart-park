package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
)

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
	Subscribe(topic string, handler mqtt.MessageHandler) error
	SubscribeCommand(topic string, handler func(*Command)) error
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
}

type MQTTClient struct {
	client    mqtt.Client
	config    *Config
	connected bool
	mu        sync.RWMutex

	commandHandlers map[string]func(*Command)
	results         chan *CommandResult
}

func NewMQTTClient(cfg *Config) *MQTTClient {
	clientID := cfg.ClientID
	if clientID == "" {
		clientID = fmt.Sprintf("smart-park-vehicle-%s", uuid.New().String()[:8])
	}

	broker := fmt.Sprintf("tcp://%s:%d", cfg.Broker, cfg.Port)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetUsername(cfg.Username)
	opts.SetPassword(cfg.Password)
	opts.SetAutoReconnect(true)
	opts.SetCleanSession(true)

	client := mqtt.NewClient(opts)

	return &MQTTClient{
		client:          client,
		config:          cfg,
		results:         make(chan *CommandResult, 100),
		commandHandlers: make(map[string]func(*Command)),
	}
}

func (c *MQTTClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
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
		close(c.results)
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

	token := c.client.Publish(topic, 1, false, payload)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish command: %w", token.Error())
	}

	return nil
}

func (c *MQTTClient) Subscribe(topic string, handler mqtt.MessageHandler) error {
	if !c.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	token := c.client.Subscribe(topic, 1, handler)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe: %w", token.Error())
	}

	return nil
}

func (c *MQTTClient) SubscribeCommand(topic string, handler func(*Command)) error {
	if !c.IsConnected() {
		return fmt.Errorf("client not connected")
	}

	token := c.client.Subscribe(topic, 1, func(client mqtt.Client, msg mqtt.Message) {
		var cmd Command
		if err := json.Unmarshal(msg.Payload(), &cmd); err != nil {
			return
		}
		handler(&cmd)
	})

	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe: %w", token.Error())
	}

	c.commandHandlers[topic] = handler
	return nil
}

func (c *MQTTClient) Unsubscribe(topic string) error {
	token := c.client.Unsubscribe(topic)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to unsubscribe: %w", token.Error())
	}

	delete(c.commandHandlers, topic)
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

func (c *MockMQTTClient) Subscribe(topic string, handler mqtt.MessageHandler) error {
	return nil
}

func (c *MockMQTTClient) SubscribeCommand(topic string, handler func(*Command)) error {
	return nil
}

func (c *MockMQTTClient) Unsubscribe(topic string) error {
	return nil
}

func (c *MockMQTTClient) Results() <-chan *CommandResult {
	return c.results
}
