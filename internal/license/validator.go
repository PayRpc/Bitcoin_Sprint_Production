package license

import (
	"fmt"
	"time"
)

// LicenseInfo contains detailed information about a validated license
type LicenseInfo struct {
	Valid     bool      `json:"valid"`
	IssuedTo  string    `json:"issued_to"`
	ExpiresAt time.Time `json:"expires_at"`
	Plan      string    `json:"plan"`
	Message   string    `json:"message"`
	Features  []string  `json:"features"`
}

// Validator validates license keys
type Validator struct {
	licenseKey string
}

// NewValidator creates a new license validator
func NewValidator(licenseKey string) *Validator {
	return &Validator{
		licenseKey: licenseKey,
	}
}

// ValidateWithDetails performs a comprehensive license validation
func (v *Validator) ValidateWithDetails(nodeID string) (LicenseInfo, error) {
	// Simple implementation for now
	if v.licenseKey == "" {
		return LicenseInfo{
			Valid:   false,
			Message: "No license key provided",
		}, nil
	}

	// In a real implementation, this would verify the license with a server
	// or perform cryptographic validation

	// For now, just validate that the key follows a basic format
	if len(v.licenseKey) < 16 {
		return LicenseInfo{
			Valid:   false,
			Message: "Invalid license key format",
		}, nil
	}

	// Mock license validation for development
	return LicenseInfo{
		Valid:     true,
		IssuedTo:  "Bitcoin Sprint User",
		ExpiresAt: time.Now().AddDate(1, 0, 0), // 1 year from now
		Plan:      "Enterprise",
		Message:   "License valid",
		Features:  []string{"p2p", "memory_channel", "kernel_bypass", "hardware_monitoring"},
	}, nil
}

// Validate performs a simple license validation
func (v *Validator) Validate() (bool, error) {
	info, err := v.ValidateWithDetails("")
	return info.Valid, err
}
