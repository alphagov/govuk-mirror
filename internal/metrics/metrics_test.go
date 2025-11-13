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
	CrawlerError(m)
	CrawlerError(m)
	CrawlerError(m)

	assert.Equal(t, float64(3), testutil.ToFloat64(m.ErrorCounter()))
}
