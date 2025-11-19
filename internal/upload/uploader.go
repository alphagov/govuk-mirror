package upload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mirrorer/internal/aws_client_interfaces"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rs/zerolog/log"
)

// Uploader represents the ability to upload a file to a remote file storage
//
//go:generate go tool counterfeiter . Uploader
type Uploader interface {
	// UploadFile uploads the file at filePath to the destinationKey in the remote file storage
	UploadFile(ctx context.Context, filePath string, destinationKey string) error
}

type S3Uploader struct {
	s3         aws_client_interfaces.S3ObjectUploadingAPI
	bucketName string
}

func NewUploader(s3 aws_client_interfaces.S3ObjectUploadingAPI, bucketName string) Uploader {
	return S3Uploader{
		s3:         s3,
		bucketName: bucketName,
	}
}

func (u S3Uploader) UploadFile(ctx context.Context, filePath string, destinationKey string) error {
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return err
	}

	s3ObjectMeta, err := u.s3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(u.bucketName),
		Key:    aws.String(destinationKey),
	})

	if err != nil {
		var notFoundErr *types.NotFound
		if !errors.As(err, &notFoundErr) {
			return fmt.Errorf("failed to get object metadata: %w", err)
		}
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer (func() {
		err := file.Close()
		if err != nil {
			log.Error().Err(err).Str("file", filePath).Msg("failed to close file")
		}
	})()

	// the object wasn't present in the remote
	// or the sizes were different
	if s3ObjectMeta == nil || *s3ObjectMeta.ContentLength != fileInfo.Size() {
		_, err = u.s3.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(u.bucketName),
			Key:    aws.String(destinationKey),
			Body:   io.Reader(file),
		})

		if err != nil {
			return fmt.Errorf("failed to write object: %w", err)
		}
	}

	return nil
}
