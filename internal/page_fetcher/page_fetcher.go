package page_fetcher

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// PageFetcher is used to retrieve pages from GOV.UK, either from the primary mirror or the live site
type PageFetcher struct {
	baseUrl *url.URL
	client  *http.Client
}

func NewPageFetcher(baseUrl string) (*PageFetcher, error) {
	base, err := url.Parse(baseUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %s", baseUrl)
	}

	return &PageFetcher{
		baseUrl: base,
		client:  &http.Client{},
	}, nil
}

func (pf *PageFetcher) FetchLivePage(path string) (string, error) {
	return pf.fetchPage(path, "never")
}

func (pf *PageFetcher) FetchMirrorPage(path string) (string, error) {
	return pf.fetchPage(path, "mirrorS3")
}

func (pf *PageFetcher) fetchPage(path string, backendHeaderValue string) (string, error) {
	reqUrl := pf.baseUrl.JoinPath(path)

	req, err := http.NewRequest("GET", reqUrl.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Backend-Override", backendHeaderValue)

	resp, err := pf.client.Do(req)
	if err != nil {
		return "", err
	}
	defer (func() {
		_ = resp.Body.Close()
	})()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
