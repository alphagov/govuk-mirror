package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"testing/synctest"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestIncrementErrorCounterMetric(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)
	HttpCrawlerError(m)
	HttpCrawlerError(m)
	HttpCrawlerError(m)

	assert.Equal(t, float64(3), testutil.ToFloat64(m.HttpErrorCounter()))
}

func TestIncrementDownloadErrorCounterMetric(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)
	DownloadCrawlerError(m)
	DownloadCrawlerError(m)
	DownloadCrawlerError(m)

	assert.Equal(t, float64(3), testutil.ToFloat64(m.DownloadErrorCounter()))
}

func TestIncrementDownloadCounterMetric(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)
	DownloadCounter(m)
	DownloadCounter(m)
	DownloadCounter(m)

	assert.Equal(t, float64(3), testutil.ToFloat64(m.DownloadCounter()))
}

func TestIncrementCrawledPagesCounterMetric(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)
	CrawledPagesCounter(m)
	CrawledPagesCounter(m)
	CrawledPagesCounter(m)

	assert.Equal(t, float64(3), testutil.ToFloat64(m.CrawledPagesCounter()))
}

func TestCrawlerDurationGaugeMetric(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reg := prometheus.NewRegistry()
		m := NewMetrics(reg)

		startTime := time.Now()

		// Pass 10 'fake' minutes
		time.Sleep(10 * time.Minute)
		CrawlerDuration(m, startTime)

		assert.Equal(t, float64(10), testutil.ToFloat64(m.CrawlerDuration()))
	})
}

func createTestServer(lastModified time.Time, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backend := r.Header.Get("Backend-Override")
		if backend == "backend1" || backend == "backend2" {
			if backend == "backend2" {
				lastModified = lastModified.AddDate(0, 0, -1)
			}

			w.Header().Set("Last-Modified", lastModified.Format(http.TimeFormat))
			w.WriteHeader(statusCode)
		} else {
			http.Error(w, "Backend-Override header not set to backend1 or backend2", http.StatusBadRequest)
		}
	}))
}

func TestFetchMirrorFreshnessMetric(t *testing.T) {
	timestamp := time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)
	ts := createTestServer(timestamp, http.StatusOK)
	defer ts.Close()

	freshness, err := fetchMirrorFreshnessMetric("backend1", ts.URL)
	assert.NoError(t, err)

	expectedFreshness := float64(timestamp.Unix())
	assert.Equal(t, expectedFreshness, freshness)
}

func TestFetchMirrorFreshnessMetric500StatusCode(t *testing.T) {
	timestamp := time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)
	ts := createTestServer(timestamp, http.StatusInternalServerError)
	defer ts.Close()

	_, err := fetchMirrorFreshnessMetric("backend1", ts.URL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request failed with status code")
}
