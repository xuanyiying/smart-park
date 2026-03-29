// Package biz provides business logic for the vehicle service.
package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	v1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/mqtt"
)

// CommandUseCase handles device command business logic.
type CommandUseCase struct {
	vehicleRepo VehicleRepo
	mqttClient  mqtt.Client
	log         *log.Helper
}

// NewCommandUseCase creates a new CommandUseCase.
func NewCommandUseCase(vehicleRepo VehicleRepo, mqttClient mqtt.Client, logger log.Logger) *CommandUseCase {
	return &CommandUseCase{
		vehicleRepo: vehicleRepo,
		mqttClient:  mqttClient,
		log:         log.NewHelper(logger),
	}
}

// SendCommand sends a command to a device via MQTT.
func (uc *CommandUseCase) SendCommand(ctx context.Context, deviceID string, command string, params map[string]string) (*v1.CommandData, error) {
	device, err := uc.vehicleRepo.GetDeviceByCode(ctx, deviceID)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to get device: %v", err)
		return nil, fmt.Errorf("device not found: %w", err)
	}

	if !device.Enabled {
		uc.log.WithContext(ctx).Warnf("device is disabled - DeviceID: %s", deviceID)
		return nil, fmt.Errorf("device is disabled")
	}

	if device.Status == "offline" {
		uc.log.WithContext(ctx).Warnf("device is offline - DeviceID: %s", deviceID)
		return nil, fmt.Errorf("device is offline")
	}

	if device.Status == "disabled" {
		uc.log.WithContext(ctx).Warnf("device status is disabled - DeviceID: %s", deviceID)
		return nil, fmt.Errorf("device status is disabled")
	}

	cmd := &mqtt.Command{
		CommandID: uuid.New().String(),
		DeviceID:  deviceID,
		Command:   mqtt.CommandType(command),
		Params:    params,
		Timestamp: time.Now().Unix(),
		Priority:  1,
	}

	if err := uc.mqttClient.PublishCommand(ctx, cmd); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to publish command: %v", err)
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	uc.log.WithContext(ctx).Infof("command sent to device %s, command_id: %s", deviceID, cmd.CommandID)

	return &v1.CommandData{
		CommandId: cmd.CommandID,
		Status:    "sent",
	}, nil
}
