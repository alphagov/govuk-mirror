package aws_client_interfaces

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/athena"
)

type AthenaExecuteQueryApi interface {
	GetQueryExecution(ctx context.Context, params *athena.GetQueryExecutionInput, optFns ...func(*athena.Options)) (*athena.GetQueryExecutionOutput, error)
	StartQueryExecution(ctx context.Context, params *athena.StartQueryExecutionInput, optFns ...func(*athena.Options)) (*athena.StartQueryExecutionOutput, error)
}
