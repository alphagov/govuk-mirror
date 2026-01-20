package page_fetcher_test

import (
	"fmt"
	"mirrorer/internal/page_fetcher"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type ReceivedRequests struct {
	requests map[string][]*http.Request
}

func (r *ReceivedRequests) Record(req *http.Request) {
	// reconstructing the full URL is necessary because
	// of the way http.Request populates the URL field on
	// the server side
	url := fmt.Sprintf("http://%s%s", req.Host, req.URL.String())
	if _, ok := r.requests[url]; !ok {
		r.requests[url] = []*http.Request{}
	}

	r.requests[url] = append(r.requests[url], req)
}

func (r *ReceivedRequests) Get(url string) []*http.Request {
	if reqs, ok := r.requests[url]; ok {
		return reqs
	}

	return []*http.Request{}
}

type TeardownFn = func()

func SetupTest(handler http.HandlerFunc) (*httptest.Server, *page_fetcher.PageFetcher, *ReceivedRequests, TeardownFn) {
	received := &ReceivedRequests{
		requests: make(map[string][]*http.Request),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Record(r)
		if handler != nil {
			handler.ServeHTTP(w, r)
		} else {
			w.WriteHeader(200)
		}
	}))
	fetcher, _ := page_fetcher.NewPageFetcher(server.URL)

	return server, fetcher, received, func() {
		server.Close()
	}
}
func TestNewPageFetcher(t *testing.T) {
	t.Run("throws an error if the baseUrl cannot be parsed as a URI", func(t *testing.T) {
		domain := "%%invalid$url"
		_, err := page_fetcher.NewPageFetcher(domain)
		assert.Error(t, err)
	})
}

func TestFetchLivePage(t *testing.T) {
	t.Run("adds the path to the baseUrl to form the request URL", func(t *testing.T) {
		server, fetcher, requests, teardown := SetupTest(nil)
		defer teardown()

		_, err := fetcher.FetchLivePage("/page")
		assert.NoError(t, err)

		assert.Len(t, requests.Get(server.URL+"/page"), 1)
	})

	t.Run("sets the backend-override header to 'never'", func(t *testing.T) {
		server, fetcher, requests, teardown := SetupTest(nil)
		defer teardown()

		_, err := fetcher.FetchLivePage("/page")
		assert.NoError(t, err)

		reqUrl := server.URL + "/page"
		assert.Len(t, requests.Get(reqUrl), 1)
		req := requests.Get(reqUrl)[0]

		header := req.Header.Get("Backend-Override")
		assert.Equal(t, "never", header)
	})

	t.Run("returns the body of the HTTP response", func(t *testing.T) {
		expectedBody := "Welcome to GOV.UK. "

		_, fetcher, _, teardown := SetupTest(func(w http.ResponseWriter, r *http.Request) {
			_, err := w.Write([]byte(expectedBody))
			assert.NoError(t, err)
		})
		defer teardown()

		body, err := fetcher.FetchLivePage("/page")
		assert.NoError(t, err)
		assert.Equal(t, expectedBody, body)
	})

	t.Run("returns an error if the request fails", func(t *testing.T) {
		_, fetcher, _, teardown := SetupTest(nil)

		// teardown to close the server before making the request
		// so that it errors
		teardown()

		_, err := fetcher.FetchLivePage("/page")
		assert.Error(t, err)
	})
}

func TestFetchMirrorPage(t *testing.T) {
	t.Run("adds the path to the baseUrl to form the request URL", func(t *testing.T) {
		server, fetcher, requests, teardown := SetupTest(nil)
		defer teardown()

		_, err := fetcher.FetchMirrorPage("/page")
		assert.NoError(t, err)

		assert.Len(t, requests.Get(server.URL+"/page"), 1)
	})

	t.Run("sets the backend-override header to 'mirrorS3'", func(t *testing.T) {
		server, fetcher, requests, teardown := SetupTest(nil)
		defer teardown()

		_, err := fetcher.FetchMirrorPage("/page")
		assert.NoError(t, err)

		reqUrl := server.URL + "/page"
		assert.Len(t, requests.Get(reqUrl), 1)
		req := requests.Get(reqUrl)[0]

		header := req.Header.Get("Backend-Override")
		assert.Equal(t, "mirrorS3", header)
	})

	t.Run("returns the body of the HTTP response", func(t *testing.T) {
		expectedBody := "Welcome to the mirror of GOV.UK."

		_, fetcher, _, teardown := SetupTest(func(w http.ResponseWriter, r *http.Request) {
			_, err := w.Write([]byte(expectedBody))
			assert.NoError(t, err)
		})
		defer teardown()

		body, err := fetcher.FetchMirrorPage("/page")
		assert.NoError(t, err)
		assert.Equal(t, expectedBody, body)
	})

	t.Run("returns an error if the request fails", func(t *testing.T) {
		_, fetcher, _, teardown := SetupTest(nil)

		// teardown to close the server before making the request
		// so that it errors
		teardown()

		_, err := fetcher.FetchMirrorPage("/page")
		assert.Error(t, err)
	})
}
