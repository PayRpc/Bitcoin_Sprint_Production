//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"
)

// Version information set by ldflags during build
var (
	Version = "dev"
	Commit  = "unknown"
)

type mockServer struct {
	cfg struct {
		Tier string
	}
	clock mockClock
}

type mockClock struct{}

func (m mockClock) Now() time.Time {
	return time.Now()
}

func (s *mockServer) versionHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"version":    Version,
		"build":      "enterprise",
		"build_time": Commit,
		"tier":       s.cfg.Tier,
		"turbo_mode": s.cfg.Tier == "turbo" || s.cfg.Tier == "Enterprise",
		"timestamp":  s.clock.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	server := &mockServer{}
	server.cfg.Tier = "Enterprise"

	req := httptest.NewRequest("GET", "/version", nil)
	w := httptest.NewRecorder()

	server.versionHandler(w, req)

	fmt.Println("Version endpoint response:")
	fmt.Println(w.Body.String())
}
