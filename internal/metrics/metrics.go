package metrics

import (
	"context"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/rs/zerolog/log"
)

type Metrics struct {
	httpErrorCounter     prometheus.Counter
	downloadErrorCounter prometheus.Counter
	downloadCounter      prometheus.Counter
}

func NewMetrics(reg *prometheus.Registry) *Metrics {
	m := &Metrics{
		httpErrorCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "crawler_errors_total",
			Help: "Total number of HTTP errors encountered by the crawler",
		}),
		downloadErrorCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "download_errors_total",
			Help: "Total number of download errors encountered by the crawler",
		}),
		downloadCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "download_total",
			Help: "Total number of files downloaded by the crawler",
		}),
	}

	reg.MustRegister(m.httpErrorCounter)
	reg.MustRegister(m.downloadErrorCounter)
	reg.MustRegister(m.downloadCounter)

	return m
}

func HttpCrawlerError(m *Metrics) {
	m.httpErrorCounter.Inc()
}

func DownloadCounter(m *Metrics) {
	m.downloadCounter.Inc()
}

func DownloadCrawlerError(m *Metrics) {
	m.downloadErrorCounter.Inc()
}

func (m Metrics) HttpErrorCounter() prometheus.Counter {
	return m.httpErrorCounter
}

func (m Metrics) DownloadErrorCounter() prometheus.Counter {
	return m.downloadErrorCounter
}

func (m Metrics) DownloadCounter() prometheus.Counter {
	return m.downloadCounter
}

func PushMetrics(reg *prometheus.Registry, ctx context.Context, t time.Duration) {
	ticker := time.NewTicker(t)

	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := push.New(os.Getenv("PROMETHEUS_PUSHGATEWAY_URL"), "mirror_metrics").Gatherer(reg).Push()

			if err != nil {
				log.Error().Err(err).Msg("Error pushing metrics to Prometheus Pushgateway")
			}

		case <-ctx.Done():
			log.Info().Msg("PushMetrics goroutine is shutting down...")
			return
		}
	}
}
