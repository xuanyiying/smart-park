package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestError_Codes(t *testing.T) {
	tests := []struct {
		name       string
		code       Code
		wantStatus int
	}{
		{"InvalidArgument", CodeInvalidArgument, http.StatusBadRequest},
		{"NotFound", CodeNotFound, http.StatusNotFound},
		{"AlreadyExists", CodeAlreadyExists, http.StatusConflict},
		{"PermissionDenied", CodePermissionDenied, http.StatusForbidden},
		{"ResourceExhausted", CodeResourceExhausted, http.StatusTooManyRequests},
		{"FailedPrecondition", CodeFailedPrecondition, http.StatusPreconditionFailed},
		{"Aborted", CodeAborted, http.StatusConflict},
		{"OutOfRange", CodeOutOfRange, http.StatusBadRequest},
		{"Unimplemented", CodeUnimplemented, http.StatusNotImplemented},
		{"Internal", CodeInternal, http.StatusInternalServerError},
		{"Unavailable", CodeUnavailable, http.StatusServiceUnavailable},
		{"DataLoss", CodeDataLoss, http.StatusInternalServerError},
		{"Unauthenticated", CodeUnauthenticated, http.StatusUnauthorized},
		{"Unknown", CodeUnknown, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := New(tt.code, "test message")
			if got := e.HTTPStatus(); got != tt.wantStatus {
				t.Errorf("HTTPStatus() = %v, want %v", got, tt.wantStatus)
			}
		})
	}
}

func TestError_Error(t *testing.T) {
	tests := []struct {
		name   string
		e      *Error
		expect string
	}{
		{
			name:   "without cause",
			e:      New(CodeInvalidArgument, "test message"),
			expect: "INVALID_ARGUMENT: test message",
		},
		{
			name:   "with cause",
			e:      Wrap(errors.New("cause error"), CodeInternal, "wrapper message"),
			expect: "INTERNAL: wrapper message: cause error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.Error(); got != tt.expect {
				t.Errorf("Error() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestError_Wrap(t *testing.T) {
	cause := errors.New("original error")
	e := Wrap(cause, CodeInternal, "wrapped error")

	if !errors.Is(e, cause) {
		t.Error("Wrapped error should contain original cause")
	}
}

func TestError_WithCause(t *testing.T) {
	original := New(CodeInvalidArgument, "original")
	cause := errors.New("new cause")

	wrapped := original.WithCause(cause)

	if wrapped.Cause != cause {
		t.Error("WithCause should set the cause")
	}
}

func TestError_WithDetails(t *testing.T) {
	e := New(CodeInvalidArgument, "error")
	detailed := e.WithDetails("additional details")

	if detailed.Details != "additional details" {
		t.Error("WithDetails should set details")
	}
}

func TestError_WithMessage(t *testing.T) {
	e := New(CodeInvalidArgument, "original message")
	newMsg := e.WithMessage("new message")

	if newMsg.Message != "new message" {
		t.Error("WithMessage should update message")
	}
}

func TestError_FactoryMethods(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(string) *Error
		message  string
		wantCode Code
	}{
		{"InvalidArgument", InvalidArgument, "test", CodeInvalidArgument},
		{"NotFound", NotFound, "test", CodeNotFound},
		{"AlreadyExists", AlreadyExists, "test", CodeAlreadyExists},
		{"PermissionDenied", PermissionDenied, "test", CodePermissionDenied},
		{"ResourceExhausted", ResourceExhausted, "test", CodeResourceExhausted},
		{"FailedPrecondition", FailedPrecondition, "test", CodeFailedPrecondition},
		{"Aborted", Aborted, "test", CodeAborted},
		{"OutOfRange", OutOfRange, "test", CodeOutOfRange},
		{"Unimplemented", Unimplemented, "test", CodeUnimplemented},
		{"Internal", Internal, "test", CodeInternal},
		{"Unavailable", Unavailable, "test", CodeUnavailable},
		{"DataLoss", DataLoss, "test", CodeDataLoss},
		{"Unauthenticated", Unauthenticated, "test", CodeUnauthenticated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := tt.fn(tt.message)
			if e.Code != tt.wantCode {
				t.Errorf("Code = %v, want %v", e.Code, tt.wantCode)
			}
			if e.Message != tt.message {
				t.Errorf("Message = %v, want %v", e.Message, tt.message)
			}
		})
	}
}

func TestError_Errorf(t *testing.T) {
	e := Errorf(CodeInvalidArgument, "user %s not found", "john")

	if e.Code != CodeInvalidArgument {
		t.Errorf("Code = %v, want %v", e.Code, CodeInvalidArgument)
	}
	if e.Message != "user john not found" {
		t.Errorf("Message = %v, want 'user john not found'", e.Message)
	}
}

func TestGetCode(t *testing.T) {
	appErr := New(CodeNotFound, "not found")
	if got := GetCode(appErr); got != CodeNotFound {
		t.Errorf("GetCode() = %v, want %v", got, CodeNotFound)
	}

	standardErr := errors.New("standard error")
	if got := GetCode(standardErr); got != CodeUnknown {
		t.Errorf("GetCode() = %v, want %v", got, CodeUnknown)
	}
}

func TestGetMessage(t *testing.T) {
	appErr := New(CodeNotFound, "not found")
	if got := GetMessage(appErr); got != "not found" {
		t.Errorf("GetMessage() = %v, want 'not found'", got)
	}

	standardErr := errors.New("standard error")
	if got := GetMessage(standardErr); got != "standard error" {
		t.Errorf("GetMessage() = %v, want 'standard error'", got)
	}
}

func TestIs(t *testing.T) {
	e1 := New(CodeNotFound, "not found")
	e2 := New(CodeNotFound, "another not found")
	e3 := New(CodeInternal, "internal error")

	if !Is(e1, e2) {
		t.Error("Is() should return true for same error code")
	}

	if Is(e1, e3) {
		t.Error("Is() should return false for different error codes")
	}
}

func TestAs(t *testing.T) {
	original := New(CodeNotFound, "not found")
	var target *Error

	if !As(original, &target) {
		t.Error("As() should return true for matching type")
	}

	if target.Code != CodeNotFound {
		t.Errorf("As() target.Code = %v, want %v", target.Code, CodeNotFound)
	}
}
