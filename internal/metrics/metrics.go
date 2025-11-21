package metrics

import (
	"context"
	"fmt"
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
	mirrorLastUpdatedGauge    *prometheus.GaugeVec
	mirrorResponseStatusCode  *prometheus.GaugeVec
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

func NewMirrorMetrics(reg *prometheus.Registry) *Metrics {
	m := &Metrics{
		mirrorLastUpdatedGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "govuk_mirror_last_updated_time",
			Help: "Last time the mirror was updated",
		}, []string{"backend"}),
		mirrorResponseStatusCode: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "govuk_mirror_response_status_code",
			Help: "Response status code for the MIRROR_AVAILABILITY_URL probe",
		}, []string{"backend"}),
	}

	reg.MustRegister(m.mirrorLastUpdatedGauge)
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

func closeResponse(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		err := resp.Body.Close()
		if err != nil {
			log.Error().Err(err).Msg("Error closing response body")
		}
	}
}

func fetchMirrorAvailabilityMetric(backend string, url string) (httpStatus int, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Backend-Override", backend)

	resp, err := httpClient.Do(req)
	if err != nil {
		return
	}
	defer closeResponse(resp)

	return resp.StatusCode, nil
}

func fetchMirrorFreshnessMetric(backend string, url string) (seconds float64, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Backend-Override", backend)

	resp, err := httpClient.Do(req)
	if err != nil {
		return
	}
	defer closeResponse(resp)

	if resp.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("request failed with status code: %s", resp.Status)
	}

	lastModified := resp.Header.Get("Last-Modified")

	t, err := time.Parse(time.RFC1123, lastModified)
	if err != nil {
		return
	}

	return float64(t.Unix()), nil
}

func updateMirrorLastUpdatedGauge(m *Metrics, url string, backend string) error {
	freshness, err := fetchMirrorFreshnessMetric(backend, url)
	if err != nil {
		return err
	}

	m.mirrorLastUpdatedGauge.With(prometheus.Labels{"backend": backend}).Set(freshness)
	return nil
}

func updateMirrorResponseStatusCode(m *Metrics, url string, backend string) error {
	statusCode, err := fetchMirrorAvailabilityMetric(backend, url)
	if err != nil {
		return err
	}

	m.mirrorResponseStatusCode.With(prometheus.Labels{"backend": backend}).Set(float64(statusCode))
	return nil
}

func UpdateMirrorMetrics(m *Metrics, cfg *config.Config, reg *prometheus.Registry, ctx context.Context) {
	for {
		for _, backend := range cfg.Backends {
			err := updateMirrorLastUpdatedGauge(m, cfg.MirrorFreshnessUrl, backend)
			if err != nil {
				log.Error().Str("metric", "govuk_mirror_last_updated_time").Str("backend", backend).Err(err).Msg("Error updating metrics")
			}

			err = updateMirrorResponseStatusCode(m, cfg.MirrorAvailabilityUrl, backend)
			if err != nil {
				log.Error().Str("metric", "govuk_mirror_response_status_code").Str("backend", backend).Err(err).Msg("Error updating metrics")
			}
		}
		PushMetrics(reg, ctx, cfg)
		time.Sleep(cfg.RefreshInterval)
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
