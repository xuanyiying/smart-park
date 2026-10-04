// Package data provides data access layer for the vehicle service.
package data

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/device"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/lane"
)

// ErrSeedLotNotConfigured is returned when seeding is requested without a target parking
// lot. Seeding used to attach placeholder lanes and devices to a
// parking lot that does not exist in any real deployment.
var ErrSeedLotNotConfigured = errors.New("seed: no target parking lot configured")

// SeedData creates the lanes and devices for the given parking lot.
//
// It is idempotent: seeding an already seeded lot is a no-op, so restarting a service in
// production does not accumulate duplicate rows.
//
// lotID must be supplied by the operator. Passing uuid.Nil skips seeding, which is the
// correct production default — a real car park's devices are registered through the
// device management API, not conjured at boot.
func (r *vehicleRepo) SeedData(ctx context.Context, lotID uuid.UUID) error {
	if lotID == uuid.Nil {
		return ErrSeedLotNotConfigured
	}

	count, err := r.data.db.Device.Query().Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		// Devices already exist, skip seeding
		return nil
	}

	now := time.Now()

	lanes := []struct {
		id        uuid.UUID
		laneNo    int
		direction lane.Direction
	}{
		{uuid.New(), 1, lane.DirectionEntry},
		{uuid.New(), 2, lane.DirectionEntry},
		{uuid.New(), 3, lane.DirectionExit},
		{uuid.New(), 4, lane.DirectionExit},
	}

	for _, l := range lanes {
		if _, err := r.data.db.Lane.Create().
			SetID(l.id).
			SetLaneNo(l.laneNo).
			SetLotID(lotID).
			SetDirection(l.direction).
			Save(ctx); err != nil {
			return err
		}
	}

	devices := []struct {
		deviceID   string
		deviceType device.DeviceType
		laneID     uuid.UUID
	}{
		{"CAM001", device.DeviceTypeCamera, lanes[0].id},
		{"GATE001", device.DeviceTypeGate, lanes[0].id},
		{"CAM002", device.DeviceTypeCamera, lanes[1].id},
		{"GATE002", device.DeviceTypeGate, lanes[1].id},
		{"CAM003", device.DeviceTypeCamera, lanes[2].id},
		{"GATE003", device.DeviceTypeGate, lanes[2].id},
		{"CAM004", device.DeviceTypeCamera, lanes[3].id},
		{"GATE004", device.DeviceTypeGate, lanes[3].id},
		{"DISP001", device.DeviceTypeDisplay, lanes[0].id},
		{"DISP002", device.DeviceTypeDisplay, lanes[2].id},
	}

	for _, d := range devices {
		// The secret authenticates the device when it connects, so it must be
		// unpredictable. "secret_"+deviceID was guessable for every device, which made
		// device authentication worthless.
		secret, err := generateDeviceSecret()
		if err != nil {
			return err
		}

		if _, err := r.data.db.Device.Create().
			SetDeviceID(d.deviceID).
			SetDeviceSecret(secret).
			SetDeviceType(d.deviceType).
			SetStatus(device.StatusActive).
			SetLaneID(d.laneID).
			SetLotID(lotID).
			SetLastHeartbeat(now).
			Save(ctx); err != nil {
			return err
		}
	}

	return nil
}

// generateDeviceSecret returns a random per-device credential.
func generateDeviceSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("seed: failed to generate device secret: %w", err)
	}
	return "dev_" + hex.EncodeToString(buf), nil
}
