package top_urls_test

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"mirrorer/internal/aws_client_mocks"
	"mirrorer/internal/config"
	"mirrorer/internal/top_urls"
	"net/url"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/athena"
	athenaTypes "github.com/aws/aws-sdk-go-v2/service/athena/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
)

func TestAwsGetTopUrls(t *testing.T) {
	cfg := config.MirrorComparisonConfig{
		CompareTopUnsampledCount:     3,
		CompareRemainingSampledCount: 2,
	}
	queryExecutionId := "123-456"

	s3Bucket := "govuk-example-test-bucket"
	s3Key := "path/to/file.csv"
	s3Path := fmt.Sprintf("s3://%s/%s", s3Bucket, s3Key)

	t.Run("GetTopUrls returns an error if athena.StartQueryExecution returns one", func(t *testing.T) {
		random := rand.New(rand.NewSource(99))
		expectedErr := fmt.Errorf("Test Error Returned by AWS")

		athenaClient := aws_client_mocks.FakeAthenaExecuteQueryApi{}
		athenaClient.StartQueryExecutionReturns(nil, expectedErr)
		s3Client := aws_client_mocks.FakeS3GetObjectAPI{}

		topUrlsClient := top_urls.NewAwsTopUrlsClient(cfg, &athenaClient, &s3Client)

		topUrls, err := topUrlsClient.GetTopUrls(random)
		assert.Nil(t, topUrls)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, 1, athenaClient.StartQueryExecutionCallCount())
	})

	t.Run("GetTopUrls returns an error if athena.GetQueryExecution returns one", func(t *testing.T) {
		random := rand.New(rand.NewSource(99))
		expectedErr := fmt.Errorf("Test Error Returned by AWS")

		athenaClient := aws_client_mocks.FakeAthenaExecuteQueryApi{}
		athenaClient.StartQueryExecutionReturns(&athena.StartQueryExecutionOutput{
			QueryExecutionId: aws.String(queryExecutionId),
		}, nil)
		athenaClient.GetQueryExecutionReturns(nil, expectedErr)
		s3Client := aws_client_mocks.FakeS3GetObjectAPI{}

		topUrlsClient := top_urls.NewAwsTopUrlsClient(cfg, &athenaClient, &s3Client)

		topUrls, err := topUrlsClient.GetTopUrls(random)
		assert.Nil(t, topUrls)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, 1, athenaClient.StartQueryExecutionCallCount())
		assert.Equal(t, 1, athenaClient.GetQueryExecutionCallCount())
	})

	for _, terminalState := range []athenaTypes.QueryExecutionState{athenaTypes.QueryExecutionStateCancelled, athenaTypes.QueryExecutionStateFailed} {
		t.Run(fmt.Sprintf("GetTopUrls returns an AthenaQueryFailed error if the athena query does not end in success with state %s", terminalState), func(t *testing.T) {
			random := rand.New(rand.NewSource(99))

			athenaClient := aws_client_mocks.FakeAthenaExecuteQueryApi{}
			athenaClient.StartQueryExecutionReturns(&athena.StartQueryExecutionOutput{
				QueryExecutionId: aws.String(queryExecutionId),
			}, nil)

			athenaClient.GetQueryExecutionReturns(&athena.GetQueryExecutionOutput{
				QueryExecution: &athenaTypes.QueryExecution{
					QueryExecutionId: aws.String(queryExecutionId),
					Status: &athenaTypes.QueryExecutionStatus{
						State: terminalState,
					},
					ResultConfiguration: &athenaTypes.ResultConfiguration{
						OutputLocation: aws.String(s3Path),
					},
				},
			}, nil)

			s3Client := aws_client_mocks.FakeS3GetObjectAPI{}

			topUrlsClient := top_urls.NewAwsTopUrlsClient(cfg, &athenaClient, &s3Client)

			topUrls, err := topUrlsClient.GetTopUrls(random)
			assert.Nil(t, topUrls)
			assert.IsType(t, &top_urls.AthenaQueryFailedError{}, err)
			assert.Equal(t, 1, athenaClient.StartQueryExecutionCallCount())
			assert.Equal(t, 1, athenaClient.GetQueryExecutionCallCount())
		})
	}

	t.Run("GetTopUrls returns an error if s3.GetObject returns an error", func(t *testing.T) {
		random := rand.New(rand.NewSource(99))
		expectedErr := fmt.Errorf("Test Error Returned by AWS")

		athenaClient := aws_client_mocks.FakeAthenaExecuteQueryApi{}
		athenaClient.StartQueryExecutionReturns(&athena.StartQueryExecutionOutput{
			QueryExecutionId: aws.String(queryExecutionId),
		}, nil)

		athenaClient.GetQueryExecutionReturns(&athena.GetQueryExecutionOutput{
			QueryExecution: &athenaTypes.QueryExecution{
				QueryExecutionId: aws.String(queryExecutionId),
				Status: &athenaTypes.QueryExecutionStatus{
					State: athenaTypes.QueryExecutionStateSucceeded,
				},
				ResultConfiguration: &athenaTypes.ResultConfiguration{
					OutputLocation: aws.String(s3Path),
				},
			},
		}, nil)

		s3Client := aws_client_mocks.FakeS3GetObjectAPI{}
		s3Client.GetObjectCalls(func(ctx context.Context, input *s3.GetObjectInput, f ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			return nil, expectedErr
		})

		topUrlsClient := top_urls.NewAwsTopUrlsClient(cfg, &athenaClient, &s3Client)

		topUrls, err := topUrlsClient.GetTopUrls(random)
		assert.Nil(t, topUrls)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, 1, athenaClient.StartQueryExecutionCallCount())
		assert.Equal(t, 1, athenaClient.GetQueryExecutionCallCount())
		assert.Equal(t, 1, len(s3Client.Invocations()))
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

		athenaClient := aws_client_mocks.FakeAthenaExecuteQueryApi{}
		athenaClient.StartQueryExecutionReturns(&athena.StartQueryExecutionOutput{
			QueryExecutionId: aws.String(queryExecutionId),
		}, nil)

		for i, state := range []athenaTypes.QueryExecutionState{
			athenaTypes.QueryExecutionStateQueued,
			athenaTypes.QueryExecutionStateRunning,
			athenaTypes.QueryExecutionStateSucceeded,
		} {
			athenaClient.GetQueryExecutionReturnsOnCall(i, &athena.GetQueryExecutionOutput{
				QueryExecution: &athenaTypes.QueryExecution{
					QueryExecutionId: aws.String(queryExecutionId),
					Status: &athenaTypes.QueryExecutionStatus{
						State: state,
					},
					ResultConfiguration: &athenaTypes.ResultConfiguration{
						OutputLocation: aws.String(s3Path),
					},
				},
			}, nil)
		}
		s3Client := aws_client_mocks.FakeS3GetObjectAPI{}
		s3Client.GetObjectReturns(&s3.GetObjectOutput{
			Body: csvFileReader,
		}, nil)

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

	t.Run("GetTopUrls sets the catalog and database in the query", func(t *testing.T) {
		// StartQueryExecution returns an error for the sake of minimising the amount of
		// test setup needed. This test is only testing what is passed to StartQueryExecution
		random := rand.New(rand.NewSource(99))
		expectedErr := fmt.Errorf("Test Error Returned by AWS")

		athenaClient := aws_client_mocks.FakeAthenaExecuteQueryApi{}
		athenaClient.StartQueryExecutionReturns(nil, expectedErr)
		s3Client := aws_client_mocks.FakeS3GetObjectAPI{}

		topUrlsClient := top_urls.NewAwsTopUrlsClient(cfg, &athenaClient, &s3Client)

		_, err := topUrlsClient.GetTopUrls(random)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, 1, athenaClient.StartQueryExecutionCallCount())

		_, callParams, _ := athenaClient.StartQueryExecutionArgsForCall(0)

		assert.NotNil(t, callParams.QueryExecutionContext)
		assert.Equal(t, aws.String("AwsDataCatalog"), callParams.QueryExecutionContext.Catalog)
		assert.Equal(t, aws.String("fastly_logs"), callParams.QueryExecutionContext.Database)
	})
}
