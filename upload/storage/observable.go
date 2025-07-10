package storage

import (
	"context"
	"io"
	"time"

	"github.com/kdsmith18542/gokit/observability"
)

// ObservableStorage wraps a Storage implementation with observability
type ObservableStorage struct {
	storage     Storage
	storageType string
}

// NewObservableStorage creates a new observable storage wrapper
func NewObservableStorage(storage Storage, storageType string) *ObservableStorage {
	return &ObservableStorage{
		storage:     storage,
		storageType: storageType,
	}
}

// Store saves a file to the storage backend with observability
func (o *ObservableStorage) Store(filename string, reader io.Reader) (string, error) {
	start := time.Now()
	ctx := context.Background()

	path, err := o.storage.Store(filename, reader)
	duration := time.Since(start)

	observability.GetObserver().OnStorageOperation(ctx, "store", o.storageType, duration, err == nil)

	return path, err
}

// GetURL returns the public URL for accessing a stored file
func (o *ObservableStorage) GetURL(path string) string {
	start := time.Now()
	ctx := context.Background()

	url := o.storage.GetURL(path)
	duration := time.Since(start)

	observability.GetObserver().OnStorageOperation(ctx, "get_url", o.storageType, duration, true)

	return url
}

// Delete removes a file from the storage backend with observability
func (o *ObservableStorage) Delete(filename string) error {
	start := time.Now()
	ctx := context.Background()

	err := o.storage.Delete(filename)
	duration := time.Since(start)

	observability.GetObserver().OnStorageOperation(ctx, "delete", o.storageType, duration, err == nil)

	return err
}

// Exists checks if a file exists in the storage backend with observability
func (o *ObservableStorage) Exists(filename string) bool {
	start := time.Now()
	ctx := context.Background()

	exists := o.storage.Exists(filename)
	duration := time.Since(start)

	observability.GetObserver().OnStorageOperation(ctx, "exists", o.storageType, duration, true)

	return exists
}

// GetReader returns a reader for the specified file with observability.
func (o *ObservableStorage) GetReader(filename string) (io.ReadCloser, error) {
	start := time.Now()
	ctx := context.Background()

	reader, err := o.storage.GetReader(filename)
	duration := time.Since(start)

	observability.GetObserver().OnStorageOperation(ctx, "get_reader", o.storageType, duration, err == nil)

	return reader, err
}

// GetSize returns the size of a stored file in bytes with observability
func (o *ObservableStorage) GetSize(filename string) (int64, error) {
	start := time.Now()
	ctx := context.Background()

	size, err := o.storage.GetSize(filename)
	duration := time.Since(start)

	observability.GetObserver().OnStorageOperation(ctx, "get_size", o.storageType, duration, err == nil)

	return size, err
}

// ListFiles returns a list of all files stored in the backend with observability
func (o *ObservableStorage) ListFiles() ([]string, error) {
	start := time.Now()
	ctx := context.Background()

	files, err := o.storage.ListFiles()
	duration := time.Since(start)

	observability.GetObserver().OnStorageOperation(ctx, "list_files", o.storageType, duration, err == nil)

	return files, err
}

// GetSignedURL generates a pre-signed URL for temporary access to a file with observability
func (o *ObservableStorage) GetSignedURL(filename string, expiration time.Duration) (string, error) {
	start := time.Now()
	ctx := context.Background()

	url, err := o.storage.GetSignedURL(filename, expiration)
	duration := time.Since(start)

	observability.GetObserver().OnStorageOperation(ctx, "get_signed_url", o.storageType, duration, err == nil)

	return url, err
}

// GetBucketInfo returns metadata about the storage backend with observability
func (o *ObservableStorage) GetBucketInfo() (map[string]interface{}, error) {
	start := time.Now()
	ctx := context.Background()

	info, err := o.storage.GetBucketInfo()
	duration := time.Since(start)

	observability.GetObserver().OnStorageOperation(ctx, "get_bucket_info", o.storageType, duration, err == nil)

	return info, err
}

// Close performs cleanup operations for the storage backend with observability
func (o *ObservableStorage) Close() error {
	start := time.Now()
	ctx := context.Background()

	err := o.storage.Close()
	duration := time.Since(start)

	observability.GetObserver().OnStorageOperation(ctx, "close", o.storageType, duration, err == nil)

	return err
}
