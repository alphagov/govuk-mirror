package metrics

import (
	"mirrorer/internal/config"
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

func TestMetricsAreCorrectlyPrefixed(t *testing.T) {
	reg := prometheus.NewRegistry()
	_ = NewMetrics(reg)

	metricValues, err := reg.Gather()
	assert.NoError(t, err)

	for _, metric := range metricValues {
		assert.NotNil(t, metric.Name)
		assert.Regexp(t, "govuk_mirror_.*", *metric.Name)
	}
}

func setup() (*Metrics, *config.Config, *prometheus.Registry) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)
	cfg := &config.Config{
		MirrorAvailabilityUrl:      "http://mirror.test/availability",
		MirrorBackends:             []string{"backend1", "backend2"},
		StatusCheckRefreshInterval: 1 * time.Hour,
	}
	return m, cfg, reg
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

func createTestPushGateway() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
}

func TestFetchMirrorAvailabilityMetric200StatusCode(t *testing.T) {
	timestamp := time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)
	ts := createTestServer(timestamp, http.StatusOK)
	defer ts.Close()

	responseCode, err := fetchMirrorAvailabilityMetric("backend1", ts.URL)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, responseCode)
}

func TestFetchMirrorAvailabilityMetric500StatusCode(t *testing.T) {
	timestamp := time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)
	ts := createTestServer(timestamp, http.StatusInternalServerError)
	defer ts.Close()

	responseCode, err := fetchMirrorAvailabilityMetric("backend1", ts.URL)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, responseCode)
}

func TestUpdateMetrics(t *testing.T) {
	m, cfg, reg := setup()

	timestamp := time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)
	ts := createTestServer(timestamp, http.StatusOK)
	defer ts.Close()

	pushGateway := createTestPushGateway()
	defer pushGateway.Close()

	cfg.MirrorAvailabilityUrl = ts.URL
	cfg.MetricRefreshInterval = 1 * time.Second
	cfg.PushGatewayUrl = pushGateway.URL

	go UpdateAndPushMirrorResponseStatusCode(m, cfg, reg)

	time.Sleep(2 * time.Second)

	assert.Equal(t, 2, testutil.CollectAndCount(m.mirrorResponseStatusCode))
	assert.Equal(t, float64(http.StatusOK), testutil.ToFloat64(m.mirrorResponseStatusCode.WithLabelValues("backend1")))
	assert.Equal(t, float64(http.StatusOK), testutil.ToFloat64(m.mirrorResponseStatusCode.WithLabelValues("backend2")))
}
