package main

import (
	"context"
	"math/rand"
	"mirrorer/internal/drift_checker"
	"mirrorer/internal/page_comparer"
	"mirrorer/internal/page_fetcher"
	"os"
	"time"

	"mirrorer/internal/config"
	"mirrorer/internal/logger"
	"mirrorer/internal/top_urls"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/athena"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog/log"
)

func main() {
	err := logger.InitialiseLogger()
	if err != nil {
		log.Fatal().Err(err).Msg("Error parsing log level")
	}

	cfg, err := config.NewMirrorComparisonConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Error parsing config")
	}

	if err := cfg.Validate(); err != nil {
		log.Fatal().Err(err).Msg("Invalid config")
	}

	awsCfg, err := awsConfig.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load AWS config")
	}

	athenaClient := athena.NewFromConfig(awsCfg)
	s3Client := s3.NewFromConfig(awsCfg)

	topUrls := top_urls.NewAwsTopUrlsClient(*cfg, athenaClient, s3Client)
	urls, err := topUrls.GetTopUrls(rand.New(rand.NewSource(time.Now().UnixNano())))
	if err != nil {
		log.Fatal().Err(err).Msg("Error generating top urls")
	}

	fetcher, err := page_fetcher.NewPageFetcher(cfg.Site)
	if err != nil {
		log.Error().Err(err).Msg("error creating page fetcher")
	}

	comparer := page_comparer.PageComparer{}

	var notifier drift_checker.DriftNotifierInterface
	if cfg.HasSlackSettings() {
		log.Info().Msg("Using Slack credentials. Will notify about drifts on Slack")
		notifier = drift_checker.NewSlackDriftNotifier(cfg.SlackWebhookURL(), cfg.Site)
	} else {
		log.Info().Msg("No Slack credentials found. Will notify about drifts on stdout")
		notifier = drift_checker.StdOutDriftNotifier{}
	}

	driftsDetected := drift_checker.CheckPagesForDrift(urls, fetcher, &comparer, notifier)

	if driftsDetected {
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
