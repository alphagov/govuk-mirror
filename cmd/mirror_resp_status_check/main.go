package main

import (
	"mirrorer/internal/config"
	"mirrorer/internal/logger"
	"mirrorer/internal/metrics"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

func main() {
	err := logger.InitialiseLogger()
	if err != nil {
		log.Fatal().Err(err).Msg("Error parsing log level")
	}

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Error parsing config")
	}

	reg := prometheus.NewRegistry()
	prometheusMetrics := metrics.NewResponseMetrics(reg)
	go updateResponseMetrics(prometheusMetrics, cfg)

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	err = http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Server error")
	}
}

func updateResponseMetrics(m *metrics.ResponseMetrics, cfg *config.Config) {
	for {
		metrics.UpdateMirrorResponseStatusCode(m, cfg)

		time.Sleep(cfg.StatusCheckRefreshInterval)
	}
}
