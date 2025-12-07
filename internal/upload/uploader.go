package upload

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
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
//counterfeiter:generate . Uploader
type Uploader interface {
	// UploadFile uploads the file at filePath to the destinationKey in the remote file storage
	UploadFile(ctx context.Context, filePath string, destinationKey string, contentType string) error
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

func (u S3Uploader) UploadFile(ctx context.Context, filePath string, destinationKey string, contentType string) error {
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
	if s3ObjectMeta == nil || *s3ObjectMeta.ContentLength != fileInfo.Size() || (s3ObjectMeta.ContentType != nil && *s3ObjectMeta.ContentType != contentType) {
		if s3ObjectMeta != nil && s3ObjectMeta.ContentType != nil && *s3ObjectMeta.ContentType != contentType {
			log.Info().Msgf("File %s has a different content type on S3 than live, uploading", filePath)
		}

		fmt.Println("FILE IS DIFFERENT")
		hasher := sha1.New()

		if _, err := io.Copy(hasher, file); err != nil {
			return fmt.Errorf("failed to copy file bytes into hashing buffer %s: %w", filePath, err)
		}
		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			return fmt.Errorf("failed to rewind file %s: %w", filePath, err)
		}

		checksum := base64.StdEncoding.EncodeToString(hasher.Sum(nil))

		_, err = u.s3.PutObject(ctx, &s3.PutObjectInput{
			Bucket:            aws.String(u.bucketName),
			Key:               aws.String(destinationKey),
			Body:              io.Reader(file),
			ChecksumAlgorithm: types.ChecksumAlgorithmSha1,
			ChecksumSHA1:      aws.String(checksum),
			ContentType:       aws.String(contentType),
		})

		if err != nil {
			return fmt.Errorf("failed to write object: %w", err)
		}
	}

	return nil
}
