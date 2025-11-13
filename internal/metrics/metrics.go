package metrics

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/rs/zerolog/log"
)

type Metrics struct {
	errorCounter prometheus.Counter
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		errorCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "crawler_errors_total",
			Help: "Total number of errors encountered by the crawler",
		}),
	}

	reg.MustRegister(m.errorCounter)

	return m
}

func CrawlerError(m *Metrics) {
	m.errorCounter.Inc()
}

func (m Metrics) ErrorCounter() prometheus.Counter {
	return m.errorCounter
}

func PushMetrics(wg *sync.WaitGroup, ctx context.Context, t time.Duration) {
	// Executes after ticker.Stop() as multiple defer use a stack
	defer wg.Done()

	ticker := time.NewTicker(t)

	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := push.New(os.Getenv("PROMETHEUS_PUSHGATEWAY_URL"), "mirror_metrics").Push()

			if err != nil {
				log.Error().Err(err).Msg("Error pushing metrics to Prometheus Pushgateway")
			}

		case <-ctx.Done():
			log.Info().Msg("PushMetrics goroutine is shutting down...")
			return
		}
	}
}
