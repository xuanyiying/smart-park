package device_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/xuanyiying/smart-park/internal/vehicle/device"
)

type mockMQTTClient struct {
	mu                sync.Mutex
	publishedTopics   []string
	publishedPayloads [][]byte
	handler           mqtt.MessageHandler
}

func (m *mockMQTTClient) Publish(ctx context.Context, topic string, payload []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishedTopics = append(m.publishedTopics, topic)
	m.publishedPayloads = append(m.publishedPayloads, payload)
	return nil
}

func (m *mockMQTTClient) Subscribe(ctx context.Context, topic string, handler mqtt.MessageHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handler = handler
	return nil
}

func (m *mockMQTTClient) Unsubscribe(ctx context.Context, topic string) error {
	return nil
}

func (m *mockMQTTClient) getPublishedPayload() ([]byte, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.publishedPayloads) == 0 {
		return nil, false
	}
	return m.publishedPayloads[0], true
}

func TestSendCommand(t *testing.T) {
	mockClient := &mockMQTTClient{}
	manager := device.NewCommandManager(mockClient)

	ctx := context.Background()
	params := map[string]interface{}{
		"message": "Welcome",
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		payload, ok := mockClient.getPublishedPayload()
		if !ok {
			t.Error("no payload published")
			return
		}
		var cmd device.Command
		if err := json.Unmarshal(payload, &cmd); err != nil {
			t.Errorf("failed to unmarshal command: %v", err)
			return
		}
		manager.HandleResponse(&device.CommandResponse{
			ID:        cmd.ID,
			Success:   true,
			Message:   "Command executed",
			Timestamp: time.Now().Unix(),
		})
	}()

	_, err := manager.SendCommand(ctx, "device-001", device.CommandDisplay, params)
	if err != nil {
		t.Errorf("SendCommand failed: %v", err)
	}

	mockClient.mu.Lock()
	topics := mockClient.publishedTopics
	mockClient.mu.Unlock()

	if len(topics) > 0 {
		if topics[0] != "device/device-001/command" {
			t.Errorf("expected topic device/device-001/command, got %s", topics[0])
		}
	}
}

func TestCommandTimeout(t *testing.T) {
	mockClient := &mockMQTTClient{}
	manager := device.NewCommandManager(mockClient)

	managerCtx := context.Background()

	_, err := manager.SendCommand(managerCtx, "device-001", device.CommandOpenGate, nil)
	if err == nil {
		t.Error("expected timeout error")
	}
}
