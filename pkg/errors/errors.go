package errors

import (
	"fmt"
	"net/http"
)

type Code string

const (
	CodeUnknown            Code = "UNKNOWN"
	CodeInvalidArgument    Code = "INVALID_ARGUMENT"
	CodeNotFound           Code = "NOT_FOUND"
	CodeAlreadyExists      Code = "ALREADY_EXISTS"
	CodePermissionDenied   Code = "PERMISSION_DENIED"
	CodeResourceExhausted  Code = "RESOURCE_EXHAUSTED"
	CodeFailedPrecondition Code = "FAILED_PRECONDITION"
	CodeAborted            Code = "ABORTED"
	CodeOutOfRange         Code = "OUT_OF_RANGE"
	CodeUnimplemented      Code = "UNIMPLEMENTED"
	CodeInternal           Code = "INTERNAL"
	CodeUnavailable        Code = "UNAVAILABLE"
	CodeDataLoss           Code = "DATA_LOSS"
	CodeUnauthenticated    Code = "UNAUTHENTICATED"
)

type Error struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	Cause   error  `json:"-"`
	Details string `json:"details,omitempty"`
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func (e *Error) WithCause(err error) *Error {
	return &Error{
		Code:    e.Code,
		Message: e.Message,
		Cause:   err,
		Details: e.Details,
	}
}

func (e *Error) WithDetails(details string) *Error {
	return &Error{
		Code:    e.Code,
		Message: e.Message,
		Cause:   e.Cause,
		Details: details,
	}
}

func (e *Error) WithMessage(msg string) *Error {
	return &Error{
		Code:    e.Code,
		Message: msg,
		Cause:   e.Cause,
		Details: e.Details,
	}
}

func (e *Error) HTTPStatus() int {
	switch e.Code {
	case CodeInvalidArgument:
		return http.StatusBadRequest
	case CodeNotFound:
		return http.StatusNotFound
	case CodeAlreadyExists:
		return http.StatusConflict
	case CodePermissionDenied:
		return http.StatusForbidden
	case CodeResourceExhausted:
		return http.StatusTooManyRequests
	case CodeFailedPrecondition:
		return http.StatusPreconditionFailed
	case CodeAborted:
		return http.StatusConflict
	case CodeOutOfRange:
		return http.StatusBadRequest
	case CodeUnimplemented:
		return http.StatusNotImplemented
	case CodeInternal:
		return http.StatusInternalServerError
	case CodeUnavailable:
		return http.StatusServiceUnavailable
	case CodeDataLoss:
		return http.StatusInternalServerError
	case CodeUnauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

func New(code Code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

func Errorf(code Code, format string, args ...interface{}) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

func InvalidArgument(message string) *Error {
	return New(CodeInvalidArgument, message)
}

func InvalidArgumentf(format string, args ...interface{}) *Error {
	return Errorf(CodeInvalidArgument, format, args...)
}

func NotFound(message string) *Error {
	return New(CodeNotFound, message)
}

func NotFoundf(format string, args ...interface{}) *Error {
	return Errorf(CodeNotFound, format, args...)
}

func AlreadyExists(message string) *Error {
	return New(CodeAlreadyExists, message)
}

func PermissionDenied(message string) *Error {
	return New(CodePermissionDenied, message)
}

func ResourceExhausted(message string) *Error {
	return New(CodeResourceExhausted, message)
}

func FailedPrecondition(message string) *Error {
	return New(CodeFailedPrecondition, message)
}

func Aborted(message string) *Error {
	return New(CodeAborted, message)
}

func OutOfRange(message string) *Error {
	return New(CodeOutOfRange, message)
}

func Unimplemented(message string) *Error {
	return New(CodeUnimplemented, message)
}

func Internal(message string) *Error {
	return New(CodeInternal, message)
}

func Internalf(format string, args ...interface{}) *Error {
	return Errorf(CodeInternal, format, args...)
}

func Unavailable(message string) *Error {
	return New(CodeUnavailable, message)
}

func DataLoss(message string) *Error {
	return New(CodeDataLoss, message)
}

func Unauthenticated(message string) *Error {
	return New(CodeUnauthenticated, message)
}

func Wrap(err error, code Code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   err,
	}
}

func Wrapf(err error, code Code, format string, args ...interface{}) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   err,
	}
}

func Is(err, target error) bool {
	if e, ok := err.(*Error); ok {
		if t, ok := target.(*Error); ok {
			return e.Code == t.Code
		}
	}
	return false
}

func As(err error, target interface{}) bool {
	if e, ok := err.(*Error); ok {
		if t, ok := target.(**Error); ok {
			*t = e
			return true
		}
	}
	return false
}

func GetCode(err error) Code {
	if e, ok := err.(*Error); ok {
		return e.Code
	}
	return CodeUnknown
}

func GetMessage(err error) string {
	if e, ok := err.(*Error); ok {
		return e.Message
	}
	return err.Error()
}

// Business errors
var (
	ErrVehicleNotFound      = New(CodeNotFound, "车辆不存在")
	ErrVehicleAlreadyExists = New(CodeAlreadyExists, "车辆已存在")
	ErrInvalidPlateNumber   = New(CodeInvalidArgument, "无效的车牌号")

	ErrRecordNotFound      = New(CodeNotFound, "停车记录不存在")
	ErrRecordAlreadyExited = New(CodeAlreadyExists, "车辆已出场")
	ErrNoEntryRecord       = New(CodeNotFound, "未找到入场记录")

	ErrDeviceNotFound = New(CodeNotFound, "设备不存在")
	ErrDeviceOffline  = New(CodeUnavailable, "设备离线")
	ErrDeviceDisabled = New(CodePermissionDenied, "设备已禁用")

	ErrLaneNotFound = New(CodeNotFound, "车道不存在")
	ErrLaneInactive = New(CodePermissionDenied, "车道未激活")

	ErrParkingLotNotFound = New(CodeNotFound, "停车场不存在")
	ErrParkingLotFull     = New(CodeResourceExhausted, "停车场已满")

	ErrOrderNotFound       = New(CodeNotFound, "订单不存在")
	ErrOrderAlreadyPaid    = New(CodeAlreadyExists, "订单已支付")
	ErrRefundAlreadyExists = New(CodeAlreadyExists, "退款申请已存在")
	ErrRefundRejected      = New(CodePermissionDenied, "退款申请被拒绝")

	ErrBillingRuleNotFound = New(CodeNotFound, "计费规则不存在")

	ErrInvalidParameter = New(CodeInvalidArgument, "参数错误")
	ErrInternalError    = New(CodeInternal, "内部服务错误")
)
