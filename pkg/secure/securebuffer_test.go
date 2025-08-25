//go:build cgo
// +build cgo

package secure

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"
)

func TestWithBytesRoundtrip(t *testing.T) {
	sb := NewSecureBuffer(16)
	defer sb.Free()
	data := []byte("supersecret123456")
	if !sb.Copy(data) {
		t.Fatal("Copy failed")
	}
	err := sb.WithBytes(func(b []byte) error {
		if len(b) != len(data) {
			t.Fatalf("unexpected length: %d", len(b))
		}
		for i := range b {
			if b[i] != data[i] {
				t.Fatalf("mismatch at %d", i)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithBytes error: %v", err)
	}
}

func TestHMACHexMatchesGo(t *testing.T) {
	key := []byte("key-abc-123")
	msg := []byte("message to sign")
	sb := NewSecureBuffer(len(key))
	defer sb.Free()
	if !sb.Copy(key) {
		t.Fatal("copy failed")
	}

	got := sb.HMACHex(msg)

	mac := hmac.New(sha256.New, key)
	mac.Write(msg)
	want := hex.EncodeToString(mac.Sum(nil))

	if got != want {
		t.Fatalf("HMACHex mismatch got=%s want=%s", got, want)
	}
}

func TestSetBasicAuthHeader(t *testing.T) {
	sb := NewSecureBuffer(8)
	defer sb.Free()
	if !sb.Copy([]byte("pwd12345")) {
		t.Fatal("copy failed")
	}
	req, _ := http.NewRequest("GET", "https://example.test/", nil)
	SetBasicAuthHeader(req, "user1", sb)
	if req.Header.Get("Authorization") == "" {
		t.Fatal("Authorization header not set")
	}
}
