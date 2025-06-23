package crawler

import (
	"context"
	"fmt"
	"mirrorer/internal/config"
	"net/http"
	"net/url"
	"time"
)

// ValidateCrawlerConfig checks if the configured domains are accessible
// Call this before starting a crawl to catch configuration issues early
func ValidateCrawlerConfig(cfg *config.Config, timeout time.Duration) error {
	// Check main site URL
	if cfg.Site != "" {
		if !isDomainAccessibleWithConfig(cfg.Site, cfg, timeout) {
			return &DomainNotAccessibleError{Domain: cfg.Site}
		}
	}

	// Check all allowed domains
	for _, domain := range cfg.AllowedDomains {
		testURL := "https://" + domain
		if !isDomainAccessibleWithConfig(testURL, cfg, timeout) {
			return &DomainNotAccessibleError{Domain: domain}
		}
	}

	return nil
}

// DomainNotAccessibleError indicates a domain is not accessible
type DomainNotAccessibleError struct {
	Domain string
}

func (e *DomainNotAccessibleError) Error() string {
	return fmt.Sprintf("domain not accessible: %s (hint: www-origin.publishing.service.gov.uk is not externally accessible, use www.gov.uk instead)", e.Domain)
}

// isDomainAccessibleWithConfig checks if a domain responds using the same config as Colly
func isDomainAccessibleWithConfig(testURL string, cfg *config.Config, timeout time.Duration) bool {
	parsedURL, err := url.Parse(testURL)
	if err != nil {
		return false
	}

	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow redirects but limit to 5
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Use GET instead of HEAD to match Colly behavior
	req, err := http.NewRequestWithContext(ctx, "GET", parsedURL.String(), nil)
	if err != nil {
		return false
	}

	// Set User-Agent to match Colly configuration
	req.Header.Set("User-Agent", cfg.UserAgent)

	// Add custom headers from configuration
	for key, value := range cfg.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Consider 2xx and 3xx status codes as accessible
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}


