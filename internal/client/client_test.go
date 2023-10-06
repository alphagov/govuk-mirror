package client

import (
	"net/http"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/stretchr/testify/assert"
)

func TestDisallowedURLError(t *testing.T) {
	err := &DisallowedURLError{Url: "http://example.com"}
	assert.Equal(t, "Not following redirect to http://example.com because its not allowed", err.Error())
}

func TestNewClient(t *testing.T) {
	c := colly.NewCollector()
	c.AllowedDomains = []string{"allowed.com"}

	called := false
	redirectHandler := func(req *http.Request, via []*http.Request) error {
		called = true
		return nil
	}

	client := NewClient(c, redirectHandler)
	assert.NotNil(t, client)
	assert.NotNil(t, client.Jar)
	assert.Equal(t, 60*time.Second, client.Timeout)

	// Create dummy requests to trigger the redirect handler
	req, _ := http.NewRequest("GET", "http://allowed.com", nil)
	redirectReq, _ := http.NewRequest("GET", "http://disallowed.com/redirect", nil)

	// Ensure redirect handler is called before redirect not followed
	err := client.CheckRedirect(redirectReq, []*http.Request{req})
	assert.IsType(t, &DisallowedURLError{}, err)
	assert.True(t, called, "redirectHandler was not called")
}

func TestIsRequestAllowedTableDriven(t *testing.T) {
	tests := []struct {
		name            string
		disallowedURLs  []*regexp.Regexp
		allowedDomains  []string
		url             string
		expectedAllowed bool
	}{
		{
			name:            "no restrictions",
			url:             "http://example.com",
			expectedAllowed: true,
		},
		{
			name:            "disallowed URL filter",
			disallowedURLs:  []*regexp.Regexp{regexp.MustCompile("http://example.com")},
			url:             "http://example.com",
			expectedAllowed: false,
		},
		{
			name:            "allowed domain",
			allowedDomains:  []string{"example.com"},
			url:             "http://example.com",
			expectedAllowed: true,
		},
		{
			name:            "different domain",
			allowedDomains:  []string{"example.com"},
			url:             "http://notallowed.com",
			expectedAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := colly.NewCollector()
			c.DisallowedURLFilters = tt.disallowedURLs
			c.AllowedDomains = tt.allowedDomains
			parsedURL, _ := url.Parse(tt.url)
			assert.Equal(t, tt.expectedAllowed, isRequestAllowed(c, parsedURL))
		})
	}
}
