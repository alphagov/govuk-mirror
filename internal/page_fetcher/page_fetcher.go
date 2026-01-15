package page_fetcher

import (
	"io"
	"net/http"
)

// PageFetcher is used to retrieve pages from GOV.UK, either from the primary mirror or the live site
type PageFetcher struct {
	client *http.Client
}

func NewPageFetcher() *PageFetcher {
	return &PageFetcher{
		client: &http.Client{},
	}
}

func (pf *PageFetcher) FetchLivePage(url string) (string, error) {
	return pf.fetchPage(url, "never")
}

func (pf *PageFetcher) FetchMirrorPage(path string) (string, error) {
	return pf.fetchPage(path, "mirrorS3")
}

func (pf *PageFetcher) fetchPage(url string, backendHeaderValue string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
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
