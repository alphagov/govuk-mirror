package metrics

import (
	"context"
	"mirrorer/internal/config"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/rs/zerolog/log"
)

var httpClient = &http.Client{}

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
			Name: "govuk_mirror_crawled_pages_total",
			Help: "Total number of pages successfully crawled",
		}),
		httpErrorCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "govuk_mirror_crawler_errors_total",
			Help: "Total number of HTTP errors encountered by the crawler",
		}),
		downloadErrorCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "govuk_mirror_crawler_pages_download_errors_total",
			Help: "Total number of download errors encountered by the crawler",
		}),
		downloadCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "govuk_mirror_crawler_pages_downloaded_total",
			Help: "Total number of files downloaded by the crawler",
		}),
		crawlerDuration: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "govuk_mirror_crawler_duration_minutes",
			Help: "Duration of crawler in minutes",
		}),
		fileUploadCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "govuk_mirror_crawler_files_uploaded_total",
			Help: "Total number of files the crawler has uploaded to the mirror",
		}),
		fileUploadFailuresCounter: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "govuk_mirror_crawler_file_upload_failures_total",
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

	return m
}

type ResponseMetrics struct {
	mirrorResponseStatusCode *prometheus.GaugeVec
}

func NewResponseMetrics(reg *prometheus.Registry) *ResponseMetrics {
	m := &ResponseMetrics{
		mirrorResponseStatusCode: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "govuk_mirror_response_status_code",
			Help: "Response status code for the MIRROR_AVAILABILITY_URL probe",
		}, []string{"backend"}),
	}

	reg.MustRegister(m.mirrorResponseStatusCode)

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

func CrawlerDuration(m *Metrics, t time.Time) {
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

func (m ResponseMetrics) MirrorResponseStatusCode() prometheus.GaugeVec {
	return *m.mirrorResponseStatusCode
}

func UpdateEndJobMetrics(m *Metrics, startTime time.Time, cfg *config.Config) {
	CrawlerDuration(m, startTime)
	timeNow := float64(time.Now().Unix())
	mirrorLastUpdatedGauge(m, timeNow)
}

func fetchMirrorAvailabilityMetric(backend string, url string) (int, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error().Str("metric", "govuk_mirror_response_status_code").Str("backend", backend).Err(err).Msg("Failed to form a HTTP GET request")
		return 0, err
	}
	req.Header.Set("Backend-Override", backend)

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Error().Str("metric", "govuk_mirror_response_status_code").Str("backend", backend).Err(err).Msg("Failed to get a HTTP response")
		return 0, err
	}

	return resp.StatusCode, nil
}

func updateMirrorResponseStatusCode(m *ResponseMetrics, url string, backend string) error {
	statusCode, err := fetchMirrorAvailabilityMetric(backend, url)
	if err != nil {
		log.Error().Str("metric", "govuk_mirror_response_status_code").Str("backend", backend).Err(err).Msg("Failed to get a HTTP status code")
		return err
	}

	m.mirrorResponseStatusCode.With(prometheus.Labels{"backend": backend}).Set(float64(statusCode))
	return nil
}

func UpdateMirrorResponseStatusCode(m *ResponseMetrics, cfg *config.Config) {
	for _, backend := range cfg.MirrorBackends {
		err := updateMirrorResponseStatusCode(m, cfg.MirrorAvailabilityUrl, backend)
		if err != nil {
			log.Error().Str("metric", "govuk_mirror_response_status_code").Str("backend", backend).Err(err).Msg("Error updating metrics")
		}
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
			err := push.New(cfg.PushGatewayUrl, "mirror_metrics").Gatherer(reg).Push()

			if err != nil {
				log.Error().Err(err).Msg("Error pushing metrics to Prometheus Pushgateway")
			}

			log.Info().Msg("PushMetrics goroutine is shutting down...")

			return
		}
	}
}
