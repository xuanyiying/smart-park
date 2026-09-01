package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/go-kratos/kratos/v2/middleware"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

type rawBodyKey struct{}

// CacheRawBody buffers the untouched HTTP request body and form so that downstream
// handlers can verify gateway signatures over the exact bytes that were signed.
//
// It must be installed before any body-binding middleware: Kratos decodes the request
// body into the protobuf message while invoking the route handler, and reading the body
// a second time without buffering yields an empty slice.
func CacheRawBody() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			r, ok := khttp.RequestFromServerContext(ctx)
			if !ok || r == nil || r.Body == nil {
				return handler(ctx, req)
			}

			body, err := io.ReadAll(r.Body)
			// Always restore the stream, otherwise request binding downstream fails.
			r.Body = io.NopCloser(bytes.NewReader(body))
			if err != nil {
				return handler(ctx, req)
			}

			// ParseForm populates r.PostForm for urlencoded callbacks (Alipay) without
			// consuming r.Body, so it is safe to call here.
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
				_ = r.ParseForm()
			}

			ctx = context.WithValue(ctx, rawBodyKey{}, &RawRequest{
				Body:    body,
				PostForm: r.PostForm,
				Header:  r.Header,
			})
			return handler(ctx, req)
		}
	}
}

// RawRequest holds the untouched payload of an inbound HTTP request.
type RawRequest struct {
	Body     []byte
	PostForm url.Values
	Header   http.Header
}

// RawRequestFromContext returns the request payload captured by CacheRawBody.
func RawRequestFromContext(ctx context.Context) (*RawRequest, bool) {
	raw, ok := ctx.Value(rawBodyKey{}).(*RawRequest)
	return raw, ok
}
