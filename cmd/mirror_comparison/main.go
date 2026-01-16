package main

import (
	"context"
	"fmt"
	"math/rand"
	"mirrorer/internal/page_comparer"
	"mirrorer/internal/page_fetcher"
	"os"
	"strings"
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

	// maintain exit code because we want a non-zero exit code if we detect drift or encounter
	// an error, but we don't want to stop processing until we're done
	exitCode := 0

	log.Info().Msgf("Comparing top %d unsampled paths", len(urls.TopUnsampledUrls))
	failure := comparePages(urls.TopUnsampledUrls, cfg.Site)
	if failure {
		exitCode = 1
	}

	log.Info().Msgf("Comparing remaining %d sampled paths", len(urls.RemainingSampledUrls))
	failure = comparePages(urls.RemainingSampledUrls, cfg.Site)
	if failure {
		exitCode = 1
	}

	os.Exit(exitCode)
}

// comparePages runs through each of the URLs and compares their
// live and mirror versions.
//
// Returns true if there was an error so that the program can exit with
// a non-zero exit code. Does not return the error, because the errors are logged.
func comparePages(pages []top_urls.UrlHitCount, baseUrl string) bool {
	failure := false
	fetcher, err := page_fetcher.NewPageFetcher(baseUrl)
	if err != nil {
		log.Error().Err(err).Msg("error creating page fetcher")
		return true
	}

	for _, page := range pages {
		url := page.ViewedUrl.String()

		live, err := fetcher.FetchLivePage(url)
		if err != nil {
			log.Error().Err(err).Msgf("error fetching live page: %s", url)
			failure = true
			continue
		}

		mirror, err := fetcher.FetchMirrorPage(url)
		if err != nil {
			log.Error().Err(err).Msgf("error fetching mirror page: %s", url)
			failure = true
			continue
		}

		same, err := page_comparer.HaveSameBody(strings.NewReader(live), strings.NewReader(mirror))
		if err != nil {
			log.Error().Err(err).Msgf("error comparing live and mirror pages: %s", url)
			failure = true
			continue
		}

		if !same {
			log.Info().Err(fmt.Errorf("drift detected between live and mirror on %s", url))
			failure = true
		}

		log.Info().Bool("drift", !same).Msgf("Comparing %s (%d views)", url, page.ViewCount)
	}

	return failure
}
