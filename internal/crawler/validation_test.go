package crawler

import (
	"mirrorer/internal/config"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestValidateCrawlerConfig ensures that the configured domains are accessible
// before starting a crawl, preventing runtime failures from inaccessible domains
func TestValidateCrawlerConfig(t *testing.T) {
	tests := []struct {
		name           string
		site           string
		allowedDomains []string
		expectError    bool
		description    string
		s3BucketName   string
	}{
		{
			name:           "accessible domain",
			site:           "https://www.gov.uk",
			allowedDomains: []string{"www.gov.uk"},
			expectError:    false,
			description:    "Should pass with accessible domain",
			s3BucketName:   "s3-bucket-name",
		},
		{
			name:           "inaccessible domain",
			site:           "https://definitely-does-not-exist.example.com",
			allowedDomains: []string{"definitely-does-not-exist.example.com"},
			expectError:    true,
			description:    "Should fail with inaccessible domain",
			s3BucketName:   "s3-bucket-name",
		},
		{
			name:           "mixed domains",
			site:           "https://www.gov.uk",
			allowedDomains: []string{"www.gov.uk", "definitely-does-not-exist.example.com"},
			expectError:    true,
			description:    "Should fail if any allowed domain is inaccessible",
			s3BucketName:   "s3-bucket-name",
		},
		{
			name:           "empty site",
			site:           "",
			allowedDomains: []string{"www.gov.uk"},
			expectError:    false,
			description:    "Should pass with empty site if allowed domains are accessible",
			s3BucketName:   "s3-bucket-name",
		},
		{
			name:           "empty s3 bucket name",
			site:           "https://www.gov.uk",
			allowedDomains: []string{"www.gov.uk"},
			expectError:    true,
			description:    "Should fail because the S3 bucket name is empty",
			s3BucketName:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Site:               tt.site,
				AllowedDomains:     tt.allowedDomains,
				UserAgent:          "test-agent",
				Concurrency:        1,
				MirrorS3BucketName: tt.s3BucketName,
			}

			err := ValidateCrawlerConfig(cfg, 5*time.Second)

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

func TestDomainNotAccessibleError(t *testing.T) {
	err := &DomainNotAccessibleError{Domain: "definitely-does-not-exist.example.com"}
	expectedMsg := "domain not accessible: definitely-does-not-exist.example.com"
	assert.Equal(t, expectedMsg, err.Error())
}
