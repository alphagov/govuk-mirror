package main

import (
	"mirrorer/internal/config"
	"mirrorer/internal/logger"
	"mirrorer/internal/metrics"
	"time"

	"github.com/prometheus/client_golang/prometheus"
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
	prometheusMetrics := metrics.NewMetrics(reg)
	updateResponseMetrics(prometheusMetrics, cfg, reg)
}

func updateResponseMetrics(m *metrics.Metrics, cfg *config.Config, reg *prometheus.Registry) {
	for {
		metrics.UpdateAndPushMirrorResponseStatusCode(m, cfg, reg)

		time.Sleep(cfg.StatusCheckRefreshInterval)
	}
}
