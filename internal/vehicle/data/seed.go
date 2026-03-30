// Package data provides data access layer for the vehicle service.
package data

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/device"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/lane"
)

// SeedData creates initial seed data for development.
func (r *vehicleRepo) SeedData(ctx context.Context) error {
	// Check if devices already exist
	count, err := r.data.db.Device.Query().Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		// Devices already exist, skip seeding
		return nil
	}

	// Create sample lanes and devices for parking lot 11111111-1111-1111-1111-111111111111
	lotID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	now := time.Now()

	// Create lanes
	lanes := []struct {
		id        uuid.UUID
		laneNo    int
		lotID     uuid.UUID
		direction lane.Direction
		name      string
	}{
		{uuid.New(), 1, lotID, lane.DirectionEntry, "入口车道1"},
		{uuid.New(), 2, lotID, lane.DirectionEntry, "入口车道2"},
		{uuid.New(), 3, lotID, lane.DirectionExit, "出口车道1"},
		{uuid.New(), 4, lotID, lane.DirectionExit, "出口车道2"},
	}

	for _, l := range lanes {
		_, err := r.data.db.Lane.Create().
			SetID(l.id).
			SetLaneNo(l.laneNo).
			SetLotID(l.lotID).
			SetDirection(l.direction).
			Save(ctx)
		if err != nil {
			return err
		}
	}

	// Create devices
	devices := []struct {
		deviceID   string
		deviceType device.DeviceType
		status     device.Status
		laneID     uuid.UUID
		lotID      uuid.UUID
		name       string
	}{
		{"CAM001", device.DeviceTypeCamera, device.StatusActive, lanes[0].id, lotID, "入口摄像头1"},
		{"GATE001", device.DeviceTypeGate, device.StatusActive, lanes[0].id, lotID, "入口道闸1"},
		{"CAM002", device.DeviceTypeCamera, device.StatusActive, lanes[1].id, lotID, "入口摄像头2"},
		{"GATE002", device.DeviceTypeGate, device.StatusActive, lanes[1].id, lotID, "入口道闸2"},
		{"CAM003", device.DeviceTypeCamera, device.StatusActive, lanes[2].id, lotID, "出口摄像头1"},
		{"GATE003", device.DeviceTypeGate, device.StatusActive, lanes[2].id, lotID, "出口道闸1"},
		{"CAM004", device.DeviceTypeCamera, device.StatusActive, lanes[3].id, lotID, "出口摄像头2"},
		{"GATE004", device.DeviceTypeGate, device.StatusActive, lanes[3].id, lotID, "出口道闸2"},
		{"DISP001", device.DeviceTypeDisplay, device.StatusActive, lanes[0].id, lotID, "入口显示屏1"},
		{"DISP002", device.DeviceTypeDisplay, device.StatusActive, lanes[2].id, lotID, "出口显示屏1"},
	}

	for _, d := range devices {
		_, err := r.data.db.Device.Create().
			SetDeviceID(d.deviceID).
			SetDeviceSecret("secret_" + d.deviceID).
			SetDeviceType(d.deviceType).
			SetStatus(d.status).
			SetLaneID(d.laneID).
			SetLotID(d.lotID).
			SetLastHeartbeat(now).
			Save(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
