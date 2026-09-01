// Package device provides device management functionality.
//
// Adapters translate vendor-neutral operations (open gate, read status) into commands on
// the device gateway. Every adapter ultimately publishes through a CommandTransport, so a
// method returning nil means the command really was handed to the device.
package device

import (
	"context"
	"errors"
	"fmt"
)

// DeviceAdapter defines the interface for device adapters.
type DeviceAdapter interface {
	// OpenGate opens the gate
	OpenGate(ctx context.Context, deviceID string) error

	// CloseGate closes the gate
	CloseGate(ctx context.Context, deviceID string) error

	// GetDeviceStatus gets the device status
	GetDeviceStatus(ctx context.Context, deviceID string) (map[string]interface{}, error)

	// SendCommand sends a custom command to the device
	SendCommand(ctx context.Context, deviceID string, command string, params map[string]interface{}) (map[string]interface{}, error)

	// GetManufacturer returns the manufacturer name
	GetManufacturer() string

	// GetModel returns the device model
	GetModel() string
}

// AdapterFactory creates device adapters based on manufacturer and model.
type AdapterFactory struct {
	adapters     map[string]DeviceAdapter
	transport    CommandTransport
	statusReader StatusProvider
}

// NewAdapterFactory creates a factory that builds adapters wired to the given transport.
//
// transport may be nil, in which case adapters still resolve but return ErrNoTransport
// when asked to act on a device. That is deliberate: silently succeeding would make the
// system report gates as open when no command was ever sent.
func NewAdapterFactory(transport CommandTransport, statusReader StatusProvider) *AdapterFactory {
	f := &AdapterFactory{
		adapters:     make(map[string]DeviceAdapter),
		transport:    transport,
		statusReader: statusReader,
	}

	for _, manufacturer := range []string{"Hikvision", "Dahua", "Jieshun", "Ketuo", "Lanka"} {
		adapter := newVendorAdapter(manufacturer, "", f.transport, f.statusReader)
		f.Register(manufacturer, "*", adapter)
	}

	return f
}

// Register registers a device adapter for a specific manufacturer and model.
func (f *AdapterFactory) Register(manufacturer, model string, adapter DeviceAdapter) {
	if model == "" {
		model = "*"
	}
	f.adapters[manufacturer+":"+model] = adapter
}

// GetAdapter returns the device adapter for the specified manufacturer and model.
func (f *AdapterFactory) GetAdapter(manufacturer, model string) (DeviceAdapter, error) {
	key := manufacturer + ":" + model
	if adapter, ok := f.adapters[key]; ok {
		return adapter, nil
	}

	key = manufacturer + ":*"
	if adapter, ok := f.adapters[key]; ok {
		return adapter, nil
	}

	return nil, fmt.Errorf("no adapter found for manufacturer %q model %q", manufacturer, model)
}

// VendorAdapter is the shared implementation behind every manufacturer adapter.
//
// Vendors differ in the command vocabulary and the parameters their controllers expect;
// those differences are expressed as data (see vendorProfile) and a vendor may still
// override any method when it genuinely needs a different protocol.
type VendorAdapter struct {
	manufacturer string
	model        string
	transport    CommandTransport
	statusReader StatusProvider
	profile      vendorProfile
}

// vendorProfile captures how a vendor names its commands.
type vendorProfile struct {
	// Command names as the vendor controller expects them.
	OpenGate  string
	CloseGate string
	Status    string
}

// defaultProfile matches the neutral command vocabulary used on the MQTT bus.
func defaultProfile() vendorProfile {
	return vendorProfile{
		OpenGate:  "open_gate",
		CloseGate: "close_gate",
		Status:    "get_status",
	}
}

func newVendorAdapter(manufacturer, model string, transport CommandTransport, statusReader StatusProvider) *VendorAdapter {
	return &VendorAdapter{
		manufacturer: manufacturer,
		model:        model,
		transport:    transport,
		statusReader: statusReader,
		profile:      defaultProfile(),
	}
}

// OpenGate instructs the barrier controller to raise the barrier.
func (a *VendorAdapter) OpenGate(ctx context.Context, deviceID string) error {
	return a.dispatch(ctx, deviceID, a.profile.OpenGate, nil)
}

// CloseGate instructs the barrier controller to lower the barrier.
func (a *VendorAdapter) CloseGate(ctx context.Context, deviceID string) error {
	return a.dispatch(ctx, deviceID, a.profile.CloseGate, nil)
}

// GetDeviceStatus reports the last known state of the device.
//
// It reads from the status provider (fed by device heartbeats) instead of returning a
// hard-coded "online", which is what made the health dashboard claim every device was up
// even when it had been offline for days.
func (a *VendorAdapter) GetDeviceStatus(ctx context.Context, deviceID string) (map[string]interface{}, error) {
	if deviceID == "" {
		return nil, errors.New("device: device id is required")
	}
	if a.statusReader == nil {
		return nil, fmt.Errorf("device: no status provider configured, cannot read state of %s", deviceID)
	}

	status, err := a.statusReader.Status(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	out := make(map[string]interface{}, len(status)+2)
	for k, v := range status {
		out[k] = v
	}
	out["manufacturer"] = a.manufacturer
	out["model"] = a.model
	return out, nil
}

// SendCommand forwards an arbitrary command to the device.
func (a *VendorAdapter) SendCommand(ctx context.Context, deviceID string, command string, params map[string]interface{}) (map[string]interface{}, error) {
	if command == "" {
		return nil, errors.New("device: command is required")
	}

	flat := make(map[string]string, len(params))
	for k, v := range params {
		flat[k] = fmt.Sprintf("%v", v)
	}

	if err := a.dispatch(ctx, deviceID, command, flat); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"result":       "accepted",
		"command":      command,
		"device_id":    deviceID,
		"manufacturer": a.manufacturer,
	}, nil
}

func (a *VendorAdapter) dispatch(ctx context.Context, deviceID, command string, params map[string]string) error {
	if deviceID == "" {
		return fmt.Errorf("device: cannot send command %q without a device id", command)
	}
	if a.transport == nil {
		return fmt.Errorf("%w: cannot send command %q to device %s", ErrNoTransport, command, deviceID)
	}
	return a.transport.Send(ctx, deviceID, command, params)
}

// GetManufacturer returns the manufacturer name.
func (a *VendorAdapter) GetManufacturer() string {
	return a.manufacturer
}

// GetModel returns the device model.
func (a *VendorAdapter) GetModel() string {
	return a.model
}

// DefaultAdapter keeps the previously exported type working. It is a thin alias for a
// vendor-neutral adapter so existing callers and tests keep compiling.
type DefaultAdapter = VendorAdapter

// NewDefaultAdapter creates a vendor-neutral adapter.
func NewDefaultAdapter(manufacturer, model string) *DefaultAdapter {
	return newVendorAdapter(manufacturer, model, nil, nil)
}
