package upload

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"mirrorer/internal/aws_client_mocks"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
)

func setupFixtures(t *testing.T, files map[string]string) string {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "govuk-mirror_internal_upload_tests")
	assert.NoError(t, err)

	for name, content := range files {
		fullPath := filepath.Join(tmpDir, name)
		err = os.WriteFile(fullPath, []byte(content), 0644)
		assert.NoError(t, err, "error writing temporary file %s	", fullPath)
	}

	return tmpDir
}

func teardownFixtures(t *testing.T, tmpDir string) {
	err := os.RemoveAll(tmpDir)
	assert.NoError(t, err, "error cleaning up after tests")
}

func assertFileWasUploaded(
	t *testing.T,
	s3Client *aws_client_mocks.FakeS3ObjectUploadingAPI,
	key string,
	contentType string,
) {

	assert.Equal(t, 1, s3Client.PutObjectCallCount(), "uploader should have made one PutObject call")

	_, putCallArgs, _ := s3Client.PutObjectArgsForCall(0)
	assert.Equal(t, aws.String("test-bucket"), putCallArgs.Bucket)
	assert.Equal(t, aws.String(key), putCallArgs.Key)
	assert.Equal(t, aws.String(contentType), putCallArgs.ContentType)
	// can't make assertions about the content of the file because
	// the reader is closed at the end of the method
}

func TestS3Uploader(t *testing.T) {
	var s3Client *aws_client_mocks.FakeS3ObjectUploadingAPI

	t.Run("returns an error if the file does not exist", func(t *testing.T) {
		tmpDir := setupFixtures(t, map[string]string{})
		defer teardownFixtures(t, tmpDir)

		s3Client = &aws_client_mocks.FakeS3ObjectUploadingAPI{}
		uploader := NewUploader(s3Client, "test-bucket")

		err := uploader.UploadFile(t.Context(), path.Join(tmpDir, "unknown_file"), "key", "text/html")
		assert.Error(t, err, fmt.Errorf("file not found"))
	})

	t.Run("returns an error if getting the object from S3 fails", func(t *testing.T) {
		tmpDir := setupFixtures(t, map[string]string{
			"a_file": "some content",
		})
		defer teardownFixtures(t, tmpDir)

		s3Client = &aws_client_mocks.FakeS3ObjectUploadingAPI{}
		uploader := NewUploader(s3Client, "test-bucket")

		var irrelevantAWSError error = &types.TooManyParts{}
		s3Client.HeadObjectReturns(nil, irrelevantAWSError)
		err := uploader.UploadFile(t.Context(), path.Join(tmpDir, "a_file"), "key", "text/html")

		assert.ErrorIs(t, err, irrelevantAWSError)
	})

	t.Run("if the object does not exist in S3, it uploads the new file", func(t *testing.T) {
		files := map[string]string{
			"a_file": "some content",
		}
		tmpDir := setupFixtures(t, files)
		defer teardownFixtures(t, tmpDir)

		s3Client = &aws_client_mocks.FakeS3ObjectUploadingAPI{}
		uploader := NewUploader(s3Client, "test-bucket")

		s3Client.HeadObjectReturns(nil, &types.NotFound{})
		s3Client.PutObjectReturns(&s3.PutObjectOutput{
			Size: aws.Int64(int64(len(files["a_file"]))),
		}, nil)

		err := uploader.UploadFile(t.Context(), path.Join(tmpDir, "a_file"), "key", "text/html")
		assert.NoError(t, err)

		assertFileWasUploaded(t, s3Client, "key", "text/html")
	})

	t.Run("if the object exists in s3, and the size is the same, does not upload the file", func(t *testing.T) {
		files := map[string]string{
			"a_file": "some content",
		}
		tmpDir := setupFixtures(t, files)
		defer teardownFixtures(t, tmpDir)

		s3Client = &aws_client_mocks.FakeS3ObjectUploadingAPI{}
		uploader := NewUploader(s3Client, "test-bucket")

		s3Client.HeadObjectReturns(&s3.HeadObjectOutput{
			ContentLength: aws.Int64(int64(len(files["a_file"]))),
		}, nil)

		err := uploader.UploadFile(t.Context(), path.Join(tmpDir, "a_file"), "key", "text/html")
		assert.NoError(t, err)

		assert.Equal(t, 0, s3Client.PutObjectCallCount())
	})

	t.Run("if the object exists in s3, and the size is different, uploads the file", func(t *testing.T) {
		files := map[string]string{
			"a_file": "some content",
		}
		tmpDir := setupFixtures(t, files)
		defer teardownFixtures(t, tmpDir)

		s3Client = &aws_client_mocks.FakeS3ObjectUploadingAPI{}
		uploader := NewUploader(s3Client, "test-bucket")

		s3Client.HeadObjectReturns(&s3.HeadObjectOutput{
			ContentLength: aws.Int64(1),
		}, nil)
		s3Client.PutObjectReturns(&s3.PutObjectOutput{
			Size: aws.Int64(int64(len(files["a_file"]))),
		}, nil)

		err := uploader.UploadFile(t.Context(), path.Join(tmpDir, "a_file"), "key", "text/html")
		assert.NoError(t, err)

		assertFileWasUploaded(t, s3Client, "key", "text/html")
	})

	t.Run("if the object exists in s3, and the content-type is different, uploads the file", func(t *testing.T) {
		files := map[string]string{
			"a_file": "some content",
		}
		tmpDir := setupFixtures(t, files)
		defer teardownFixtures(t, tmpDir)

		s3Client = &aws_client_mocks.FakeS3ObjectUploadingAPI{}
		uploader := NewUploader(s3Client, "test-bucket")

		s3Client.HeadObjectReturns(&s3.HeadObjectOutput{
			ContentLength: aws.Int64(1),
			ContentType: aws.String("application/octet-stream"),
		}, nil)
		s3Client.PutObjectReturns(&s3.PutObjectOutput{
			Size: aws.Int64(int64(len(files["a_file"]))),
		}, nil)

		err := uploader.UploadFile(t.Context(), path.Join(tmpDir, "a_file"), "key", "text/html")
		assert.NoError(t, err)

		assertFileWasUploaded(t, s3Client, "key", "text/html")
	})

	t.Run("returns an error if putting the object fails", func(t *testing.T) {
		tmpDir := setupFixtures(t, map[string]string{
			"a_file": "some content",
		})
		defer teardownFixtures(t, tmpDir)

		s3Client = &aws_client_mocks.FakeS3ObjectUploadingAPI{}
		uploader := NewUploader(s3Client, "test-bucket")

		expectedError := &types.InvalidRequest{}
		s3Client.HeadObjectReturns(nil, &types.NotFound{})
		s3Client.PutObjectReturns(nil, expectedError)
		err := uploader.UploadFile(t.Context(), path.Join(tmpDir, "a_file"), "key", "text/html")

		assert.ErrorIs(t, err, expectedError)
	})

	t.Run("when uploading a file, the SHA1 checksum is provided", func(t *testing.T) {
		files := map[string]string{
			"a_file": "some content",
		}
		tmpDir := setupFixtures(t, files)
		defer teardownFixtures(t, tmpDir)

		hasher := sha1.New()
		hasher.Write([]byte(files["a_file"]))
		checksum := base64.StdEncoding.EncodeToString(hasher.Sum(nil))

		s3Client = &aws_client_mocks.FakeS3ObjectUploadingAPI{}
		uploader := NewUploader(s3Client, "test-bucket")

		s3Client.HeadObjectReturns(nil, &types.NotFound{})
		s3Client.PutObjectReturns(&s3.PutObjectOutput{
			Size:         aws.Int64(int64(len(files["a_file"]))),
			ChecksumSHA1: aws.String(checksum),
		}, nil)

		err := uploader.UploadFile(t.Context(), path.Join(tmpDir, "a_file"), "key", "text/html")
		assert.NoError(t, err)

		assert.Equal(t, 1, s3Client.PutObjectCallCount())

		_, args, _ := s3Client.PutObjectArgsForCall(0)

		assert.Equal(t, types.ChecksumAlgorithmSha1, args.ChecksumAlgorithm)
		assert.Equal(t, aws.String(checksum), args.ChecksumSHA1)
	})

	t.Run("when uploading a file, the Content Type is provided", func(t *testing.T) {
		files := map[string]string{
			"a_file": "some content",
		}
		tmpDir := setupFixtures(t, files)
		defer teardownFixtures(t, tmpDir)

		s3Client = &aws_client_mocks.FakeS3ObjectUploadingAPI{}
		uploader := NewUploader(s3Client, "test-bucket")

		s3Client.HeadObjectReturns(nil, &types.NotFound{})
		s3Client.PutObjectReturns(&s3.PutObjectOutput{
			Size: aws.Int64(int64(len(files["a_file"]))),
		}, nil)

		err := uploader.UploadFile(t.Context(), path.Join(tmpDir, "a_file"), "key", "text/css")
		assert.NoError(t, err)

		assertFileWasUploaded(t, s3Client, "key", "text/css")
	})
}
