package ws

import (
	"net/http"
	"strings"
	"sync"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/xuanyiying/smart-park/pkg/auth"
)

type AuthMiddleware struct {
	jwtManager *auth.JWTManager
	log        *log.Helper
}

func NewAuthMiddleware(jwtManager *auth.JWTManager, logger log.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
		log:        log.NewHelper(logger),
	}
}

func (m *AuthMiddleware) Authenticate(r *http.Request) (*auth.Claims, error) {
	token := extractWSHeaderToken(r)
	if token == "" {
		return nil, ErrMissingToken
	}

	claims, err := m.jwtManager.ParseToken(token)
	if err != nil {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func extractWSHeaderToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}

	if token := r.Header.Get("Sec-WebSocket-Protocol"); token != "" {
		return token
	}

	return ""
}

type TenantMiddleware struct {
	log *log.Helper
}

func NewTenantMiddleware(logger log.Logger) *TenantMiddleware {
	return &TenantMiddleware{
		log: log.NewHelper(logger),
	}
}

func (m *TenantMiddleware) ExtractTenantID(claims *auth.Claims) string {
	if claims == nil {
		return ""
	}
	return ""
}

type RateLimitMiddleware struct {
	maxPerUser int
	connections map[string]int
	mu          sync.Mutex
	log         *log.Helper
}

func NewRateLimitMiddleware(maxPerUser int, logger log.Logger) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		maxPerUser:  maxPerUser,
		connections: make(map[string]int),
		log:         log.NewHelper(logger),
	}
}

func (m *RateLimitMiddleware) Acquire(userID string) bool {
	if userID == "" {
		return true
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.connections[userID] >= m.maxPerUser {
		m.log.Warnf("rate limit exceeded for user %s: %d/%d", userID, m.connections[userID], m.maxPerUser)
		return false
	}

	m.connections[userID]++
	return true
}

func (m *RateLimitMiddleware) Release(userID string) {
	if userID == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.connections[userID] > 0 {
		m.connections[userID]--
	}
	if m.connections[userID] == 0 {
		delete(m.connections, userID)
	}
}
