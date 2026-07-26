// Package device provides device management functionality.
package device

import (
	"context"
	"errors"
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
	adapters map[string]DeviceAdapter
}

// NewAdapterFactory creates a new AdapterFactory.
func NewAdapterFactory() *AdapterFactory {
	return &AdapterFactory{
		adapters: make(map[string]DeviceAdapter),
	}
}

// Register registers a device adapter for a specific manufacturer and model.
func (f *AdapterFactory) Register(manufacturer, model string, adapter DeviceAdapter) {
	key := manufacturer + ":" + model
	f.adapters[key] = adapter
}

// GetAdapter returns a device adapter for the specified manufacturer and model.
func (f *AdapterFactory) GetAdapter(manufacturer, model string) (DeviceAdapter, error) {
	key := manufacturer + ":" + model
	adapter, ok := f.adapters[key]
	if !ok {
		// Try to find an adapter for the manufacturer only
		key = manufacturer + ":*"
		adapter, ok = f.adapters[key]
		if !ok {
			return nil, errors.New("no adapter found for manufacturer: " + manufacturer + " and model: " + model)
		}
	}
	return adapter, nil
}

// DefaultAdapter is a default adapter that implements the DeviceAdapter interface.
type DefaultAdapter struct {
	manufacturer string
	model        string
}

// NewDefaultAdapter creates a new DefaultAdapter.
func NewDefaultAdapter(manufacturer, model string) *DefaultAdapter {
	return &DefaultAdapter{
		manufacturer: manufacturer,
		model:        model,
	}
}

// OpenGate opens the gate
func (a *DefaultAdapter) OpenGate(ctx context.Context, deviceID string) error {
	// Default implementation
	return nil
}

// CloseGate closes the gate
func (a *DefaultAdapter) CloseGate(ctx context.Context, deviceID string) error {
	// Default implementation
	return nil
}

// GetDeviceStatus gets the device status
func (a *DefaultAdapter) GetDeviceStatus(ctx context.Context, deviceID string) (map[string]interface{}, error) {
	// Default implementation
	return map[string]interface{}{"status": "unknown"}, nil
}

// SendCommand sends a custom command to the device
func (a *DefaultAdapter) SendCommand(ctx context.Context, deviceID string, command string, params map[string]interface{}) (map[string]interface{}, error) {
	// Default implementation
	return map[string]interface{}{"result": "unknown"}, nil
}

// GetManufacturer returns the manufacturer name
func (a *DefaultAdapter) GetManufacturer() string {
	return a.manufacturer
}

// GetModel returns the device model
func (a *DefaultAdapter) GetModel() string {
	return a.model
}
