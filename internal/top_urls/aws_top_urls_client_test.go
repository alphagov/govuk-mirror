package top_urls_test

import (
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"mirrorer/internal/aws_client_interfaces"
	"mirrorer/internal/config"
	"mirrorer/internal/top_urls"

	"github.com/aws/aws-sdk-go-v2/service/athena"
	athenaTypes "github.com/aws/aws-sdk-go-v2/service/athena/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
)

func TestAwsGetTopUrls(t *testing.T) {
	yesterday := time.Now().AddDate(0, 0, -1)
	yesterdaysDay := strconv.FormatInt(int64(yesterday.Day()), 10)
	yesterdaysMonth := strconv.FormatInt(int64(yesterday.Month()), 10)
	yesterdaysYear := strconv.FormatInt(int64(yesterday.Year()), 10)

	cfg := config.MirrorComparisonConfig{
		CompareTopUnsampledCount:     3,
		CompareRemainingSampledCount: 2,
	}

	queryString := `
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
	queryExecutionId := "123-456"

	s3Bucket := "govuk-example-test-bucket"
	s3Key := "/path/to/file.csv"
	s3Path := fmt.Sprintf("s3://%s%s", s3Bucket, s3Key)

	athenaStartQueryExecutionInput := &athena.StartQueryExecutionInput{
		QueryString:         &queryString,
		ExecutionParameters: []string{yesterdaysDay, yesterdaysMonth, yesterdaysYear},
	}
	athenaGetQueryExecutionInput := &athena.GetQueryExecutionInput{
		QueryExecutionId: &queryExecutionId,
	}
	s3GetObjectInput := &s3.GetObjectInput{
		Bucket: &s3Bucket,
		Key:    &s3Key,
	}

	t.Run("GetTopUrls returns an error if athena.StartQueryExecution returns one", func(t *testing.T) {
		random := rand.New(rand.NewSource(99))
		expectedErr := fmt.Errorf("Test Error Returned by AWS")

		athenaClient := aws_client_interfaces.NewMockAthenaClient()
		athenaClient.AddMockStartQueryExecutionError(athenaStartQueryExecutionInput, expectedErr)
		s3Client := aws_client_interfaces.NewMockS3Client()

		topUrlsClient := top_urls.NewAwsTopUrlsClient(cfg, &athenaClient, &s3Client)

		topUrls, err := topUrlsClient.GetTopUrls(random)
		assert.Nil(t, topUrls)
		assert.ErrorIs(t, err, expectedErr)
		assert.True(t, athenaClient.AllMocksCalled())
	})

	t.Run("GetTopUrls returns an error if athena.GetQueryExecution returns one", func(t *testing.T) {
		random := rand.New(rand.NewSource(99))
		expectedErr := fmt.Errorf("Test Error Returned by AWS")

		athenaClient := aws_client_interfaces.NewMockAthenaClient()
		athenaClient.AddMockStartQueryExecutionResponse(athenaStartQueryExecutionInput, queryExecutionId)
		athenaClient.AddMockGetQueryExecutionError(athenaGetQueryExecutionInput, expectedErr)
		s3Client := aws_client_interfaces.NewMockS3Client()

		topUrlsClient := top_urls.NewAwsTopUrlsClient(cfg, &athenaClient, &s3Client)

		topUrls, err := topUrlsClient.GetTopUrls(random)
		assert.Nil(t, topUrls)
		assert.ErrorIs(t, err, expectedErr)
		assert.True(t, athenaClient.AllMocksCalled())
	})

	t.Run("GetTopUrls returns an AthenaQueryFailed error if the athena query does not end in success", func(t *testing.T) {
		random := rand.New(rand.NewSource(99))

		athenaClient := aws_client_interfaces.NewMockAthenaClient()
		athenaClient.AddMockStartQueryExecutionResponse(athenaStartQueryExecutionInput, queryExecutionId)
		athenaClient.AddMockGetQueryExecutionResponse(athenaGetQueryExecutionInput, queryExecutionId, athenaTypes.QueryExecutionStateCancelled, s3Path)
		s3Client := aws_client_interfaces.NewMockS3Client()

		topUrlsClient := top_urls.NewAwsTopUrlsClient(cfg, &athenaClient, &s3Client)

		topUrls, err := topUrlsClient.GetTopUrls(random)
		assert.Nil(t, topUrls)
		assert.IsType(t, &top_urls.AthenaQueryFailedError{}, err)
		assert.True(t, athenaClient.AllMocksCalled())
		assert.True(t, s3Client.AllMocksCalled())
	})

	t.Run("GetTopUrls returns an error if s3.GetObject returns an error", func(t *testing.T) {
		random := rand.New(rand.NewSource(99))
		expectedErr := fmt.Errorf("Test Error Returned by AWS")

		athenaClient := aws_client_interfaces.NewMockAthenaClient()
		athenaClient.AddMockStartQueryExecutionResponse(athenaStartQueryExecutionInput, queryExecutionId)
		athenaClient.AddMockGetQueryExecutionResponse(athenaGetQueryExecutionInput, queryExecutionId, athenaTypes.QueryExecutionStateSucceeded, s3Path)
		s3Client := aws_client_interfaces.NewMockS3Client()
		s3Client.AddMockGetObjectError(s3GetObjectInput, expectedErr)

		topUrlsClient := top_urls.NewAwsTopUrlsClient(cfg, &athenaClient, &s3Client)

		topUrls, err := topUrlsClient.GetTopUrls(random)
		assert.Nil(t, topUrls)
		assert.ErrorIs(t, err, expectedErr)
		assert.True(t, athenaClient.AllMocksCalled())
		assert.True(t, s3Client.AllMocksCalled())
	})

	t.Run("GetTopUrls returns the correct TopUrls list and ignores unparseable urls and counts", func(t *testing.T) {
		random := rand.New(rand.NewSource(99))

		csvFileReader := io.NopCloser(
			strings.NewReader(
				strings.Join(
					[]string{
						`"url","count"`,
						`"/Lorem","2173"`,
						`"/ipsum","1928"`,
						`"/dolor","0"`,
						`"/sit","4619"`,
						`"/amet","5"`,
						`"/consectetur","10"`,
						`"/adipiscing","3"`,
						`"/elit","10000"`,
						`"/sed","500"`,
						`"/do","7782"`,
						`"	","123"`, // The string contains a single tab character which is one of the few things net/url doesn't consider valid
						`"/good_path","bad_number"`,
					},
					"\n",
				),
			),
		)

		athenaClient := aws_client_interfaces.NewMockAthenaClient()
		athenaClient.AddMockStartQueryExecutionResponse(
			&athena.StartQueryExecutionInput{
				QueryString:         &queryString,
				ExecutionParameters: []string{yesterdaysDay, yesterdaysMonth, yesterdaysYear},
			},
			queryExecutionId,
		)
		athenaClient.AddMockGetQueryExecutionResponse(
			&athena.GetQueryExecutionInput{QueryExecutionId: &queryExecutionId},
			queryExecutionId,
			athenaTypes.QueryExecutionStateRunning,
			s3Path,
		)
		athenaClient.AddMockGetQueryExecutionResponse(
			&athena.GetQueryExecutionInput{QueryExecutionId: &queryExecutionId},
			queryExecutionId,
			athenaTypes.QueryExecutionStateSucceeded,
			s3Path,
		)
		s3Client := aws_client_interfaces.NewMockS3Client()
		s3Client.AddMockGetObjectResponse(
			&s3.GetObjectInput{
				Bucket: &s3Bucket,
				Key:    &s3Key,
			},
			csvFileReader,
		)

		topUrlsClient := top_urls.NewAwsTopUrlsClient(cfg, &athenaClient, &s3Client)

		topUrl1, err := url.Parse("/elit")
		assert.NoError(t, err)
		topUrl2, err := url.Parse("/do")
		assert.NoError(t, err)
		topUrl3, err := url.Parse("/sit")
		assert.NoError(t, err)

		sampledUrl1, err := url.Parse("/Lorem")
		assert.NoError(t, err)
		sampledUrl2, err := url.Parse("/amet")
		assert.NoError(t, err)

		expectedUnsampled := []top_urls.UrlHitCount{
			{ViewedUrl: *topUrl1, ViewCount: 10_000},
			{ViewedUrl: *topUrl2, ViewCount: 7_782},
			{ViewedUrl: *topUrl3, ViewCount: 4_619},
		}
		expectedSampled := []top_urls.UrlHitCount{
			{ViewedUrl: *sampledUrl1, ViewCount: 2_173},
			{ViewedUrl: *sampledUrl2, ViewCount: 5},
		}

		topUrls, err := topUrlsClient.GetTopUrls(random)
		assert.NoError(t, err)
		assert.Equal(t, expectedUnsampled, topUrls.TopUnsampledUrls)
		assert.Equal(t, expectedSampled, topUrls.RemainingSampledUrls)
	})
}
