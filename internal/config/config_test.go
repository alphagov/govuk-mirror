package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected *Config
	}{
		{
			name: "defaults",
			expected: &Config{
				UserAgent:   "govukbot",
				Concurrency: 10,
			},
		},
		{
			name: "env vars",
			envVars: map[string]string{
				"SITE":            "example.com",
				"ALLOWED_DOMAINS": "example.com",
				"USER_AGENT":      "custom-agent",
				"HEADERS":         "Test-Header:Test-Value",
				"CONCURRENCY":     "20",
			},
			expected: &Config{
				Site:           "example.com",
				AllowedDomains: "example.com",
				UserAgent:      "custom-agent",
				Headers: map[string]string{
					"Test-Header": "Test-Value",
				},
				Concurrency: 20,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range test.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k) // Clean up
			}

			cfg, err := NewConfig()
			assert.NoError(t, err)
			assert.Equal(t, test.expected, cfg)
		})
	}
}
