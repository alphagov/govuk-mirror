package main

import (
	"context"
	"mirrorer/internal/config"
	"mirrorer/internal/crawler"
	"mirrorer/internal/metrics"
	"mirrorer/internal/mime"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	initLogger()
	initMime()

	// Create waitGroup
	var wg sync.WaitGroup

	// Create context
	ctx, cancel := context.WithCancel(context.Background())

	// Create a non-global registry
	reg := prometheus.NewRegistry()

	// Initialize metrics
	prometheusMetrics := metrics.NewMetrics(reg)

	cfg := initConfig()

	// Validate that the SITE URL and allowed domains are accessible before crawling
	// Skip validation if SKIP_VALIDATION=true for offline testing
	if !cfg.SkipValidation {
		err := crawler.ValidateCrawlerConfig(cfg, 10*time.Second)
		checkError(err, "Configuration validation failed")
	}

	cr, err := crawler.NewCrawler(cfg, prometheusMetrics)
	checkError(err, "Error creating new crawler")

	// Go routine to send metrics to Prometheus Pushgateway
	wg.Go(func() {
		metrics.PushMetrics(&wg, ctx, cfg.MetricRefreshInterval)
	})

	// Run crawler
	cr.Run()

	// Signal PushMetrics goroutine to gracefully shutdown
	cancel()

	// Wait for PushMetrics goroutine to gracefully shutdown
	log.Info().Msg("Waiting for PushMetrics goroutine to gracefully shutdown")
	wg.Wait()
	log.Info().Msg("PushMetrics goroutine has shutdown. Main thread is shutting down")
}

func initLogger() {
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logLevel, ok := os.LookupEnv("LOG_LEVEL")
	if !ok {
		logLevel = "INFO"
	}
	level, err := zerolog.ParseLevel(logLevel)
	checkError(err, "Error parsing log level")
	zerolog.SetGlobalLevel(level)
}

func initMime() {
	err := mime.LoadAdditionalMimeTypes()
	checkError(err, "Error loading additional mime types")
}

func initConfig() *config.Config {
	cfg, err := config.NewConfig()
	checkError(err, "Error parsing configuration")
	return cfg
}

func checkError(err error, message string) {
	if err != nil {
		log.Fatal().Err(err).Msg(message)
	}
}
