package config

import (
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

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
				UserAgent:             "govuk-mirror-bot",
				Concurrency:           10,
				SkipValidation:        false,
				MetricRefreshInterval: 10 * time.Second,
				Async:                 true,
				MirrorS3BucketName:    "",
				Backends:              nil,
				PushGatewayUrl:        "",
			},
		},
		{
			name: "env vars",
			envVars: map[string]string{
				"SITE":                       "example.com",
				"ALLOWED_DOMAINS":            "example.com,foo.bar",
				"USER_AGENT":                 "custom-agent",
				"HEADERS":                    "Test-Header:Test-Value",
				"CONCURRENCY":                "20",
				"URL_RULES":                  "rule1,rule2",
				"DISALLOWED_URL_RULES":       "rule3,rule4",
				"SKIP_VALIDATION":            "true",
				"METRIC_REFRESH_INTERVAL":    "10s",
				"ASYNC":                      "true",
				"S3_BUCKET_NAME":             "s3-bucket-name",
				"PROMETHEUS_PUSHGATEWAY_URL": "http://pushgateway.test",
			},
			expected: &Config{
				Site:           "example.com",
				AllowedDomains: []string{"example.com", "foo.bar"},
				UserAgent:      "custom-agent",
				Headers: map[string]string{
					"Test-Header": "Test-Value",
				},
				Concurrency: 20,
				URLFilters: []*regexp.Regexp{
					regexp.MustCompile("rule1"),
					regexp.MustCompile("rule2"),
				},
				DisallowedURLFilters: []*regexp.Regexp{
					regexp.MustCompile("rule3"),
					regexp.MustCompile("rule4"),
				},
				SkipValidation:        true,
				MetricRefreshInterval: 10 * time.Second,
				Async:                 true,
				MirrorS3BucketName:    "s3-bucket-name",
				Backends:              []string{"backend1", "backend2"},
				PushGatewayUrl:        "http://pushgateway.test",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range test.envVars {
				_ = os.Setenv(k, v)
				defer func() {
					if err := os.Unsetenv(k); err != nil {
						fmt.Println("Error when unsetting:", err)
					}
				}()
			}

			cfg, err := NewConfig()
			assert.NoError(t, err)
			assert.Equal(t, test.expected, cfg)
		})
	}
}
