package mqtt

import (
	"context"
	"testing"
	"time"
)

func TestCommandStruct(t *testing.T) {
	cmd := &Command{
		CommandID: "test-123",
		DeviceID:  "device-456",
		Command:   CommandOpenGate,
		Params:    map[string]string{"timeout": "5"},
		Timestamp: time.Now().Unix(),
		Priority:  1,
	}

	if cmd.CommandID != "test-123" {
		t.Errorf("CommandID = %s, want test-123", cmd.CommandID)
	}

	if cmd.DeviceID != "device-456" {
		t.Errorf("DeviceID = %s, want device-456", cmd.DeviceID)
	}

	if cmd.Command != CommandOpenGate {
		t.Errorf("Command = %s, want %s", cmd.Command, CommandOpenGate)
	}

	if cmd.Params["timeout"] != "5" {
		t.Errorf("Params[timeout] = %s, want 5", cmd.Params["timeout"])
	}
}

func TestCommandResultStruct(t *testing.T) {
	result := &CommandResult{
		CommandID: "cmd-789",
		DeviceID:  "device-001",
		Status:    "delivered",
		Message:   "Command sent successfully",
		Timestamp: time.Now().Unix(),
	}

	if result.CommandID != "cmd-789" {
		t.Errorf("CommandID = %s, want cmd-789", result.CommandID)
	}

	if result.Status != "delivered" {
		t.Errorf("Status = %s, want delivered", result.Status)
	}
}

func TestMockMQTTClient_Connect(t *testing.T) {
	client := NewMockMQTTClient()

	if client.IsConnected() {
		t.Error("Client should not be connected initially")
	}

	err := client.Connect()
	if err != nil {
		t.Errorf("Connect() returned error: %v", err)
	}

	if !client.IsConnected() {
		t.Error("Client should be connected after Connect()")
	}
}

func TestMockMQTTClient_Disconnect(t *testing.T) {
	client := NewMockMQTTClient()

	err := client.Connect()
	if err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}

	err = client.Disconnect()
	if err != nil {
		t.Errorf("Disconnect() returned error: %v", err)
	}

	if client.IsConnected() {
		t.Error("Client should not be connected after Disconnect()")
	}
}

func TestMockMQTTClient_PublishCommand(t *testing.T) {
	client := NewMockMQTTClient()
	_ = client.Connect()

	cmd := &Command{
		DeviceID: "device-test",
		Command:  CommandOpenGate,
		Params:   map[string]string{},
	}

	err := client.PublishCommand(context.Background(), cmd)
	if err != nil {
		t.Errorf("PublishCommand() returned error: %v", err)
	}

	if cmd.CommandID == "" {
		t.Error("CommandID should be set by PublishCommand")
	}

	if cmd.Timestamp == 0 {
		t.Error("Timestamp should be set by PublishCommand")
	}
}

func TestMockMQTTClient_PublishCommandNotConnected(t *testing.T) {
	client := NewMockMQTTClient()

	cmd := &Command{
		DeviceID: "device-test",
		Command:  CommandOpenGate,
	}

	err := client.PublishCommand(context.Background(), cmd)
	if err == nil {
		t.Error("PublishCommand() should return error when not connected")
	}
}

func TestMockMQTTClient_ResultsChannel(t *testing.T) {
	client := NewMockMQTTClient()

	err := client.Connect()
	if err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}

	cmd := &Command{
		DeviceID: "device-result-test",
		Command:  CommandCloseGate,
	}

	err = client.PublishCommand(context.Background(), cmd)
	if err != nil {
		t.Fatalf("PublishCommand() failed: %v", err)
	}

	select {
	case result := <-client.Results():
		if result.CommandID != cmd.CommandID {
			t.Errorf("Result CommandID = %s, want %s", result.CommandID, cmd.CommandID)
		}
		if result.DeviceID != cmd.DeviceID {
			t.Errorf("Result DeviceID = %s, want %s", result.DeviceID, cmd.DeviceID)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for command result")
	}
}

func TestMockMQTTClient_Subscribe(t *testing.T) {
	client := NewMockMQTTClient()

	err := client.Subscribe("test/topic", func(cmd *Command) {})
	if err != nil {
		t.Errorf("Subscribe() returned error: %v", err)
	}
}

func TestMockMQTTClient_Unsubscribe(t *testing.T) {
	client := NewMockMQTTClient()

	err := client.Unsubscribe("test/topic")
	if err != nil {
		t.Errorf("Unsubscribe() returned error: %v", err)
	}
}

func TestMQTTClient_Connect(t *testing.T) {
	t.Skip("Skipping network dependent test")
	cfg := &Config{
		Broker:   "tcp://localhost:1883",
		Port:     1883,
		ClientID: "test-client",
		Username: "user",
		Password: "pass",
		Topics:   []string{"test/topic"},
	}

	client := NewMQTTClient(cfg)

	if client.IsConnected() {
		t.Error("Client should not be connected initially")
	}

	err := client.Connect()
	if err != nil {
		t.Errorf("Connect() returned error: %v", err)
	}

	if !client.IsConnected() {
		t.Error("Client should be connected after Connect()")
	}

	err = client.Disconnect()
	if err != nil {
		t.Errorf("Disconnect() returned error: %v", err)
	}
}

func TestMQTTClient_ClientID(t *testing.T) {
	t.Skip()
	cfg := &Config{
		Broker: "tcp://localhost:1883",
	}

	client := NewMQTTClient(cfg)
	if client.config.ClientID == "" {
		t.Error("ClientID should be auto-generated if not provided")
	}

	cfg2 := &Config{
		Broker:   "tcp://localhost:1883",
		ClientID: "my-custom-client",
	}

	client2 := NewMQTTClient(cfg2)
	if client2.config.ClientID != "my-custom-client" {
		t.Errorf("ClientID = %s, want my-custom-client", client2.config.ClientID)
	}
}
