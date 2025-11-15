package top_urls

import (
	"context"
	"encoding/csv"
	"fmt"
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"

	"mirrorer/internal/aws_client_interfaces"
	"mirrorer/internal/config"

	"github.com/aws/aws-sdk-go-v2/service/athena"
	athenaTypes "github.com/aws/aws-sdk-go-v2/service/athena/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog/log"
)

type athenaQueryExecutionId *string
type resultsS3Path struct {
	Bucket string
	Key    string
}

type AwsTopUrlsClient struct {
	cfg          config.MirrorComparisonConfig
	athenaClient aws_client_interfaces.AthenaExecuteQueryApi
	s3Client     aws_client_interfaces.S3GetObjectAPI
}

type AthenaQueryFailedError struct {
	QueryState       athenaTypes.QueryExecutionState
	QueryExecutionId string
}

func (e *AthenaQueryFailedError) Error() string {
	return fmt.Sprintf(
		"The athena query with Query Execution ID %s, to generate top results did not succeeed. Query ended in %s state",
		e.QueryExecutionId,
		e.QueryState,
	)
}

func NewAwsTopUrlsClient(cfg config.MirrorComparisonConfig, athenaClient aws_client_interfaces.AthenaExecuteQueryApi, s3Client aws_client_interfaces.S3GetObjectAPI) AwsTopUrlsClient {
	return AwsTopUrlsClient{
		cfg:          cfg,
		athenaClient: athenaClient,
		s3Client:     s3Client,
	}
}

func (topUrlsClient *AwsTopUrlsClient) GetTopUrls(random *rand.Rand) (*TopUrls, error) {
	ctx := context.Background()

	queryExecutionId, err := topUrlsClient.startAthenaQuery(ctx)
	if err != nil {
		return nil, err
	}

	s3Path, err := topUrlsClient.waitForAthenaQuery(ctx, queryExecutionId)
	if err != nil {
		return nil, err
	}

	csvRows, err := topUrlsClient.getAthenaQueryResultsFromS3(ctx, s3Path)
	if err != nil {
		return nil, err
	}

	urlHitCounts := csvRowsToUrlHitCounts(csvRows)
	topUrls, err := NewTopUrls(urlHitCounts, topUrlsClient.cfg.CompareTopUnsampledCount, topUrlsClient.cfg.CompareRemainingSampledCount, random)
	if err != nil {
		return nil, err
	}

	return topUrls, nil
}

func csvRowsToUrlHitCounts(csvRows [][]string) []UrlHitCount {
	var urlHitCounts []UrlHitCount

	for _, row := range csvRows[1:] {
		u, err := url.Parse(row[0])
		if err != nil {
			log.Warn().Msgf("Couldn't url parse '%s', skipping", row[0])
			continue
		}

		c, err := strconv.ParseInt(row[1], 10, 64)
		if err != nil {
			log.Warn().Msgf("Couldn't parse the view count '%s' of url '%s', skipping", row[1], row[0])
			continue
		}

		urlHitCounts = append(urlHitCounts, UrlHitCount{
			ViewedUrl: *u,
			ViewCount: c,
		})
	}

	return urlHitCounts
}

func (topUrlsClient *AwsTopUrlsClient) startAthenaQuery(ctx context.Context) (athenaQueryExecutionId, error) {
	yesterday := time.Now().AddDate(0, 0, -1)
	athenaQuery := `
		SELECT
		    url, count(1) as "count"
		FROM
		    fastly_logs.govuk_www
		WHERE
		    date = ?
		    AND month = ?
		    AND year = ?
		    AND url NOT LIKE '%/assets/%'
		    AND url NOT LIKE '/api/%'
		    AND url NOT LIKE '/search/%'
		    AND status >= 200 AND status < 300
		GROUP BY
		    url
		ORDER BY
		    "count" DESC
		LIMIT 1000
	`

	log.Info().Msg("Starting Athena Query")
	startQueryExecutionResponse, err := topUrlsClient.athenaClient.StartQueryExecution(ctx, &athena.StartQueryExecutionInput{
		QueryString: &athenaQuery,
		ExecutionParameters: []string{
			strconv.FormatInt(int64(yesterday.Day()), 10),
			strconv.FormatInt(int64(yesterday.Month()), 10),
			strconv.FormatInt(int64(yesterday.Year()), 10),
		},
	})

	if err != nil {
		return nil, err
	}

	return startQueryExecutionResponse.QueryExecutionId, nil
}

func (topUrlsClient *AwsTopUrlsClient) waitForAthenaQuery(ctx context.Context, queryExecutionId athenaQueryExecutionId) (*resultsS3Path, error) {
	var queryExecutionState = athenaTypes.QueryExecutionStateQueued
	var queryExecutionResponse *athena.GetQueryExecutionOutput
	var err error

	log.Info().Msg("Waiting for Athena Query to complete")
	for queryExecutionState == athenaTypes.QueryExecutionStateQueued || queryExecutionState == athenaTypes.QueryExecutionStateRunning {
		queryExecutionResponse, err = topUrlsClient.athenaClient.GetQueryExecution(ctx, &athena.GetQueryExecutionInput{
			QueryExecutionId: queryExecutionId,
		})

		if err != nil {
			return nil, err
		}

		queryExecutionState = queryExecutionResponse.QueryExecution.Status.State
		log.Info().Msgf("Athena Query current state is %s", queryExecutionState)
		time.Sleep(500 * time.Millisecond)
	}

	if queryExecutionState != athenaTypes.QueryExecutionStateSucceeded {
		log.Error().Msgf("Athena Query did not succeed, terminal state is %s", queryExecutionState)
		return nil, &AthenaQueryFailedError{
			QueryState:       queryExecutionState,
			QueryExecutionId: *queryExecutionId,
		}
	}

	log.Info().Msgf("Athena Query did not succeed, terminal state is %s", queryExecutionState)
	s3Path, err := s3PathStringToS3Path(queryExecutionResponse.QueryExecution.ResultConfiguration.OutputLocation)
	if err != nil {
		return nil, err
	}

	log.Info().Msgf("Athena Query succeeded, output saved to %s", *queryExecutionResponse.QueryExecution.ResultConfiguration.OutputLocation)
	return s3Path, nil
}

func s3PathStringToS3Path(s3PathString *string) (*resultsS3Path, error) {
	parsedUrl, err := url.Parse(*s3PathString)
	if err != nil {
		return nil, err
	}

	return &resultsS3Path{
		Bucket: parsedUrl.Host,
		Key:    strings.TrimPrefix(parsedUrl.Path, "/"),
	}, nil
}

func (topUrlsClient *AwsTopUrlsClient) getAthenaQueryResultsFromS3(ctx context.Context, s3Path *resultsS3Path) ([][]string, error) {
	log.Info().Msgf("Getting Athena Query output from S3")
	s3Object, err := topUrlsClient.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s3Path.Bucket,
		Key:    &s3Path.Key,
	})
	if err != nil {
		return nil, err
	}

	defer func() {
		err := s3Object.Body.Close()
		if err != nil {
			log.Warn().Msgf(
				"Error %v when trying to close the body when reading from s3 object s3://%s%s",
				err,
				s3Path.Bucket,
				s3Path.Key,
			)
		}
	}()

	csvReader := csv.NewReader(s3Object.Body)
	csvRows, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}

	log.Info().Msgf("CSV successfully read from S3")
	return csvRows, nil
}
