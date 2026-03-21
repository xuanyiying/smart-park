// Package errors provides error definitions for the vehicle service.
package errors

import (
	"github.com/go-kratos/kratos/v2/errors"
)

// Vehicle service errors.
var (
	ErrVehicleNotFound      = errors.NotFound("VEHICLE_NOT_FOUND", "车辆不存在")
	ErrVehicleAlreadyExists = errors.Conflict("VEHICLE_ALREADY_EXISTS", "车辆已存在")
	ErrInvalidPlateNumber   = errors.BadRequest("INVALID_PLATE_NUMBER", "无效的车牌号")

	ErrRecordNotFound      = errors.NotFound("RECORD_NOT_FOUND", "停车记录不存在")
	ErrRecordAlreadyExited = errors.Conflict("RECORD_ALREADY_EXITED", "车辆已出场")
	ErrNoEntryRecord       = errors.NotFound("NO_ENTRY_RECORD", "未找到入场记录")

	ErrDeviceNotFound = errors.NotFound("DEVICE_NOT_FOUND", "设备不存在")
	ErrDeviceOffline  = errors.ServiceUnavailable("DEVICE_OFFLINE", "设备离线")
	ErrDeviceDisabled = errors.Forbidden("DEVICE_DISABLED", "设备已禁用")

	ErrLaneNotFound   = errors.NotFound("LANE_NOT_FOUND", "车道不存在")
	ErrLaneInactive   = errors.Forbidden("LANE_INACTIVE", "车道未激活")

	ErrParkingLotNotFound = errors.NotFound("PARKING_LOT_NOT_FOUND", "停车场不存在")
	ErrParkingLotFull     = errors.ResourceExhausted("PARKING_LOT_FULL", "停车场已满")

	ErrInvalidParameter = errors.BadRequest("INVALID_PARAMETER", "参数错误")
	ErrInternalError    = errors.InternalServer("INTERNAL_ERROR", "内部服务错误")
)
