package metrics

import (
	"testing"

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
