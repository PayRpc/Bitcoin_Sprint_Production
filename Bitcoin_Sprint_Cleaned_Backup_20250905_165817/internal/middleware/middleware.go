package middleware

import "net/http"

// Middleware is a placeholder type for middleware functions
type Middleware func(http.Handler) http.Handler

// Profiling returns a http.HandlerFunc that proxies to pprof when enabled
func Profiling(enabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !enabled {
			http.NotFound(w, r)
			return
		}
		// In real implementation this would proxy to net/http/pprof handlers
		http.NotFound(w, r)
	}
}

// SecurityHeadersHandler wraps an http.Handler to add security headers when enabled
func SecurityHeadersHandler(h http.Handler, enabled bool) http.Handler {
	if !enabled {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		h.ServeHTTP(w, r)
	})
}
