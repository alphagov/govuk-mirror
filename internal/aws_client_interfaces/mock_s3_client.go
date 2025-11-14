package aws_client_interfaces

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type MockS3Client struct {
	mockGetObjectResponses []MockGetObjectResponse
}

type MockGetObjectResponse struct {
	expectedInput   *s3.GetObjectInput
	getObjectOutput *s3.GetObjectOutput
	err             error
}

func NewMockS3Client() MockS3Client {
	return MockS3Client{
		mockGetObjectResponses: []MockGetObjectResponse{},
	}
}

func (mock *MockS3Client) AllMocksCalled() bool {
	return len(mock.mockGetObjectResponses) == 0
}

func (mock *MockS3Client) AddMockGetObjectResponse(expectedInput *s3.GetObjectInput, body io.ReadCloser) {
	mock.mockGetObjectResponses = append(mock.mockGetObjectResponses, MockGetObjectResponse{
		expectedInput: expectedInput,
		err:           nil,
		getObjectOutput: &s3.GetObjectOutput{
			Body: body,
		},
	})
}

func (mock *MockS3Client) AddMockGetObjectError(expectedInput *s3.GetObjectInput, err error) {
	mock.mockGetObjectResponses = append(mock.mockGetObjectResponses, MockGetObjectResponse{
		expectedInput:   expectedInput,
		err:             err,
		getObjectOutput: nil,
	})
}

func (mock *MockS3Client) GetObject(_ context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if len(mock.mockGetObjectResponses) == 0 {
		return nil, &UnmockedCallToAwsApiClient{
			AwsClient:  "S3",
			MethodCall: "GetObject",
		}
	}

	response := mock.mockGetObjectResponses[0]
	mock.mockGetObjectResponses = mock.mockGetObjectResponses[1:]

	if response.err != nil {
		return nil, response.err
	}

	err := compareExpectedGetObjectInputToActual(response.expectedInput, params)
	if err != nil {
		return nil, err
	}

	return response.getObjectOutput, nil
}

func compareExpectedGetObjectInputToActual(expected *s3.GetObjectInput, actual *s3.GetObjectInput) error {
	if *expected.Bucket == *actual.Bucket && *expected.Key == *actual.Key {
		return nil
	}

	return &UnexpectedInputToAwsApiClientMethod{
		MethodCall: "GetObject",
		S3: &S3Inputs{
			GetObjectsInputs: &S3GetObjectsInputs{
				ExpectedInput: expected,
				ActualInput:   actual,
			},
		},
	}
}
