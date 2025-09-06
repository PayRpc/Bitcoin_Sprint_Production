package config

import "time"

// ExternalEndpoint represents an external API endpoint configuration
type ExternalEndpoint struct {
	URL      string        `json:"url"`
	Priority int           `json:"priority"`
	Timeout  time.Duration `json:"timeout"`
}
