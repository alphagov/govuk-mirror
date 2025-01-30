package client

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/gocolly/colly/v2"
)

type DisallowedURLError struct{}

func (e DisallowedURLError) Error() string {
	return "Not following redirect because it's not allowed"
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
			return DisallowedURLError{}
		}

		return nil
	}

	return client
}

func isRequestAllowed(c *colly.Collector, parsedURL *url.URL) bool {
	u := []byte(parsedURL.String())

	for _, r := range c.URLFilters {
		if !r.Match(u) {
			return false
		}
	}

	for _, r := range c.DisallowedURLFilters {
		if r.Match(u) {
			return false
		}
	}

	if len(c.AllowedDomains) == 0 {
		return true
	}
	for _, d := range c.AllowedDomains {
		if d == parsedURL.Hostname() {
			return true
		}
	}
	return false
}
