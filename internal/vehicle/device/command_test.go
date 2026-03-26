package device_test

import (
	"context"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/xuanyiying/smart-park/internal/vehicle/device"
)

type mockMQTTClient struct {
	publishedTopics   []string
	publishedPayloads [][]byte
}

func (m *mockMQTTClient) Publish(ctx context.Context, topic string, payload []byte) error {
	m.publishedTopics = append(m.publishedTopics, topic)
	m.publishedPayloads = append(m.publishedPayloads, payload)
	return nil
}

func (m *mockMQTTClient) Subscribe(ctx context.Context, topic string, handler mqtt.MessageHandler) error {
	return nil
}

func (m *mockMQTTClient) Unsubscribe(ctx context.Context, topic string) error {
	return nil
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
		manager.HandleResponse(&device.CommandResponse{
			ID:        "test-command-id",
			Success:   true,
			Message:   "Command executed",
			Timestamp: time.Now().Unix(),
		})
	}()

	_, err := manager.SendCommand(ctx, "device-001", device.CommandDisplay, params)
	if err != nil {
		t.Errorf("SendCommand failed: %v", err)
	}

	if len(mockClient.publishedTopics) > 0 {
		if mockClient.publishedTopics[0] != "device/device-001/command" {
			t.Errorf("expected topic device/device-001/command, got %s", mockClient.publishedTopics[0])
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
