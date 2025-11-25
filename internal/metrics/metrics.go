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
	httpErrorCounter     prometheus.Counter
	downloadErrorCounter prometheus.Counter
	downloadCounter      prometheus.Counter
	crawledPagesCounter  prometheus.Counter
	crawlerDuration      prometheus.Gauge
}

type MirrorMetrics struct {
	mirrorLastUpdatedGauge   *prometheus.GaugeVec
	mirrorResponseStatusCode *prometheus.GaugeVec
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
	}

	reg.MustRegister(m.httpErrorCounter)
	reg.MustRegister(m.downloadErrorCounter)
	reg.MustRegister(m.downloadCounter)
	reg.MustRegister(m.crawledPagesCounter)
	reg.MustRegister(m.crawlerDuration)

	return m
}

func NewMirrorMetrics(reg *prometheus.Registry) *MirrorMetrics {
	m := &MirrorMetrics{
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

// func getMirrorLastUpdated(last_update_chan chan float64, url string, backend string) {
// 	freshness, err := fetchMirrorFreshnessMetric(backend, url)
// 	if err != nil {
// 		return -1, err
// 	}
// 	last_update_chan <- freshness

// 	return freshness, nil
// }

func updateMirrorLastUpdatedGauge(m *MirrorMetrics, url string, backend string) error {
	freshness, err := fetchMirrorFreshnessMetric(backend, url)
	if err != nil {
		return err
	}
	fmt.Printf("Freshness %+v\n", freshness)

	m.mirrorLastUpdatedGauge.With(prometheus.Labels{"backend": backend}).Set(freshness)
	return nil
}

func updateMirrorResponseStatusCode(m *MirrorMetrics, url string, backend string) error {
	statusCode, err := fetchMirrorAvailabilityMetric(backend, url)
	if err != nil {
		return err
	}
	fmt.Printf("Status Code %+v\n", statusCode)

	m.mirrorResponseStatusCode.With(prometheus.Labels{"backend": backend}).Set(float64(statusCode))
	return nil
}

func UpdateMirrorMetrics(m *MirrorMetrics, cfg *config.Config, reg *prometheus.Registry, ctx context.Context) {
	// only run for up to 24 hours as next mirror job will then start
	// eg default of 4 hour interval means running for 20 hours with 5 iterations before next job starts
	// for range time.Hour*24/cfg.RefreshInterval - 1 {
	var index = 0
	for range time.Second*30/cfg.RefreshInterval - 1 {
		index += 1
		fmt.Printf("Index: %+v\n", index)
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
		fmt.Printf("Pushing mirror metrics")
		PushMirrorMetrics(reg, ctx, cfg)
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
			fmt.Printf("URL: %+v", cfg.PushGatewayUrl)
			if err != nil {
				log.Error().Err(err).Msg("Error pushing metrics to Prometheus Pushgateway")
			}

		case <-ctx.Done():
			log.Info().Msg("PushMetrics goroutine is shutting down...")
			return
		}
	}
}

func PushMirrorMetrics(reg *prometheus.Registry, ctx context.Context, cfg *config.Config) {
	err := push.New(cfg.PushGatewayUrl, "mirror_metrics").Gatherer(reg).Push()
	fmt.Printf("URL: %+v", cfg.PushGatewayUrl)
	if err != nil {
		log.Error().Err(err).Msg("Error pushing metrics to Prometheus Pushgateway")
	}
}
