// Package biz provides business logic for the vehicle service.
package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/mqtt"
	v1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
)

// CommandUseCase handles device command business logic.
type CommandUseCase struct {
	mqttClient mqtt.Client
	log        *log.Helper
}

// NewCommandUseCase creates a new CommandUseCase.
func NewCommandUseCase(mqttClient mqtt.Client, logger log.Logger) *CommandUseCase {
	return &CommandUseCase{
		mqttClient: mqttClient,
		log:        log.NewHelper(logger),
	}
}

// SendCommand sends a command to a device via MQTT.
func (uc *CommandUseCase) SendCommand(ctx context.Context, deviceID string, command string, params map[string]string) (*v1.CommandData, error) {
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
