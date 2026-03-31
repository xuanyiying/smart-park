// Package middleware provides HTTP middleware components for the gateway.
package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/xuanyiying/smart-park/pkg/auth"
)

// JWTClaimsKey is the context key for JWT claims
const JWTClaimsKey = "jwt_claims"

// JWTMiddleware JWT 认证中间件
type JWTMiddleware struct {
	jwtManager *auth.JWTManager
	log        *log.Helper
	skipPaths  []string // 不需要认证的路径
}

// NewJWTMiddleware 创建 JWT 中间件
func NewJWTMiddleware(jwtManager *auth.JWTManager, logger log.Logger, skipPaths []string) *JWTMiddleware {
	return &JWTMiddleware{
		jwtManager: jwtManager,
		log:        log.NewHelper(logger),
		skipPaths:  skipPaths,
	}
}

// Handler 返回 HTTP 处理函数
func (m *JWTMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查是否跳过认证
		if m.shouldSkip(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// 从 Header 获取 Token
		token := m.extractToken(r)
		if token == "" {
			m.log.Warnf("missing token for path: %s", r.URL.Path)
			http.Error(w, "Unauthorized: missing token", http.StatusUnauthorized)
			return
		}

		// 验证 Token
		claims, err := m.jwtManager.ParseToken(token)
		if err != nil {
			m.log.Warnf("invalid token: %v, path: %s", err, r.URL.Path)
			http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
			return
		}

		// 将 claims 放入 context
		ctx := context.WithValue(r.Context(), JWTClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractToken 从请求中提取 Token
// 支持从 Authorization header (Bearer token) 或 query parameter (token)
func (m *JWTMiddleware) extractToken(r *http.Request) string {
	// 1. 从 Authorization header 获取
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	// 2. 从 query parameter 获取
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}

	// 3. 从 cookie 获取
	if cookie, err := r.Cookie("token"); err == nil {
		return cookie.Value
	}

	return ""
}

// shouldSkip 检查路径是否需要跳过认证
func (m *JWTMiddleware) shouldSkip(path string) bool {
	for _, skipPath := range m.skipPaths {
		// 支持通配符匹配
		if strings.HasSuffix(skipPath, "*") {
			prefix := strings.TrimSuffix(skipPath, "*")
			if strings.HasPrefix(path, prefix) {
				return true
			}
		} else if path == skipPath || strings.HasPrefix(path, skipPath+"/") {
			return true
		}
	}
	return false
}

// GetClaimsFromContext 从 context 获取 JWT claims
func GetClaimsFromContext(ctx context.Context) (*auth.Claims, error) {
	claims, ok := ctx.Value(JWTClaimsKey).(*auth.Claims)
	if !ok || claims == nil {
		return nil, errors.New("no JWT claims found in context")
	}
	return claims, nil
}

// RequireAuth 强制要求认证的中间件包装器
func RequireAuth(jwtManager *auth.JWTManager, logger log.Logger) func(http.Handler) http.Handler {
	m := NewJWTMiddleware(jwtManager, logger, []string{})
	return m.Handler
}

// SkipAuth 跳过认证的辅助函数，用于创建不需要认证的中间件
func SkipAuth(skipPaths []string) func(http.Handler) http.Handler {
	m := &JWTMiddleware{
		skipPaths: skipPaths,
		log:       log.NewHelper(nil),
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if m.shouldSkip(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			// 非跳过路径要求认证
			token := m.extractToken(r)
			if token == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}