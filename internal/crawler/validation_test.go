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
	}{
		{
			name:           "accessible domain",
			site:           "https://www.gov.uk",
			allowedDomains: []string{"www.gov.uk"},
			expectError:    false,
			description:    "Should pass with accessible domain",
		},
		{
			name:           "inaccessible domain",
			site:           "https://definitely-does-not-exist.example.com",
			allowedDomains: []string{"definitely-does-not-exist.example.com"},
			expectError:    true,
			description:    "Should fail with inaccessible domain",
		},
		{
			name:           "mixed domains",
			site:           "https://www.gov.uk",
			allowedDomains: []string{"www.gov.uk", "definitely-does-not-exist.example.com"},
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
	err := &DomainNotAccessibleError{Domain: "definitely-does-not-exist.example.com"}
	expectedMsg := "domain not accessible: definitely-does-not-exist.example.com"
	assert.Equal(t, expectedMsg, err.Error())
}
