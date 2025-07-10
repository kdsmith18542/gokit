// Package storage provides a unified interface for file storage backends in the upload package.
package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Config holds configuration for S3 storage backend.
// All fields are optional and will use AWS defaults if not provided.
type S3Config struct {
	Bucket          string // S3 bucket name (required)
	Region          string // AWS region (optional, uses default if empty)
	AccessKeyID     string // AWS access key ID (optional, uses environment/default)
	SecretAccessKey string // AWS secret access key (optional, uses environment/default)
	Endpoint        string // Custom S3 endpoint (optional, for S3-compatible services)
	ForcePathStyle  bool   // Use path-style addressing (optional, for S3-compatible services)
}

// S3Storage implements the Storage interface for Amazon S3.
// It provides a high-performance, scalable storage solution for file uploads.
type S3Storage struct {
	client *s3.Client
	bucket string
	config S3Config
}

// NewS3 creates a new S3 storage instance with the specified configuration.
// The configuration can use AWS environment variables, IAM roles, or explicit credentials.
//
// Example:
//
//	// Using environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
//	s3Storage := storage.NewS3(storage.S3Config{
//	    Bucket: "my-bucket",
//	    Region: "us-west-2",
//	})
//
//	// Using explicit credentials
//	s3Storage := storage.NewS3(storage.S3Config{
//	    Bucket:          "my-bucket",
//	    Region:          "us-west-2",
//	    AccessKeyID:     "AKIA...",
//	    SecretAccessKey: "...",
//	})
//
//	// Using S3-compatible service (like MinIO)
//	s3Storage := storage.NewS3(storage.S3Config{
//	    Bucket:         "my-bucket",
//	    Endpoint:       "http://localhost:9000",
//	    ForcePathStyle: true,
//	})
func NewS3(config S3Config) (Storage, error) {
	if config.Bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	// Create AWS config
	var awsConfig aws.Config
	var err error

	if config.Endpoint != "" {
		// Custom endpoint (for S3-compatible services)
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               config.Endpoint,
				SigningRegion:     config.Region,
				HostnameImmutable: true,
			}, nil
		})

		awsConfig, err = awsconfig.LoadDefaultConfig(context.Background(),
			awsconfig.WithEndpointResolverWithOptions(customResolver),
			awsconfig.WithRegion(config.Region),
		)
	} else {
		// Standard AWS S3
		awsConfig, err = awsconfig.LoadDefaultConfig(context.Background(),
			awsconfig.WithRegion(config.Region),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}

	// Override credentials if provided
	if config.AccessKeyID != "" && config.SecretAccessKey != "" {
		awsConfig.Credentials = aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     config.AccessKeyID,
				SecretAccessKey: config.SecretAccessKey,
			}, nil
		})
	}

	// Create S3 client
	client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		if config.ForcePathStyle {
			o.UsePathStyle = true
		}
	})

	return &S3Storage{
		client: client,
		bucket: config.Bucket,
		config: config,
	}, nil
}

// Store saves a file to S3.
// The filename parameter is the desired name for the stored file.
// The reader provides the file content to be stored.
// Returns the S3 object key for the stored file.
func (s *S3Storage) Store(filename string, reader io.Reader) (string, error) {
	// Clean the filename to ensure it's a valid S3 key
	key := strings.TrimPrefix(filepath.Clean(filename), "/")

	// Upload to S3
	_, err := s.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   reader,
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %v", err)
	}

	return key, nil
}

// GetURL returns the public URL for accessing a stored file.
// The path parameter is the S3 object key returned by Store().
// Returns the S3 public URL for the object.
func (s *S3Storage) GetURL(path string) string {
	if s.config.Endpoint != "" {
		// Custom endpoint (S3-compatible service)
		endpoint := strings.TrimSuffix(s.config.Endpoint, "/")
		return fmt.Sprintf("%s/%s/%s", endpoint, s.bucket, path)
	}

	// Standard AWS S3 URL
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.config.Region, path)
}

// Delete removes a file from S3.
// The filename parameter should match the S3 object key returned by Store().
// Returns an error if the file doesn't exist or cannot be deleted.
func (s *S3Storage) Delete(filename string) error {
	_, err := s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filename),
	})

	if err != nil {
		return fmt.Errorf("failed to delete from S3: %v", err)
	}

	return nil
}

// Exists checks if a file exists in S3.
// The filename parameter should match the S3 object key returned by Store().
// Returns true if the file exists, false otherwise.
func (s *S3Storage) Exists(filename string) bool {
	_, err := s.client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filename),
	})

	return err == nil
}

// GetSize returns the size of a stored file in bytes.
// The filename parameter should match the S3 object key returned by Store().
// Returns an error if the file doesn't exist or size cannot be determined.
func (s *S3Storage) GetSize(filename string) (int64, error) {
	result, err := s.client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filename),
	})

	if err != nil {
		return 0, fmt.Errorf("failed to get object info: %v", err)
	}

	if result.ContentLength == nil {
		return 0, fmt.Errorf("content length not available")
	}

	return *result.ContentLength, nil
}

// ListFiles returns a list of all files stored in S3.
// Returns S3 object keys that can be used with other methods.
func (s *S3Storage) ListFiles() ([]string, error) {
	var files []string
	var continuationToken *string

	for {
		input := &s3.ListObjectsV2Input{
			Bucket: aws.String(s.bucket),
		}

		if continuationToken != nil {
			input.ContinuationToken = continuationToken
		}

		result, err := s.client.ListObjectsV2(context.Background(), input)
		if err != nil {
			return nil, fmt.Errorf("failed to list S3 objects: %v", err)
		}

		for _, object := range result.Contents {
			if object.Key != nil {
				files = append(files, *object.Key)
			}
		}

		// Check if there are more objects to fetch
		if result.IsTruncated == nil || !*result.IsTruncated {
			break
		}

		continuationToken = result.NextContinuationToken
	}

	return files, nil
}

// GetSignedURL generates a pre-signed URL for temporary access to a file.
// The filename parameter should match the S3 object key returned by Store().
// The expiration parameter specifies how long the URL should be valid.
// Returns a pre-signed URL for temporary access.
func (s *S3Storage) GetSignedURL(filename string, expiration time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

	request, err := presignClient.PresignPutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filename),
	}, s3.WithPresignExpires(expiration))

	if err != nil {
		return "", fmt.Errorf("failed to generate pre-signed URL: %v", err)
	}

	return request.URL, nil
}

// GetReader returns an io.ReadCloser for reading a stored file from S3.
// The filename parameter should match the S3 object key returned by Store().
// Returns an error if the file doesn't exist or cannot be read.
func (s *S3Storage) GetReader(filename string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filename),
	}

	result, err := s.client.GetObject(context.Background(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %v", err)
	}

	return result.Body, nil
}

// GetBucketInfo returns metadata about the S3 bucket.
// The returned map contains bucket information such as
// bucket name, region, total size, file count, etc.
func (s *S3Storage) GetBucketInfo() (map[string]interface{}, error) {
	info := map[string]interface{}{
		"type":   "s3",
		"bucket": s.bucket,
		"region": s.config.Region,
	}

	// Get bucket location
	locationResult, err := s.client.GetBucketLocation(context.Background(), &s3.GetBucketLocationInput{
		Bucket: aws.String(s.bucket),
	})
	if err == nil && locationResult.LocationConstraint != "" {
		info["location"] = string(locationResult.LocationConstraint)
	}

	// Calculate total size and file count
	var totalSize int64
	var fileCount int

	var continuationToken *string
	for {
		input := &s3.ListObjectsV2Input{
			Bucket: aws.String(s.bucket),
		}

		if continuationToken != nil {
			input.ContinuationToken = continuationToken
		}

		result, err := s.client.ListObjectsV2(context.Background(), input)
		if err != nil {
			break // Don't fail the entire operation for this
		}

		for _, object := range result.Contents {
			if object.Size != nil {
				totalSize += *object.Size
			}
			fileCount++
		}

		if result.IsTruncated == nil || !*result.IsTruncated {
			break
		}

		continuationToken = result.NextContinuationToken
	}

	info["totalSize"] = totalSize
	info["fileCount"] = fileCount

	return info, nil
}

// Close performs cleanup operations for the S3 storage backend.
// S3 client doesn't require explicit cleanup, but we implement it for interface compliance.
func (s *S3Storage) Close() error {
	// S3 client doesn't require cleanup
	return nil
}
