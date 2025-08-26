package sprintclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetStatus_PrefersSprint(t *testing.T) {
	// Sprint server
	sprint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/status" {
			json.NewEncoder(w).Encode(map[string]any{"source": "sprint", "ok": true})
			return
		}
		w.WriteHeader(404)
	}))
	defer sprint.Close()

	// Core server - shouldn't be called
	coreCalled := false
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		coreCalled = true
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"chain": "main"}, "error": nil, "id": "1"})
	}))
	defer core.Close()

	c := NewSprintClient(sprint.URL, core.URL, "user", "pass")
	res, err := c.GetStatus()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res["source"] != "sprint" {
		t.Fatalf("expected sprint source, got %v", res)
	}
	if coreCalled {
		t.Fatalf("core should not have been called when sprint is available")
	}
}

func TestGetStatus_FallbackToCore(t *testing.T) {
	// Sprint server returns 500
	sprint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("down"))
	}))
	defer sprint.Close()

	// Core server - should be called
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// validate basic auth
		user, pass, ok := r.BasicAuth()
		if !ok || user != "user" || pass != "pass" {
			w.WriteHeader(401)
			return
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"chain": "main"}, "error": nil, "id": "1"})
	}))
	defer core.Close()

	c := NewSprintClient(sprint.URL, core.URL, "user", "pass")
	res, err := c.GetStatus()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// core RPC returns a JSON-RPC envelope with result
	if res["source"] != "core" {
		t.Fatalf("expected source core, got %v", res)
	}
	if _, ok := res["data"]; !ok {
		t.Fatalf("expected data field in normalized envelope, got %v", res)
	}
}
