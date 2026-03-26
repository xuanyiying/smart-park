package mqtt_test

import (
	"context"
	"testing"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/xuanyiying/smart-park/pkg/mqtt"
)

func TestMQTTClientConnect(t *testing.T) {
	config := &mqtt.Config{
		Broker:   "tcp://localhost:1883",
		ClientID: "test-client",
		QoS:      1,
	}

	client := mqtt.NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	defer client.Disconnect()
}

func TestMQTTPublishSubscribe(t *testing.T) {
	t.Skip("需要真实的 MQTT Broker 才能运行")

	config := &mqtt.Config{
		Broker:   "tcp://localhost:1883",
		ClientID: "test-client",
		QoS:      1,
	}

	client := mqtt.NewClient(config, nil)

	ctx := context.Background()

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer client.Disconnect()

	received := make(chan []byte, 1)

	handler := func(client pahomqtt.Client, msg pahomqtt.Message) {
		received <- msg.Payload()
	}

	err = client.Subscribe(ctx, "test/topic", handler)
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	err = client.Publish(ctx, "test/topic", []byte("test message"))
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	select {
	case msg := <-received:
		if string(msg) != "test message" {
			t.Errorf("expected 'test message', got %s", string(msg))
		}
	case <-time.After(5 * time.Second):
		t.Error("timeout waiting for message")
	}
}
