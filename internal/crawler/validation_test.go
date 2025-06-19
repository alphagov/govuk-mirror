package crawler

import (
	"testing"
	"time"

	"mirrorer/internal/config"

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
	}{
		{
			name:           "accessible domain",
			site:           "https://www.gov.uk",
			allowedDomains: []string{"www.gov.uk"},
			expectError:    false,
			description:    "Should pass with accessible domain",
		},
		{
			name:           "inaccessible origin domain",
			site:           "https://www-origin.publishing.service.gov.uk",
			allowedDomains: []string{"www-origin.publishing.service.gov.uk"},
			expectError:    true,
			description:    "Should fail with inaccessible origin domain",
		},
		{
			name:           "mixed domains",
			site:           "https://www.gov.uk",
			allowedDomains: []string{"www.gov.uk", "www-origin.publishing.service.gov.uk"},
			expectError:    true,
			description:    "Should fail if any allowed domain is inaccessible",
		},
		{
			name:           "empty site",
			site:           "",
			allowedDomains: []string{"www.gov.uk"},
			expectError:    false,
			description:    "Should pass with empty site if allowed domains are accessible",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Site:           tt.site,
				AllowedDomains: tt.allowedDomains,
				UserAgent:      "test-agent",
				Concurrency:    1,
			}

			err := ValidateCrawlerConfig(cfg, 5*time.Second)

			if tt.expectError {
				assert.Error(t, err, tt.description)
				// Check that the error message is helpful
				if err != nil {
					assert.Contains(t, err.Error(), "not accessible", "Error should indicate domain is not accessible")
				}
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

func TestDomainNotAccessibleError(t *testing.T) {
	err := &DomainNotAccessibleError{Domain: "www-origin.publishing.service.gov.uk"}
	expectedMsg := "domain not accessible: www-origin.publishing.service.gov.uk (hint: www-origin.publishing.service.gov.uk is not externally accessible, use www.gov.uk instead)"
	assert.Equal(t, expectedMsg, err.Error())
}