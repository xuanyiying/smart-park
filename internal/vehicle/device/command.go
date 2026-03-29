package device

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
	CommandOpenGate  CommandType = "open_gate"
	CommandCloseGate CommandType = "close_gate"
	CommandDisplay   CommandType = "display"
	CommandVoice     CommandType = "voice"
)

type Command struct {
	ID        string                 `json:"id"`
	Type      CommandType            `json:"type"`
	DeviceID  string                 `json:"device_id"`
	Timestamp int64                  `json:"timestamp"`
	Params    map[string]interface{} `json:"params"`
}

type CommandResponse struct {
	ID        string `json:"id"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

type MQTTClient interface {
	Publish(ctx context.Context, topic string, payload []byte) error
	Subscribe(ctx context.Context, topic string, handler mqtt.MessageHandler) error
	Unsubscribe(ctx context.Context, topic string) error
}

type CommandManager struct {
	mqttClient MQTTClient
	pending    map[string]chan *CommandResponse
	mu         sync.RWMutex
	timeout    time.Duration
}

func NewCommandManager(mqttClient MQTTClient) *CommandManager {
	return &CommandManager{
		mqttClient: mqttClient,
		pending:    make(map[string]chan *CommandResponse),
		timeout:    10 * time.Second,
	}
}

func (m *CommandManager) SendCommand(ctx context.Context, deviceID string, cmdType CommandType, params map[string]interface{}) (*CommandResponse, error) {
	cmd := &Command{
		ID:        uuid.New().String(),
		Type:      cmdType,
		DeviceID:  deviceID,
		Timestamp: time.Now().Unix(),
		Params:    params,
	}

	topic := fmt.Sprintf("device/%s/command", deviceID)

	payload, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	respChan := make(chan *CommandResponse, 1)
	m.mu.Lock()
	m.pending[cmd.ID] = respChan
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		delete(m.pending, cmd.ID)
		m.mu.Unlock()
	}()

	if err := m.mqttClient.Publish(ctx, topic, payload); err != nil {
		return nil, err
	}

	select {
	case resp := <-respChan:
		return resp, nil
	case <-time.After(m.timeout):
		return nil, fmt.Errorf("command timeout")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *CommandManager) HandleResponse(resp *CommandResponse) {
	m.mu.RLock()
	ch, ok := m.pending[resp.ID]
	m.mu.RUnlock()
	if ok {
		ch <- resp
	}
}

func (m *CommandManager) SubscribeToResponses(ctx context.Context, deviceID string) error {
	topic := fmt.Sprintf("device/%s/response", deviceID)

	handler := func(client mqtt.Client, msg mqtt.Message) {
		var resp CommandResponse
		if err := json.Unmarshal(msg.Payload(), &resp); err != nil {
			return
		}
		m.HandleResponse(&resp)
	}

	return m.mqttClient.Subscribe(ctx, topic, handler)
}
