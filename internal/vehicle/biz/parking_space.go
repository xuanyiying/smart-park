// Package biz provides business logic for the vehicle service.
package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// ParkingSpaceStatusUpdate represents a parking space status update message
type ParkingSpaceStatusUpdate struct {
	SpaceID      string    `json:"space_id"`
	DeviceID     string    `json:"device_id"`
	Status       string    `json:"status"`
	VehiclePlate *string   `json:"vehicle_plate,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// ParkingSpaceUseCase handles parking space business logic
type ParkingSpaceUseCase struct {
	vehicleRepo VehicleRepo
	config      *Config
	log         *log.Helper
}

// NewParkingSpaceUseCase creates a new ParkingSpaceUseCase
func NewParkingSpaceUseCase(vehicleRepo VehicleRepo, logger log.Logger) *ParkingSpaceUseCase {
	return &ParkingSpaceUseCase{
		vehicleRepo: vehicleRepo,
		config:      DefaultConfig(),
		log:         log.NewHelper(logger),
	}
}

// HandleParkingSpaceStatusUpdate handles parking space status update
func (uc *ParkingSpaceUseCase) HandleParkingSpaceStatusUpdate(ctx context.Context, update *ParkingSpaceStatusUpdate) error {
	// Validate input
	if update.SpaceID == "" {
		return fmt.Errorf("space_id is required")
	}
	if update.DeviceID == "" {
		return fmt.Errorf("device_id is required")
	}
	if update.Status == "" {
		return fmt.Errorf("status is required")
	}

	// Check if device exists
	device, err := uc.vehicleRepo.GetDeviceByID(ctx, update.DeviceID)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to get device: %v", err)
		return fmt.Errorf("failed to get device: %w", err)
	}
	if device == nil {
		return fmt.Errorf("device not found: %s", update.DeviceID)
	}

	// Update parking space status
	if err := uc.vehicleRepo.UpdateParkingSpaceStatus(ctx, &ParkingSpace{
		SpaceID:      update.SpaceID,
		DeviceID:     update.DeviceID,
		LotID:        device.LotID,
		Status:       update.Status,
		VehiclePlate: update.VehiclePlate,
		LastUpdate:   update.Timestamp,
	}); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to update parking space status: %v", err)
		return fmt.Errorf("failed to update parking space status: %w", err)
	}

	uc.log.WithContext(ctx).Infof("Parking space %s status updated to %s", update.SpaceID, update.Status)
	return nil
}

// // GetParkingSpaceStatus retrieves parking space status
// func (uc *ParkingSpaceUseCase) GetParkingSpaceStatus(ctx context.Context, spaceID string) (*v1.ParkingSpaceStatus, error) {
// 	if spaceID == "" {
// 		return nil, fmt.Errorf("space_id is required")
// 	}
// 
// 	space, err := uc.vehicleRepo.GetParkingSpaceByID(ctx, spaceID)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get parking space: %w", err)
// 	}
// 	if space == nil {
// 		return nil, fmt.Errorf("parking space not found: %s", spaceID)
// 	}
// 
// 	return &v1.ParkingSpaceStatus{
// 		SpaceId:      space.SpaceID,
// 		Status:       space.Status,
// 		LastUpdate:   space.LastUpdate.Format(time.RFC3339),
// 		VehiclePlate: space.VehiclePlate,
// 		DeviceId:     space.DeviceID,
// 	}, nil
// }
// 
// // ListParkingSpaces lists parking spaces with pagination
// func (uc *ParkingSpaceUseCase) ListParkingSpaces(ctx context.Context, lotID string, page, pageSize int) ([]*v1.ParkingSpaceStatus, int, error) {
// 	var lotUUID *uuid.UUID
// 	if lotID != "" {
// 		id, err := uuid.Parse(lotID)
// 		if err != nil {
// 			return nil, 0, fmt.Errorf("invalid lot id: %w", err)
// 		}
// 		lotUUID = &id
// 	}
// 
// 	spaces, total, err := uc.vehicleRepo.ListParkingSpaces(ctx, lotUUID, page, pageSize)
// 	if err != nil {
// 		return nil, 0, err
// 	}
// 
// 	result := make([]*v1.ParkingSpaceStatus, len(spaces))
// 	for i, space := range spaces {
// 		result[i] = &v1.ParkingSpaceStatus{
// 			SpaceId:      space.SpaceID,
// 			Status:       space.Status,
// 			LastUpdate:   space.LastUpdate.Format(time.RFC3339),
// 			VehiclePlate: space.VehiclePlate,
// 			DeviceId:     space.DeviceID,
// 		}
// 	}
// 
// 	return result, total, nil
// }

// ParkingSpaceMessageHandler handles MQTT messages for parking space status updates
func (uc *ParkingSpaceUseCase) ParkingSpaceMessageHandler(client mqtt.Client, msg mqtt.Message) {
	ctx := context.Background()
	uc.log.Infof("Received message on topic %s: %s", msg.Topic(), string(msg.Payload()))

	// Parse message payload
	var update ParkingSpaceStatusUpdate
	if err := json.Unmarshal(msg.Payload(), &update); err != nil {
		uc.log.Errorf("Failed to parse parking space status update: %v", err)
		return
	}

	// Set timestamp if not provided
	if update.Timestamp.IsZero() {
		update.Timestamp = time.Now()
	}

	// Handle status update
	if err := uc.HandleParkingSpaceStatusUpdate(ctx, &update); err != nil {
		uc.log.Errorf("Failed to handle parking space status update: %v", err)
	}
}
