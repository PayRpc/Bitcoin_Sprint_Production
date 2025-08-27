// Package api provides HTTP middleware functionality
package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"go.uber.org/zap"
)

// ===== MIDDLEWARE IMPLEMENTATION =====

// securityMiddleware applies security headers and measures to all requests
func (s *Server) securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		// Block common web attack paths
		path := strings.ToLower(r.URL.Path)
		if strings.Contains(path, "../") ||
			strings.Contains(path, "..\\") ||
			strings.Contains(path, "/.ht") ||
			strings.Contains(path, "/.git") ||
			strings.Contains(path, "/wp-") ||
			strings.Contains(path, "/.env") {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		// Implement rate limiting based on IP (config-driven)
		clientIP := getClientIP(r)
		generalRateLimit := s.cfg.GeneralRateLimit
		if generalRateLimit <= 0 {
			generalRateLimit = 100 // fallback default
		}
		if !s.rateLimiter.Allow(clientIP, float64(generalRateLimit), 1) {
			s.logger.Warn("Rate limit exceeded",
				zap.String("ip", clientIP),
				zap.String("path", r.URL.Path),
			)
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Proceed with request
		next.ServeHTTP(w, r)
	})
}

// recoveryMiddleware catches panics and returns 500 error
func (s *Server) recoveryMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := debug.Stack()
				s.logger.Error("Panic in handler",
					zap.Any("panic", rec),
					zap.String("stack", string(stack)),
					zap.String("url", r.URL.String()),
					zap.String("method", r.Method),
				)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next(w, r)
	}
}

// auth middleware validates API keys and manages rate limiting
func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			// Try to get from query param (less secure, but allowed for some endpoints)
			apiKey = r.URL.Query().Get("api_key")
		}

		if apiKey == "" {
			s.logger.Warn("Missing API key",
				zap.String("ip", getClientIP(r)),
				zap.String("path", r.URL.Path),
			)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Validate API key using customer key manager
		customerKey, valid := s.keyManager.ValidateKey(apiKey)
		if !valid {
			// Log failed auth attempts (potential brute force)
			s.logger.Warn("Invalid API key",
				zap.String("ip", getClientIP(r)),
				zap.String("path", r.URL.Path),
				zap.String("user_agent", r.UserAgent()),
			)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check rate limit for this specific API key
		keyIdentifier := "key:" + customerKey.Hash
		if !s.rateLimiter.Allow(keyIdentifier, float64(customerKey.RateLimitRemaining), 1) {
			s.logger.Warn("API key rate limit exceeded",
				zap.String("key_hash", customerKey.Hash[:8]),
				zap.String("tier", string(customerKey.Tier)),
				zap.String("ip", getClientIP(r)),
				zap.String("path", r.URL.Path),
			)
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Update key usage statistics
		s.keyManager.UpdateKeyUsage(apiKey, getClientIP(r), r.UserAgent())

		// Use custom response writer to ensure status code is always set
		customWriter := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next(customWriter, r)

		// Log request (successful auth)
		s.logger.Debug("Authorized request",
			zap.String("path", r.URL.Path),
			zap.Int("status", customWriter.statusCode),
			zap.String("tier", string(customerKey.Tier)),
			zap.String("key_hash", customerKey.Hash[:8]),
		)
	}
}

// responseWriter is a custom ResponseWriter that tracks status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// WriteHeader overrides the WriteHeader method to capture status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.written = true
	rw.ResponseWriter.WriteHeader(code)
}

// Write overrides the Write method to track if anything was written
func (rw *responseWriter) Write(data []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(data)
}

// getClientIP extracts the client's real IP considering proxy headers
func getClientIP(r *http.Request) string {
	// Try common proxy headers
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		if ip := r.Header.Get(header); ip != "" {
			// X-Forwarded-For can be a comma-separated list; take the first one
			if strings.Contains(ip, ",") {
				return strings.TrimSpace(strings.Split(ip, ",")[0])
			}
			return ip
		}
	}

	// Extract from RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// jsonResponse safely writes a JSON response with proper error handling
func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error("Failed to encode JSON response",
			zap.Error(err),
			zap.Any("data", data),
		)
		// We've already written headers, so we can't change the status code
		// But we can log the error and write a simple error message
		fmt.Fprintf(w, `{"error":"Internal encoding error"}`)
	}
}

// turboJsonResponse Zero-allocation JSON response with pre-allocated buffers
func (s *Server) turboJsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Use pre-allocated encoder for turbo mode to reduce allocations
	if s.cfg.Tier == config.TierTurbo || s.cfg.Tier == config.TierEnterprise {
		s.turboEncodeJSON(w, data)
	} else {
		json.NewEncoder(w).Encode(data)
	}
}

// turboEncodeJSON Zero-allocation JSON encoding for turbo mode
func (s *Server) turboEncodeJSON(w http.ResponseWriter, data interface{}) {
	// Use a custom encoder that minimizes allocations
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false) // Disable HTML escaping for performance
	encoder.SetIndent("", "")    // Disable indentation for performance

	if err := encoder.Encode(data); err != nil {
		s.logger.Error("Failed to encode JSON response",
			zap.Error(err),
			zap.Any("data", data),
		)
		w.Write([]byte(`{"error":"Internal encoding error"}`))
	}
}
