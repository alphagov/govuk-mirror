package main

import (
	"context"
	"fmt"
	"math/rand"
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

	awsCfg, err := awsConfig.LoadDefaultConfig(context.TODO())
	athenaClient := athena.NewFromConfig(awsCfg)
	s3Client := s3.NewFromConfig(awsCfg)

	topUrls := top_urls.NewAwsTopUrlsClient(*cfg, athenaClient, s3Client)
	urls, err := topUrls.GetTopUrls(rand.New(rand.NewSource(time.Now().UnixNano())))

	if err != nil {
		log.Fatal().Err(err).Msg("Error generating top urls")
	}

	printTopUrls(urls)

	log.Fatal().Msg("Command not yet implemented")
}

// Note, this function will be removed in the follow on PR, for now it's just to help validate
// the behaviour of the aws top urls client
func printTopUrls(topUrls *top_urls.TopUrls) {
	fmt.Println("Top 100 unsampled paths")
	for _, topUrl := range topUrls.TopUnsampledUrls {
		fmt.Printf("URL: %s Views: %d\n", &topUrl.ViewedUrl, topUrl.ViewCount)
	}

	fmt.Println("\n\n100 Sampled URLs from next 900 most popular")
	for _, topUrl := range topUrls.RemainingSampledUrls {
		fmt.Printf("URL: %s Views: %d\n", &topUrl.ViewedUrl, topUrl.ViewCount)
	}
}
