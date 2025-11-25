package main

import (
	"context"
	"fmt"
	"mirrorer/internal/config"
	"mirrorer/internal/crawler"
	"mirrorer/internal/logger"
	"mirrorer/internal/metrics"
	"mirrorer/internal/mime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

func main() {
	err := logger.InitialiseLogger()
	checkError(err, "Error parsing log level")

	initMime()

	// Create waitGroup
	var wg sync.WaitGroup
	var wgMirrorMetrics sync.WaitGroup

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
		metrics.PushMetrics(reg, ctx, cfg)
	})

	// Run crawler
	cr.Run(prometheusMetrics)

	// Go routine to update mirror metrics
	wgMirrorMetrics.Go(func() {
		wgMirrorMetrics.Add(1)
		fmt.Printf("REFRESH_INTERVAL %+v\n", cfg.RefreshInterval)
		mirrorMetrics := metrics.NewMirrorMetrics(reg)
		metrics.UpdateMirrorMetrics(mirrorMetrics, cfg, reg, ctx)
	})

	wgMirrorMetrics.Wait()

	// Signal PushMetrics goroutine to gracefully shutdown
	// TODO: wait for mirror metrics job to complete
	cancel()

	// Wait for PushMetrics goroutine to gracefully shutdown
	log.Info().Msg("Waiting for PushMetrics goroutine to gracefully shutdown")
	wg.Wait()
	log.Info().Msg("PushMetrics goroutine has shutdown. Main thread is shutting down")
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
