package client

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/gocolly/colly/v2"
)

type DisallowedURLError struct {
	Url string
}

func (r *DisallowedURLError) Error() string {
	return fmt.Sprintf("Not following redirect to %s because its not allowed", r.Url)
}

func NewClient(c *colly.Collector, redirectHandler func(*http.Request, []*http.Request) error) *http.Client {
	jar, _ := cookiejar.New(nil)

	client := &http.Client{
		Jar:     jar,
		Timeout: 60 * time.Second,
	}

	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		err := redirectHandler(req, via)
		if err != nil {
			return err
		}

		if !isRequestAllowed(c, req.URL) {
			return &DisallowedURLError{
				Url: req.URL.String(),
			}
		}

		return nil
	}

	return client
}

func isRequestAllowed(c *colly.Collector, parsedURL *url.URL) bool {
	u := []byte(parsedURL.String())

	for _, r := range c.DisallowedURLFilters {
		if r.Match(u) {
			return false
		}
	}

	if c.AllowedDomains == nil || len(c.AllowedDomains) == 0 {
		return true
	}
	for _, d := range c.AllowedDomains {
		if d == parsedURL.Hostname() {
			return true
		}
	}
	return false
}
