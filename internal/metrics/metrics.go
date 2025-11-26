package metrics

import (
	"context"
	"mirrorer/internal/config"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/rs/zerolog/log"
)

type Metrics struct {
	httpErrorCounter          prometheus.Counter
	downloadErrorCounter      prometheus.Counter
	downloadCounter           prometheus.Counter
	crawledPagesCounter       prometheus.Counter
	crawlerDuration           prometheus.Gauge
	fileUploadCounter         prometheus.Counter
	fileUploadFailuresCounter prometheus.Counter
	mirrorLastUpdatedGauge    prometheus.Gauge
}

func NewMetrics(reg *prometheus.Registry) *Metrics {
	m := &Metrics{
		crawledPagesCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "crawled_pages_total",
			Help: "Total number of pages successfully crawled",
		}),
		httpErrorCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "crawler_errors_total",
			Help: "Total number of HTTP errors encountered by the crawler",
		}),
		downloadErrorCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "crawler_pages_download_errors_total",
			Help: "Total number of download errors encountered by the crawler",
		}),
		downloadCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "crawler_pages_downloaded_total",
			Help: "Total number of files downloaded by the crawler",
		}),
		crawlerDuration: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "crawler_duration_minutes",
			Help: "Duration of crawler in minutes",
		}),
		fileUploadCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "files_uploaded_total",
			Help: "Total number of files the crawler has uploaded to the mirror",
		}),
		fileUploadFailuresCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "file_upload_failures_total",
			Help: "Total number of upload failures encounterd by the crawler",
		}),
		mirrorLastUpdatedGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "govuk_mirror_last_updated_time",
			Help: "Last time the mirror was updated",
		}),
	}

	reg.MustRegister(m.httpErrorCounter)
	reg.MustRegister(m.downloadErrorCounter)
	reg.MustRegister(m.downloadCounter)
	reg.MustRegister(m.crawledPagesCounter)
	reg.MustRegister(m.crawlerDuration)
	reg.MustRegister(m.fileUploadCounter)
	reg.MustRegister(m.fileUploadFailuresCounter)
	reg.MustRegister(m.mirrorLastUpdatedGauge)

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

func CrawledPagesCounter(m *Metrics) {
	m.crawledPagesCounter.Inc()
}
func FileUploaded(m *Metrics) {
	m.fileUploadCounter.Inc()
}

func FileUploadFailed(m *Metrics) {
	m.fileUploadFailuresCounter.Inc()
}

func crawlerDuration(m *Metrics, t time.Time) {
	m.crawlerDuration.Set(time.Since(t).Minutes())
}

func mirrorLastUpdatedGauge(m *Metrics, last_updated_time float64) {
	m.mirrorLastUpdatedGauge.Set(last_updated_time)
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

func (m Metrics) CrawledPagesCounter() prometheus.Counter {
	return m.crawledPagesCounter
}

func (m Metrics) CrawlerDuration() prometheus.Gauge {
	return m.crawlerDuration
}

func (m Metrics) FileUploadCounter() prometheus.Counter {
	return m.fileUploadCounter
}

func (m Metrics) FileUploadFailuresCounter() prometheus.Counter {
	return m.fileUploadFailuresCounter
}

func (m Metrics) MirrorLastUpdatedGauge() prometheus.Gauge {
	return m.mirrorLastUpdatedGauge
}

func UpdateAndPushEndJobMetrics(m *Metrics, startTime time.Time, cfg *config.Config, reg *prometheus.Registry) {
	crawlerDuration(m, startTime)
	timeNow := float64(time.Now().Unix())
	mirrorLastUpdatedGauge(m, timeNow)

	err := push.New(cfg.PushGatewayUrl, "mirror_metrics").Gatherer(reg).Push()
	if err != nil {
		log.Error().Err(err).Msg("Error pushing metrics to Prometheus Pushgateway")
	}
}

func PushMetrics(reg *prometheus.Registry, ctx context.Context, cfg *config.Config) {
	ticker := time.NewTicker(cfg.MetricRefreshInterval)

	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := push.New(cfg.PushGatewayUrl, "mirror_metrics").Gatherer(reg).Push()

			if err != nil {
				log.Error().Err(err).Msg("Error pushing metrics to Prometheus Pushgateway")
			}

		case <-ctx.Done():
			err := push.New(os.Getenv("PROMETHEUS_PUSHGATEWAY_URL"), "mirror_metrics").Gatherer(reg).Push()

			if err != nil {
				log.Error().Err(err).Msg("Error pushing metrics to Prometheus Pushgateway")
			}

			log.Info().Msg("PushMetrics goroutine is shutting down...")

			return
		}
	}
}
