package aws_client_interfaces

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/athena"
	athenaTypes "github.com/aws/aws-sdk-go-v2/service/athena/types"
)

type MockAthenaClient struct {
	mockGetQueryExecutionResponses   []MockGetQueryExecutionResponse
	mockStartQueryExecutionResponses []MockStartQueryExecutionResponse
}

type MockGetQueryExecutionResponse struct {
	expectedInput           *athena.GetQueryExecutionInput
	getQueryExecutionOutput *athena.GetQueryExecutionOutput
	err                     error
}

type MockStartQueryExecutionResponse struct {
	expectedInput             *athena.StartQueryExecutionInput
	startQueryExecutionOutput *athena.StartQueryExecutionOutput
	err                       error
}

func NewMockAthenaClient() MockAthenaClient {
	return MockAthenaClient{
		mockGetQueryExecutionResponses:   []MockGetQueryExecutionResponse{},
		mockStartQueryExecutionResponses: []MockStartQueryExecutionResponse{},
	}
}

func (mock *MockAthenaClient) AllMocksCalled() bool {
	return len(mock.mockGetQueryExecutionResponses)+len(mock.mockStartQueryExecutionResponses) == 0
}

func (mock *MockAthenaClient) AddMockGetQueryExecutionResponse(expectedInput *athena.GetQueryExecutionInput, executionId string, queryState athenaTypes.QueryExecutionState, outputLocation string) {
	mock.mockGetQueryExecutionResponses = append(mock.mockGetQueryExecutionResponses, MockGetQueryExecutionResponse{
		expectedInput: expectedInput,
		err:           nil,
		getQueryExecutionOutput: &athena.GetQueryExecutionOutput{
			QueryExecution: &athenaTypes.QueryExecution{
				QueryExecutionId: expectedInput.QueryExecutionId,
				Status: &athenaTypes.QueryExecutionStatus{
					State: queryState,
				},
				ResultConfiguration: &athenaTypes.ResultConfiguration{
					OutputLocation: &outputLocation,
				},
			},
		},
	})
}

func (mock *MockAthenaClient) AddMockGetQueryExecutionError(expectedInput *athena.GetQueryExecutionInput, err error) {
	mock.mockGetQueryExecutionResponses = append(mock.mockGetQueryExecutionResponses, MockGetQueryExecutionResponse{
		expectedInput:           expectedInput,
		err:                     err,
		getQueryExecutionOutput: nil,
	})
}

func (mock *MockAthenaClient) AddMockStartQueryExecutionResponse(expectedInput *athena.StartQueryExecutionInput, executionId string) {
	mock.mockStartQueryExecutionResponses = append(mock.mockStartQueryExecutionResponses, MockStartQueryExecutionResponse{
		expectedInput: expectedInput,
		err:           nil,
		startQueryExecutionOutput: &athena.StartQueryExecutionOutput{
			QueryExecutionId: &executionId,
		},
	})
}

func (mock *MockAthenaClient) AddMockStartQueryExecutionError(expectedInput *athena.StartQueryExecutionInput, err error) {
	mock.mockStartQueryExecutionResponses = append(mock.mockStartQueryExecutionResponses, MockStartQueryExecutionResponse{
		expectedInput:             expectedInput,
		err:                       err,
		startQueryExecutionOutput: nil,
	})
}

func (mock *MockAthenaClient) GetQueryExecution(ctx context.Context, params *athena.GetQueryExecutionInput, optFns ...func(*athena.Options)) (*athena.GetQueryExecutionOutput, error) {
	if len(mock.mockGetQueryExecutionResponses) == 0 {
		return nil, &UnmockedCallToAwsApiClient{
			AwsClient:  "Athena",
			MethodCall: "GetQueryExecution",
		}
	}

	response := mock.mockGetQueryExecutionResponses[0]
	mock.mockGetQueryExecutionResponses = mock.mockGetQueryExecutionResponses[1:]

	if response.err != nil {
		return nil, response.err
	}

	err := compareExpectedGetQueryExecutionInputToActual(response.expectedInput, params)
	if err != nil {
		return nil, err
	}

	return response.getQueryExecutionOutput, nil
}

func compareExpectedGetQueryExecutionInputToActual(expected *athena.GetQueryExecutionInput, actual *athena.GetQueryExecutionInput) error {
	if expected.QueryExecutionId == actual.QueryExecutionId {
		return nil
	}

	return &UnexpectedInputToAwsApiClientMethod{
		MethodCall: "GetQueryExecution",
		Athena: &AthenaInputs{
			GetQueryExecutionInputs: &AthenaGetQueryExecutionInputs{
				ExpectedInput: expected,
				ActualInput:   actual,
			},
		},
	}
}

func (mock *MockAthenaClient) StartQueryExecution(ctx context.Context, params *athena.StartQueryExecutionInput, optFns ...func(*athena.Options)) (*athena.StartQueryExecutionOutput, error) {
	if len(mock.mockStartQueryExecutionResponses) == 0 {
		return nil, &UnmockedCallToAwsApiClient{
			AwsClient:  "Athena",
			MethodCall: "StartQueryExecution",
		}
	}

	response := mock.mockStartQueryExecutionResponses[0]
	mock.mockStartQueryExecutionResponses = mock.mockStartQueryExecutionResponses[1:]

	if response.err != nil {
		return nil, response.err
	}

	err := compareExpectedStartQueryExecutionInputToActual(response.expectedInput, params)
	if err != nil {
		return nil, err
	}

	return response.startQueryExecutionOutput, nil
}

func compareExpectedStartQueryExecutionInputToActual(expected *athena.StartQueryExecutionInput, actual *athena.StartQueryExecutionInput) error {
	if stripWhitespace(expected.QueryString) == stripWhitespace(actual.QueryString) && compareQueryParams(expected.ExecutionParameters, actual.ExecutionParameters) {
		return nil
	}

	return &UnexpectedInputToAwsApiClientMethod{
		MethodCall: "StartQueryExecution",
		Athena: &AthenaInputs{
			StartQueryExecutionInputs: &AthenaStartQueryExecutionInputs{
				ExpectedInput: expected,
				ActualInput:   actual,
			},
		},
	}
}

func stripWhitespace(input *string) string {
	return strings.Join(strings.Fields(*input), " ")
}

func compareQueryParams(expected []string, actual []string) bool {
	if len(expected) != len(actual) {
		return false
	}

	for i, expectedParam := range expected {
		if expectedParam != actual[i] {
			return false
		}
	}

	return true
}
