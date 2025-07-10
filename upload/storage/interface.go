// Package storage provides a unified interface for file storage backends in the upload package.
//
// Features:
//   - Pluggable storage backends (Local, S3, GCS, Azure Blob)
//   - Consistent API across all storage providers
//   - Pre-signed URL generation for direct uploads
//   - File metadata and management operations
//   - Thread-safe implementations
//
// Supported backends:
//   - Local: File system storage with configurable base path
//   - S3: Amazon S3 storage with IAM or access key authentication
//   - GCS: Google Cloud Storage with service account authentication
//   - Mock: In-memory storage for testing
//
// Example:
//
//	// Local storage
//	localStorage := storage.NewLocal("./uploads")
//	defer localStorage.Close()
//
//	// S3 storage
//	s3Storage := storage.NewS3(storage.S3Config{
//	    Bucket: "my-bucket",
//	    Region: "us-west-2",
//	    AccessKey: "AKIA...",
//	    SecretKey: "...",
//	})
//	defer s3Storage.Close()
//
//	// Use with upload processor
//	processor := upload.NewProcessor(localStorage, upload.Options{})
package storage

import (
	"io"
	"time"
)

// Storage defines the interface for file storage backends.
// All storage implementations must provide these methods for consistent behavior
// across different storage providers.
type Storage interface {
	// Store saves a file to the storage backend.
	// The filename parameter is the desired name for the stored file.
	// The reader provides the file content to be stored.
	// Returns the internal path/identifier for the stored file.
	Store(filename string, reader io.Reader) (string, error)

	// GetURL returns the public URL for accessing a stored file.
	// The path parameter is the internal path returned by Store().
	// Returns an empty string if the storage backend doesn't support public URLs.
	GetURL(path string) string

	// Delete removes a file from the storage backend.
	// The filename parameter should match the internal path returned by Store().
	// Returns an error if the file doesn't exist or cannot be deleted.
	Delete(filename string) error

	// Exists checks if a file exists in the storage backend.
	// The filename parameter should match the internal path returned by Store().
	// Returns true if the file exists, false otherwise.
	Exists(filename string) bool

	// GetSize returns the size of a stored file in bytes.
	// The filename parameter should match the internal path returned by Store().
	// Returns an error if the file doesn't exist or size cannot be determined.
	GetSize(filename string) (int64, error)

	// ListFiles returns a list of all files stored in the backend.
	// Returns file paths/identifiers that can be used with other methods.
	// The exact format of returned paths depends on the storage backend.
	ListFiles() ([]string, error)

	// GetSignedURL generates a pre-signed URL for temporary access to a file.
	// The filename parameter should match the internal path returned by Store().
	// The expiration parameter specifies how long the URL should be valid.
	// Returns an error if the storage backend doesn't support signed URLs.
	GetSignedURL(filename string, expiration time.Duration) (string, error)

	// GetReader returns an io.ReadCloser for reading a stored file.
	// The filename parameter should match the internal path returned by Store().
	// Returns an error if the file doesn't exist or cannot be read.
	// The caller is responsible for closing the returned reader.
	GetReader(filename string) (io.ReadCloser, error)

	// GetBucketInfo returns metadata about the storage backend.
	// The returned map contains backend-specific information such as
	// bucket name, region, total size, file count, etc.
	// Returns an error if the information cannot be retrieved.
	GetBucketInfo() (map[string]interface{}, error)

	// Close performs cleanup operations for the storage backend.
	// Should be called when the storage instance is no longer needed.
	// Returns an error if cleanup fails.
	Close() error
}
