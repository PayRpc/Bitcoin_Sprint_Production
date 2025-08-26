package license

import (
	"crypto/ed25519"
	"encoding/hex"
	"strings"
	"time"
)

// Replace with your actual pubkey (from build or config)
var pubKey = ed25519.PublicKey{
	0x12, 0x34, 0x56, 0x78, // ...
}

// Validate license using Ed25519
func Validate(key string) bool {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return false
	}
	payload, sigHex := parts[0], parts[1]
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}
	return ed25519.Verify(pubKey, []byte(payload), sig)
}

// Enforce tier limits
func EnforceTier(key string) (tier string, dailyLimit int) {
	if strings.HasPrefix(key, "free") {
		return "free", 500
	}
	if strings.HasPrefix(key, "pro") {
		return "pro", 50000
	}
	return "enterprise", -1 // unlimited
}

// Dummy expiry parser (extend with JWT if needed)
func Expired(key string) bool {
	// Example: keys encoded like "<tier>-<expiry_unix>.<sig>"
	parts := strings.Split(key, "-")
	if len(parts) < 2 {
		return false
	}
	expUnix, err := time.ParseDuration(parts[1] + "s")
	if err != nil {
		return false
	}
	return time.Now().Unix() > int64(expUnix.Seconds())
}
