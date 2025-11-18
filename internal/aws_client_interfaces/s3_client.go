package aws_client_interfaces

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

//go:generate go tool counterfeiter -o ../aws_client_mocks/ . S3GetObjectAPI
type S3GetObjectAPI interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}
