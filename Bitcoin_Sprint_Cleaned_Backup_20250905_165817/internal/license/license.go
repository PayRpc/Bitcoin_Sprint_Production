package license

import (
	"strings"
	"time"
)

func Validate(key string) bool {
	return key != ""
}

func EnforceTier(key string) (tier string, dailyLimit int) {
	if strings.HasPrefix(key, "free") {
		return "free", 500
	}
	if strings.HasPrefix(key, "pro") {
		return "pro", 50000
	}
	return "enterprise", -1
}

func Expired(key string) bool {
	return false
}

type LicenseInfo struct {
	Tier                  string
	Valid                 bool
	ExpiresAt             int64
	Features              []string
	RequestsPerHour       int
	ConcurrentConnections int
	DataRetentionDays     int
}

func GetInfo(key string) LicenseInfo {
	tier, dailyLimit := EnforceTier(key)
	valid := Validate(key)

	info := LicenseInfo{
		Tier:                  strings.ToUpper(tier),
		Valid:                 valid,
		ExpiresAt:             time.Now().AddDate(1, 0, 0).Unix(),
		RequestsPerHour:       dailyLimit / 24,
		ConcurrentConnections: 5,
		DataRetentionDays:     30,
	}

	switch strings.ToLower(tier) {
	case "free":
		info.Features = []string{"basic_api", "rate_limited"}
		info.ConcurrentConnections = 1
		info.DataRetentionDays = 7
	case "pro":
		info.Features = []string{"basic_api", "metrics", "analytics", "priority_support"}
		info.ConcurrentConnections = 5
		info.DataRetentionDays = 30
	case "enterprise":
		info.Features = []string{"basic_api", "metrics", "analytics", "predictive", "streaming", "admin_access", "enterprise_support", "custom_integration"}
		info.ConcurrentConnections = 50
		info.DataRetentionDays = 365
		info.RequestsPerHour = -1
	}

	return info
}
