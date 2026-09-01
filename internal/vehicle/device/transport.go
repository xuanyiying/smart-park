package device

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xuanyiying/smart-park/internal/vehicle/data/mqtt"
)

// ErrNoTransport is returned when an adapter is asked to act on a device but no command
// transport was configured. Reporting this is the whole point: the previous adapters
// returned nil without doing anything, so a gate "opened" successfully in the logs while
// nothing happened at the barrier.
var ErrNoTransport = errors.New("device: no command transport configured")

// CommandTransport delivers a command to a physical device and reports whether the
// gateway accepted it.
type CommandTransport interface {
	// Send publishes command with params to deviceID. A nil error means the command was
	// handed to the device gateway, not that the device executed it.
	Send(ctx context.Context, deviceID string, command string, params map[string]string) error
}

// StatusProvider reports the last known state of a device. Implementations read it from
// the device registry (driven by heartbeats) rather than querying the device on demand.
type StatusProvider interface {
	Status(ctx context.Context, deviceID string) (map[string]interface{}, error)
}

// MQTTCommandTransport sends device commands over the MQTT command topic.
type MQTTCommandTransport struct {
	client mqtt.Client
}

// NewMQTTCommandTransport wraps the vehicle service MQTT client as a command transport.
func NewMQTTCommandTransport(client mqtt.Client) *MQTTCommandTransport {
	return &MQTTCommandTransport{client: client}
}

func (t *MQTTCommandTransport) Send(ctx context.Context, deviceID string, command string, params map[string]string) error {
	if t == nil || t.client == nil {
		return ErrNoTransport
	}
	if deviceID == "" {
		return fmt.Errorf("device: cannot send command %q without a device id", command)
	}

	return t.client.PublishCommand(ctx, &mqtt.Command{
		DeviceID:  deviceID,
		Command:   mqtt.CommandType(command),
		Params:    params,
		Timestamp: time.Now().Unix(),
	})
}

// DeviceStateReader is the minimal contract the device registry must satisfy for
// adapters to report real device state. Declaring it here (instead of depending on the
// business layer) keeps the device package free of import cycles.
type DeviceStateReader interface {
	// DeviceState returns the recorded status and last heartbeat time of a device.
	DeviceState(ctx context.Context, deviceID string) (status string, lastHeartbeat *time.Time, err error)
}

// RegistryStatusProvider derives device state from heartbeats stored in the registry.
//
// A device is only reported online when its last heartbeat is within the threshold. This
// replaces the previous behaviour of unconditionally answering "online", which meant an
// offline barrier looked healthy forever.
type RegistryStatusProvider struct {
	reader     DeviceStateReader
	threshold  time.Duration
	now        func() time.Time
}

// NewRegistryStatusProvider builds a status provider with the given offline threshold.
// now is injectable so tests can simulate heartbeat ageing without sleeping.
func NewRegistryStatusProvider(reader DeviceStateReader, onlineThreshold time.Duration, now func() time.Time) *RegistryStatusProvider {
	if now == nil {
		now = time.Now
	}
	return &RegistryStatusProvider{reader: reader, threshold: onlineThreshold, now: now}
}

func (p *RegistryStatusProvider) Status(ctx context.Context, deviceID string) (map[string]interface{}, error) {
	if p == nil || p.reader == nil {
		return nil, fmt.Errorf("device: no device registry configured, cannot read state of %s", deviceID)
	}

	status, lastHeartbeat, err := p.reader.DeviceState(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	online := false
	var lastHeartbeatText string
	if lastHeartbeat != nil {
		lastHeartbeatText = lastHeartbeat.Format(time.RFC3339)
		online = p.now().Sub(*lastHeartbeat) < p.threshold
	}

	if status == "" {
		status = "unknown"
	}

	return map[string]interface{}{
		"device_id":      deviceID,
		"status":         status,
		"online":         online,
		"last_heartbeat": lastHeartbeatText,
	}, nil
}

// StaticStatusProvider answers from a fixed map, useful for tests and for deployments
// that manage device state outside this service.
type StaticStatusProvider struct {
	statuses map[string]map[string]interface{}
}

func NewStaticStatusProvider(statuses map[string]map[string]interface{}) *StaticStatusProvider {
	return &StaticStatusProvider{statuses: statuses}
}

func (p *StaticStatusProvider) Status(ctx context.Context, deviceID string) (map[string]interface{}, error) {
	if p == nil || p.statuses == nil {
		return nil, fmt.Errorf("device: no status available for %s", deviceID)
	}
	status, ok := p.statuses[deviceID]
	if !ok {
		return nil, fmt.Errorf("device: unknown device %s", deviceID)
	}
	return status, nil
}
