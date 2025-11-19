package aws_client_interfaces

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3GetObjectAPI is a subset of the AWS S3 API surface area that deals with retrieving objects
//
//go:generate go tool counterfeiter -o ../aws_client_mocks/ . S3GetObjectAPI
type S3GetObjectAPI interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

// S3ObjectUploadingAPI is a subset of the AWS S3 API surface area that deals with uploading objects
//
//go:generate go tool counterfeiter -o ../aws_client_mocks/ . S3ObjectUploadingAPI
type S3ObjectUploadingAPI interface {
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}
