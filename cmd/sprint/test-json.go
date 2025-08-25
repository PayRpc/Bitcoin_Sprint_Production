package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

func main() {
	// Test JSON with environment variables
	testJSON := `{"license_key": "${LICENSE_KEY}", "rpc_user": "${RPC_USER}"}`
	fmt.Printf("Original JSON: %s\n", testJSON)

	// Set some test environment variables
	os.Setenv("LICENSE_KEY", "test_license_123")
	// Leave RPC_USER unset to test empty replacement

	// Apply the same replacement logic as LoadConfig
	re := regexp.MustCompile(`\$\{([A-Z0-9_]+)\}`)
	resolved := re.ReplaceAllFunc([]byte(testJSON), func(b []byte) []byte {
		m := re.FindSubmatch(b)
		if len(m) < 2 {
			return b
		}
		name := string(m[1])
		val := os.Getenv(name)
		// Return the raw value (no JSON marshaling since the quotes are already in the template)
		return []byte(val)
	})

	fmt.Printf("Resolved JSON: %s\n", resolved)

	// Test if it parses
	var config map[string]interface{}
	if err := json.Unmarshal(resolved, &config); err != nil {
		fmt.Printf("Parse error: %v\n", err)
	} else {
		fmt.Printf("Parsed successfully: %+v\n", config)
	}
}
