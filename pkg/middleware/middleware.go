package middleware

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

func Recovery() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			defer func() {
				if r := recover(); r != nil {
					err = &RecoverError{
						Err:   r,
						Stack: string(debug.Stack()),
					}
				}
			}()
			return handler(ctx, req)
		}
	}
}

type RecoverError struct {
	Err   interface{}
	Stack string
}

func (e *RecoverError) Error() string {
	return fmt.Sprintf("panic recovered: %v\n%s", e.Err, e.Stack)
}

func Logging(logger log.Logger) middleware.Middleware {
	helper := log.NewHelper(logger)
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			startTime := time.Now()

			var operation string
			if tr, ok := transport.FromServerContext(ctx); ok {
				operation = tr.Operation()
			}

			reply, err = handler(ctx, req)

			duration := time.Since(startTime)

			if err != nil {
				helper.WithContext(ctx).Errorf("[%s] duration=%v error=%v", operation, duration, err)
			} else {
				helper.WithContext(ctx).Infof("[%s] duration=%v", operation, duration)
			}

			return reply, err
		}
	}
}

func Tracing() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			return handler(ctx, req)
		}
	}
}

func Metrics(name string) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			start := time.Now()
			reply, err = handler(ctx, req)
			_ = time.Since(start)
			return reply, err
		}
	}
}

func Timeout(timeout time.Duration) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			return handler(ctx, req)
		}
	}
}

func Validation() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if v, ok := req.(Validator); ok {
				if err := v.Validate(); err != nil {
					return nil, err
				}
			}
			return handler(ctx, req)
		}
	}
}

type Validator interface {
	Validate() error
}

type Chain []middleware.Middleware

func (c Chain) Append(m ...middleware.Middleware) Chain {
	return append(c, m...)
}

func (c Chain) Prepend(m ...middleware.Middleware) Chain {
	return append(Chain{}, append(m, c...)...)
}

func (c Chain) Then(h middleware.Handler) middleware.Handler {
	for i := len(c) - 1; i >= 0; i-- {
		h = c[i](h)
	}
	return h
}

func NewChain(m ...middleware.Middleware) Chain {
	return Chain(m)
}
