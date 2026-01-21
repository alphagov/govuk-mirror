package page_fetcher

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

//counterfeiter:generate -o ./fakes/ . PageFetcherInterface
type PageFetcherInterface interface {
	FetchLivePage(path string) (*Page, error)
	FetchMirrorPage(path string) (*Page, error)
}

type Page struct {
	Body        string
	ContentType string
}

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

func (pf *PageFetcher) FetchLivePage(path string) (*Page, error) {
	return pf.fetchPage(path, "never")
}

func (pf *PageFetcher) FetchMirrorPage(path string) (*Page, error) {
	return pf.fetchPage(path, "mirrorS3")
}

func (pf *PageFetcher) fetchPage(path string, backendHeaderValue string) (*Page, error) {
	reqUrl := pf.baseUrl.JoinPath(path)

	req, err := http.NewRequest("GET", reqUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Backend-Override", backendHeaderValue)

	resp, err := pf.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer (func() {
		_ = resp.Body.Close()
	})()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	page := &Page{
		Body:        string(body),
		ContentType: resp.Header.Get("Content-Type"),
	}

	return page, nil
}
