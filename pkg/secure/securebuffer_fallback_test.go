//go:build !cgo
// +build !cgo

package secure

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"
)

func TestWithBytesRoundtrip(t *testing.T) {
	sb := NewSecureBuffer(17)
	defer sb.Free()
	data := []byte("supersecret123456")
	if !sb.Copy(data) {
		t.Fatal("Copy failed")
	}
	err := sb.WithBytes(func(b []byte) error {
		if len(b) != len(data) {
			t.Fatalf("unexpected length: %d", len(b))
		}
		if string(b[:len(data)]) != string(data) {
			t.Fatalf("data mismatch: %q vs %q", b[:len(data)], data)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHMACHex(t *testing.T) {
	sb := NewSecureBuffer(32)
	defer sb.Free()

	key := []byte("test-key")
	sb.Copy(key)

	data := []byte("test data")
	result := sb.HMACHex(data)

	// Compare with standard library
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	expected := hex.EncodeToString(mac.Sum(nil))

	if result != expected {
		t.Fatalf("HMAC mismatch: got %s, expected %s", result, expected)
	}
}

func TestSetBasicAuthHeader(t *testing.T) {
	sb := NewSecureBuffer(32)
	defer sb.Free()

	password := "secret123"
	sb.Copy([]byte(password))

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	SetBasicAuthHeader(req, "testuser", sb)

	auth := req.Header.Get("Authorization")
	if auth == "" {
		t.Fatal("Authorization header not set")
	}

	// Should start with "Basic "
	if len(auth) < 6 || auth[:6] != "Basic " {
		t.Fatalf("Invalid auth header format: %s", auth)
	}
}

func TestSecureBufferBasics(t *testing.T) {
	sb := NewSecureBuffer(10)
	if sb == nil {
		t.Fatal("NewSecureBuffer returned nil")
	}
	defer sb.Free()

	data := []byte("hello")
	if !sb.Copy(data) {
		t.Fatal("Copy failed")
	}

	result := sb.String()
	if result != "hello" {
		t.Fatalf("String() returned %q, expected %q", result, "hello")
	}

	// Test Data() method
	rawData := sb.Data()
	if string(rawData[:5]) != "hello" {
		t.Fatalf("Data() returned invalid data")
	}
}
