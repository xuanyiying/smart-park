// Package middleware provides HTTP middleware components for the gateway.
package middleware

import (
	"io"
	"log"
	"net/http"
	"time"
)

// LogMiddleware 日志中间件
type LogMiddleware struct {
	logger *log.Logger
}

// NewLogMiddleware 创建日志中间件
func NewLogMiddleware(logger *log.Logger) *LogMiddleware {
	if logger == nil {
		logger = log.Default()
	}
	return &LogMiddleware{logger: logger}
}

// Handler 返回 HTTP 处理函数
func (m *LogMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// 包装 ResponseWriter 以获取状态码
		wrapper := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(wrapper, r)

		duration := time.Since(startTime)
		m.logger.Printf("[%s] %s %s - %d - %v",
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			wrapper.statusCode,
			duration,
		)
	})
}

// responseWriterWrapper 包装 ResponseWriter 以捕获状态码
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	// 如果状态码是 204 No Content，不写入响应体
	if w.statusCode == http.StatusNoContent {
		return 0, nil
	}
	return w.ResponseWriter.Write(b)
}

// LoggerWithRequest 使用请求信息创建日志中间件
func LoggerWithRequest(logger *log.Logger, pretty bool) func(http.Handler) http.Handler {
	m := NewLogMiddleware(logger)
	if pretty {
		return m.PrettyHandler
	}
	return m.Handler
}

// PrettyHandler 格式化日志处理函数
func (m *LogMiddleware) PrettyHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		wrapper := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(wrapper, r)

		duration := time.Since(startTime)

		// 美化输出
		m.logger.Printf("┌─────────────────────────────────────────────────────┐")
		m.logger.Printf("│ %-10s %s %s", r.Method, r.URL.Path, padSpaces(r.URL.Path, 50))
		m.logger.Printf("│ Status:  %d  Duration: %v", wrapper.statusCode, duration)
		m.logger.Printf("│ Client:  %s", r.RemoteAddr)
		m.logger.Printf("└─────────────────────────────────────────────────────┘")
	})
}

func padSpaces(s string, length int) string {
	if len(s) >= length {
		return ""
	}
	return string(make([]byte, length-len(s)))
}

// RecoverMiddleware 恢复中间件，处理 panic
type RecoverMiddleware struct {
	logger *log.Logger
}

// NewRecoverMiddleware 创建恢复中间件
func NewRecoverMiddleware(logger *log.Logger) *RecoverMiddleware {
	if logger == nil {
		logger = log.Default()
	}
	return &RecoverMiddleware{logger: logger}
}

// Handler 返回 HTTP 处理函数
func (m *RecoverMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				m.logger.Printf("panic recovered: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware CORS 中间件
type CORSMiddleware struct {
	allowedOrigins []string
	allowedMethods []string
	allowedHeaders []string
}

// NewCORSMiddleware 创建 CORS 中间件
func NewCORSMiddleware(allowedOrigins, allowedMethods, allowedHeaders []string) *CORSMiddleware {
	if len(allowedMethods) == 0 {
		allowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if len(allowedHeaders) == 0 {
		allowedHeaders = []string{"Content-Type", "Authorization", "X-Requested-With"}
	}
	return &CORSMiddleware{
		allowedOrigins: allowedOrigins,
		allowedMethods: allowedMethods,
		allowedHeaders: allowedHeaders,
	}
}

// Handler 返回 HTTP 处理函数
func (m *CORSMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 处理预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 设置 CORS 头
		origin := r.Header.Get("Origin")
		if origin != "" {
			// 检查 origin 是否允许
			allowed := len(m.allowedOrigins) == 0
			for _, o := range m.allowedOrigins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}
			if allowed {
				if len(m.allowedOrigins) == 1 && m.allowedOrigins[0] == "*" {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
			}
		}

		w.Header().Set("Access-Control-Allow-Methods", joinStrings(m.allowedMethods, ", "))
		w.Header().Set("Access-Control-Allow-Headers", joinStrings(m.allowedHeaders, ", "))
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400")

		next.ServeHTTP(w, r)
	})
}

func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for i := 1; i < len(ss); i++ {
		result += sep + ss[i]
	}
	return result
}

// HTTPChain 链式调用多个 HTTP 中间件
func HTTPChain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for _, m := range middlewares {
		h = m(h)
	}
	return h
}

// NopCloser 丢弃响应体的包装器
type NopCloser struct {
	http.ResponseWriter
}

func (n *NopCloser) Write(b []byte) (int, error) {
	return len(b), nil
}

var _ io.Closer = (*NopCloser)(nil)

func (n *NopCloser) Close() error {
	return nil
}