package mqtt

import (
	"context"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-kratos/kratos/v2/log"
)

type Config struct {
	Broker   string
	ClientID string
	Username string
	Password string
	QoS      byte
}

type Client interface {
	Connect(ctx context.Context) error
	Disconnect()
	Publish(ctx context.Context, topic string, payload []byte) error
	Subscribe(ctx context.Context, topic string, handler mqtt.MessageHandler) error
	Unsubscribe(ctx context.Context, topic string) error
}

type MQTTClient struct {
	client mqtt.Client
	config *Config
	log    *log.Helper
	mu     sync.RWMutex
}

func NewClient(config *Config, logger log.Logger) *MQTTClient {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.Broker)
	opts.SetClientID(config.ClientID)
	opts.SetUsername(config.Username)
	opts.SetPassword(config.Password)
	opts.SetAutoReconnect(true)
	opts.SetCleanSession(true)

	client := mqtt.NewClient(opts)

	return &MQTTClient{
		client: client,
		config: config,
		log:    log.NewHelper(logger),
	}
}

func (c *MQTTClient) Connect(ctx context.Context) error {
	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	c.log.Info("MQTT client connected")
	return nil
}

func (c *MQTTClient) Disconnect() {
	c.client.Disconnect(250)
	c.log.Info("MQTT client disconnected")
}

func (c *MQTTClient) Publish(ctx context.Context, topic string, payload []byte) error {
	token := c.client.Publish(topic, c.config.QoS, false, payload)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	c.log.Infof("Published to topic %s", topic)
	return nil
}

func (c *MQTTClient) Subscribe(ctx context.Context, topic string, handler mqtt.MessageHandler) error {
	token := c.client.Subscribe(topic, c.config.QoS, handler)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	c.log.Infof("Subscribed to topic %s", topic)
	return nil
}

func (c *MQTTClient) Unsubscribe(ctx context.Context, topic string) error {
	token := c.client.Unsubscribe(topic)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	c.log.Infof("Unsubscribed from topic %s", topic)
	return nil
}
