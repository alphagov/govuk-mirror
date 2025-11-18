package top_urls_test

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/athena"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type UnmockedCallToAwsApiClient struct {
	AwsClient  string
	MethodCall string
}

func (e *UnmockedCallToAwsApiClient) Error() string {
	return fmt.Sprintf(
		"received unlcall to %s.%s with no mock added",
		e.AwsClient,
		e.MethodCall,
	)
}

type UnexpectedInputToAwsApiClientMethod struct {
	MethodCall string
	Athena     *AthenaInputs
	S3         *S3Inputs
}

type AthenaInputs struct {
	GetQueryExecutionInputs   *AthenaGetQueryExecutionInputs
	StartQueryExecutionInputs *AthenaStartQueryExecutionInputs
}

type AthenaGetQueryExecutionInputs struct {
	ExpectedInput *athena.GetQueryExecutionInput
	ActualInput   *athena.GetQueryExecutionInput
}

type AthenaStartQueryExecutionInputs struct {
	ExpectedInput *athena.StartQueryExecutionInput
	ActualInput   *athena.StartQueryExecutionInput
}

type S3Inputs struct {
	GetObjectsInputs *S3GetObjectsInputs
}

type S3GetObjectsInputs struct {
	ExpectedInput *s3.GetObjectInput
	ActualInput   *s3.GetObjectInput
}

func (e *UnexpectedInputToAwsApiClientMethod) Error() string {
	var expectedInputs, actualInputs any
	var client, methodCall string

	if e.Athena != nil {
		client = "Athena"
		if e.Athena.GetQueryExecutionInputs != nil {
			methodCall = "GetQueryExecutionInputs"
			expectedInputs = e.Athena.GetQueryExecutionInputs.ExpectedInput
			actualInputs = e.Athena.GetQueryExecutionInputs.ActualInput
		} else if e.Athena.StartQueryExecutionInputs != nil {
			methodCall = "StartQueryExecutionInputs"
			expectedInputs = e.Athena.StartQueryExecutionInputs.ExpectedInput
			actualInputs = e.Athena.StartQueryExecutionInputs.ActualInput
		}
	} else if e.S3 != nil {
		client = "S3"
		if e.S3.GetObjectsInputs != nil {
			expectedInputs = e.S3.GetObjectsInputs.ExpectedInput
			actualInputs = e.S3.GetObjectsInputs.ActualInput
		}
	}

	return fmt.Sprintf(
		"received unexpected input to %s.%s\n\nExpected: %v\n\nActual: %v",
		client,
		methodCall,
		expectedInputs,
		actualInputs,
	)
}
