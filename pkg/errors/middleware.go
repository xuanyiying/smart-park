package errors

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
)

func ServerErrorHandler() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			resp, err := handler(ctx, req)

			if err == nil {
				return resp, nil
			}

			var se *Error
			var ok bool

			if se, ok = err.(*Error); !ok {
				if !isContextError(err) {
					se = Internalf("%v", err)
				}
			}

			return resp, toKratosError(se)
		}
	}
}

func isContextError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}

func toKratosError(e *Error) error {
	if e == nil {
		return nil
	}

	httpx := e.HTTPStatus()
	return errors.New(httpx, string(e.Code), e.Message)
}

func FromKratos(kerr *errors.Error) *Error {
	if kerr == nil {
		return nil
	}

	code := CodeInternal
	switch kerr.Code / 100 {
	case 4:
		switch kerr.Code {
		case 400:
			code = CodeInvalidArgument
		case 401:
			code = CodeUnauthenticated
		case 403:
			code = CodePermissionDenied
		case 404:
			code = CodeNotFound
		case 409:
			code = CodeAlreadyExists
		case 429:
			code = CodeResourceExhausted
		default:
			code = CodeInvalidArgument
		}
	case 5:
		code = CodeInternal
	}

	return &Error{
		Code:    code,
		Message: kerr.Reason,
		Details: kerr.Message,
	}
}

func IsAppError(err error) bool {
	_, ok := err.(*Error)
	return ok
}

func IsKratosError(err error) bool {
	_, ok := err.(*errors.Error)
	return ok
}

func ConvertError(err error) error {
	if err == nil {
		return nil
	}

	if IsAppError(err) {
		return err
	}

	if IsKratosError(err) {
		return err
	}

	return Internalf("%v", err)
}

func ToKratos(e *Error) *errors.Error {
	if e == nil {
		return nil
	}
	return errors.New(e.HTTPStatus(), string(e.Code), e.Message)
}
