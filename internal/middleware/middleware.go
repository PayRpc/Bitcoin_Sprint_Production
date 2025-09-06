package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Middleware is a function type for HTTP middleware
type Middleware func(http.Handler) http.Handler

// RequestContext keys for storing request-scoped data
type contextKey string

const (
	RequestIDKey contextKey = "request_id"
	StartTimeKey contextKey = "start_time"
	ClientIPKey  contextKey = "client_ip"
	UserAgentKey contextKey = "user_agent"
)

// Config holds middleware configuration
type Config struct {
	EnableProfiling    bool
	EnableMetrics      bool
	EnableSecurity     bool
	EnableRecovery     bool
	EnableLogging      bool
	EnableCORS         bool
	TrustedProxies     []string
	SecurityHeaders    map[string]string
	AllowedOrigins     []string
	AllowedMethods     []string
	AllowedHeaders     []string
	MaxRequestSize     int64
	RequestTimeout     time.Duration
	EnableCompression  bool
	EnableRateLimiting bool
	Logger             *zap.Logger
}

// DefaultConfig returns production-ready middleware configuration
func DefaultConfig() *Config {
	return &Config{
		EnableProfiling:    false, // Enable only in development
		EnableMetrics:      true,
		EnableSecurity:     true,
		EnableRecovery:     true,
		EnableLogging:      true,
		EnableCORS:         true,
		TrustedProxies:     []string{"127.0.0.1", "::1"},
		MaxRequestSize:     10 * 1024 * 1024, // 10MB
		RequestTimeout:     30 * time.Second,
		EnableCompression:  true,
		EnableRateLimiting: true,
		SecurityHeaders: map[string]string{
			"X-Content-Type-Options":    "nosniff",
			"X-Frame-Options":           "DENY",
			"X-XSS-Protection":          "1; mode=block",
			"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
			"Content-Security-Policy":   "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'",
			"Referrer-Policy":           "strict-origin-when-cross-origin",
			"Permissions-Policy":        "geolocation=(), microphone=(), camera=()",
		},
		AllowedOrigins: []string{
			"http://localhost:3000",
			"http://localhost:3002",
			"https://*.bitcoin-sprint.com",
		},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD"},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-API-Key",
			"X-Request-ID",
			"X-Forwarded-For",
		},
	}
}

// Chain combines multiple middleware functions
func Chain(middlewares ...Middleware) Middleware {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

// RequestID generates and injects unique request IDs
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}

			w.Header().Set("X-Request-ID", requestID)
			ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Recovery catches panics and returns structured error responses
func Recovery(logger *zap.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					stack := debug.Stack()
					requestID := getRequestID(r.Context())

					if logger != nil {
						logger.Error("Panic recovered",
							zap.String("request_id", requestID),
							zap.Any("panic", rec),
							zap.String("stack", string(stack)),
							zap.String("method", r.Method),
							zap.String("url", r.URL.String()),
							zap.String("user_agent", r.UserAgent()),
							zap.String("remote_addr", r.RemoteAddr),
						)
					}

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, `{"error":"Internal Server Error","request_id":"%s"}`, requestID)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// Security applies comprehensive security headers and protections
func Security(config *Config) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Apply security headers
			for key, value := range config.SecurityHeaders {
				w.Header().Set(key, value)
			}

			// Block common attack paths
			path := strings.ToLower(r.URL.Path)
			suspiciousPaths := []string{
				"../", "..\\", "/.ht", "/.git", "/wp-", "/.env",
				"/admin", "/phpmyadmin", "/.well-known/",
				"/vendor/", "/config/", "/backup/",
			}

			for _, suspicious := range suspiciousPaths {
				if strings.Contains(path, suspicious) {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}
			}

			// Block suspicious user agents
			userAgent := strings.ToLower(r.UserAgent())
			suspiciousAgents := []string{"sqlmap", "nikto", "nmap", "dirb", "gobuster"}
			for _, agent := range suspiciousAgents {
				if strings.Contains(userAgent, agent) {
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORS handles Cross-Origin Resource Sharing
func CORS(config *Config) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range config.AllowedOrigins {
				if allowedOrigin == "*" || origin == allowedOrigin {
					allowed = true
					break
				}
				// Support wildcard subdomains
				if strings.HasPrefix(allowedOrigin, "*.") {
					domain := allowedOrigin[2:]
					if strings.HasSuffix(origin, domain) {
						allowed = true
						break
					}
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Profiling provides pprof endpoints for performance debugging
func Profiling(enabled bool) http.Handler {
	if !enabled {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Profiling disabled", http.StatusNotFound)
		})
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))

	return mux
}

// Metrics provides runtime and application metrics
func Metrics(enabled bool) http.HandlerFunc {
	if !enabled {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Metrics disabled", http.StatusNotFound)
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
  "runtime": {
    "goroutines": %d,
    "memory": {
      "alloc": %d,
      "total_alloc": %d,
      "sys": %d,
      "heap_alloc": %d,
      "heap_sys": %d,
      "gc_cycles": %d
    },
    "version": "%s",
    "arch": "%s",
    "os": "%s"
  },
  "timestamp": "%s"
}`,
			runtime.NumGoroutine(),
			m.Alloc,
			m.TotalAlloc,
			m.Sys,
			m.HeapAlloc,
			m.HeapSys,
			m.NumGC,
			runtime.Version(),
			runtime.GOARCH,
			runtime.GOOS,
			time.Now().Format(time.RFC3339),
		)
	}
}

// Logger provides structured request/response logging
func Logger(logger *zap.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ctx := context.WithValue(r.Context(), StartTimeKey, start)

			// Capture client IP
			clientIP := getClientIP(r)
			ctx = context.WithValue(ctx, ClientIPKey, clientIP)
			ctx = context.WithValue(ctx, UserAgentKey, r.UserAgent())

			// Wrap response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r.WithContext(ctx))

			duration := time.Since(start)
			requestID := getRequestID(r.Context())

			if logger != nil {
				logger.Info("Request completed",
					zap.String("request_id", requestID),
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("query", r.URL.RawQuery),
					zap.Int("status", wrapped.statusCode),
					zap.Duration("duration", duration),
					zap.String("client_ip", clientIP),
					zap.String("user_agent", r.UserAgent()),
					zap.Int64("response_size", wrapped.size),
				)
			}
		})
	}
}

// Timeout enforces request timeout limits
func Timeout(timeout time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, timeout, `{"error":"Request timeout","message":"Request took too long to process"}`)
	}
}

// Helper functions

func generateRequestID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("req_%d_%s", time.Now().UnixNano(), hex.EncodeToString(bytes))
}

func getRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return "unknown"
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (from proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to remote address
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}

	return r.RemoteAddr
}

// responseWriter wraps http.ResponseWriter to capture response details
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int64
	mu         sync.Mutex
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.mu.Lock()
	rw.statusCode = statusCode
	rw.mu.Unlock()
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(data)
	rw.mu.Lock()
	rw.size += int64(n)
	rw.mu.Unlock()
	return n, err
}
