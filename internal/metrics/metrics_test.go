package metrics

import (
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
