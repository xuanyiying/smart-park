package middleware

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"

	"github.com/xuanyiying/smart-park/pkg/auth"
)

const (
	UserIDKey = "user_id"
	OpenIDKey = "open_id"
)

func JWTAuth(jwtManager *auth.JWTManager) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				token := tr.RequestHeader().Get("Authorization")
				if token == "" {
					return nil, errors.Unauthorized("UNAUTHORIZED", "missing authorization header")
				}

				if after, ok0 :=strings.CutPrefix(token, "Bearer "); ok0  {
					token = after
				}

				claims, err := jwtManager.ParseToken(token)
				if err != nil {
					return nil, errors.Unauthorized("UNAUTHORIZED", "invalid token")
				}

				ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
				ctx = context.WithValue(ctx, OpenIDKey, claims.OpenID)
			}

			return handler(ctx, req)
		}
	}
}

func GetUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

func GetOpenIDFromContext(ctx context.Context) string {
	if openID, ok := ctx.Value(OpenIDKey).(string); ok {
		return openID
	}
	return ""
}
