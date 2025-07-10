package storage

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// MockS3Storage provides a mock implementation of S3 storage for testing
type MockS3Storage struct {
	files map[string][]byte
}

// NewMockS3 creates a new mock S3 storage instance
func NewMockS3() Storage {
	return &MockS3Storage{
		files: make(map[string][]byte),
	}
}

// Store saves a file to the mock S3 storage.
func (m *MockS3Storage) Store(filename string, reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	m.files[filename] = data
	return filename, nil
}

// GetURL returns a mock URL for the given path in mock S3 storage.
func (m *MockS3Storage) GetURL(path string) string {
	return fmt.Sprintf("https://mock-s3.amazonaws.com/test-bucket/%s", path)
}

// Delete removes a file from the mock S3 storage.
func (m *MockS3Storage) Delete(filename string) error {
	if _, exists := m.files[filename]; !exists {
		return fmt.Errorf("file not found: %s", filename)
	}
	delete(m.files, filename)
	return nil
}

// Exists checks if a file exists in the mock S3 storage.
func (m *MockS3Storage) Exists(filename string) bool {
	_, exists := m.files[filename]
	return exists
}

// GetSize returns the size of a file in mock S3 storage.
func (m *MockS3Storage) GetSize(filename string) (int64, error) {
	if data, exists := m.files[filename]; exists {
		return int64(len(data)), nil
	}
	return 0, fmt.Errorf("file not found: %s", filename)
}

// ListFiles returns a list of all files in mock S3 storage.
func (m *MockS3Storage) ListFiles() ([]string, error) {
	files := make([]string, 0, len(m.files))
	for filename := range m.files {
		files = append(files, filename)
	}
	return files, nil
}

// GetSignedURL generates a mock signed URL for mock S3 storage.
func (m *MockS3Storage) GetSignedURL(filename string, expiration time.Duration) (string, error) {
	if _, exists := m.files[filename]; !exists {
		return "", fmt.Errorf("file not found: %s", filename)
	}
	return fmt.Sprintf("https://mock-s3.amazonaws.com/test-bucket/%s?signature=mock", filename), nil
}

// GetReader returns a reader for the specified file in mock S3 storage.
func (m *MockS3Storage) GetReader(filename string) (io.ReadCloser, error) {
	if data, exists := m.files[filename]; exists {
		return io.NopCloser(strings.NewReader(string(data))), nil
	}
	return nil, fmt.Errorf("file not found: %s", filename)
}

// GetBucketInfo returns mock bucket information for mock S3 storage.
func (m *MockS3Storage) GetBucketInfo() (map[string]interface{}, error) {
	totalSize := int64(0)
	for _, data := range m.files {
		totalSize += int64(len(data))
	}

	return map[string]interface{}{
		"type":      "mock-s3",
		"bucket":    "test-bucket",
		"fileCount": len(m.files),
		"totalSize": totalSize,
	}, nil
}

// Close performs cleanup operations for mock S3 storage.
func (m *MockS3Storage) Close() error {
	return nil
}

// MockGCSStorage provides a mock implementation of GCS storage for testing
type MockGCSStorage struct {
	files map[string][]byte
}

// NewMockGCS creates a new mock GCS storage instance
func NewMockGCS() Storage {
	return &MockGCSStorage{
		files: make(map[string][]byte),
	}
}

// Store saves a file to the mock GCS storage.
func (m *MockGCSStorage) Store(filename string, reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	m.files[filename] = data
	return filename, nil
}

// GetURL returns a mock URL for the given path in mock GCS storage.
func (m *MockGCSStorage) GetURL(path string) string {
	return fmt.Sprintf("https://storage.googleapis.com/test-bucket/%s", path)
}

// Delete removes a file from the mock GCS storage.
func (m *MockGCSStorage) Delete(filename string) error {
	if _, exists := m.files[filename]; !exists {
		return fmt.Errorf("file not found: %s", filename)
	}
	delete(m.files, filename)
	return nil
}

// Exists checks if a file exists in the mock GCS storage.
func (m *MockGCSStorage) Exists(filename string) bool {
	_, exists := m.files[filename]
	return exists
}

// GetSize returns the size of a file in mock GCS storage.
func (m *MockGCSStorage) GetSize(filename string) (int64, error) {
	if data, exists := m.files[filename]; exists {
		return int64(len(data)), nil
	}
	return 0, fmt.Errorf("file not found: %s", filename)
}

// ListFiles returns a list of all files in mock GCS storage.
func (m *MockGCSStorage) ListFiles() ([]string, error) {
	files := make([]string, 0, len(m.files))
	for filename := range m.files {
		files = append(files, filename)
	}
	return files, nil
}

// GetSignedURL generates a mock signed URL for mock GCS storage.
func (m *MockGCSStorage) GetSignedURL(filename string, expiration time.Duration) (string, error) {
	if _, exists := m.files[filename]; !exists {
		return "", fmt.Errorf("file not found: %s", filename)
	}
	return fmt.Sprintf("https://storage.googleapis.com/test-bucket/%s?signature=mock", filename), nil
}

// GetReader returns a reader for the specified file in mock GCS storage.
func (m *MockGCSStorage) GetReader(filename string) (io.ReadCloser, error) {
	if data, exists := m.files[filename]; exists {
		return io.NopCloser(strings.NewReader(string(data))), nil
	}
	return nil, fmt.Errorf("file not found: %s", filename)
}

// GetBucketInfo returns mock bucket information for mock GCS storage.
func (m *MockGCSStorage) GetBucketInfo() (map[string]interface{}, error) {
	totalSize := int64(0)
	for _, data := range m.files {
		totalSize += int64(len(data))
	}

	return map[string]interface{}{
		"type":      "mock-gcs",
		"bucket":    "test-bucket",
		"fileCount": len(m.files),
		"totalSize": totalSize,
	}, nil
}

// Close performs cleanup operations for mock GCS storage.
func (m *MockGCSStorage) Close() error {
	return nil
}

// MockAzureStorage provides a mock implementation of Azure storage for testing
type MockAzureStorage struct {
	files map[string][]byte
}

// NewMockAzure creates a new mock Azure storage instance
func NewMockAzure() Storage {
	return &MockAzureStorage{
		files: make(map[string][]byte),
	}
}

// Store saves a file to the mock Azure storage.
func (m *MockAzureStorage) Store(filename string, reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	m.files[filename] = data
	return filename, nil
}

// GetURL returns a mock URL for the given path in mock Azure storage.
func (m *MockAzureStorage) GetURL(path string) string {
	return fmt.Sprintf("https://testaccount.blob.core.windows.net/test-container/%s", path)
}

// Delete removes a file from the mock Azure storage.
func (m *MockAzureStorage) Delete(filename string) error {
	if _, exists := m.files[filename]; !exists {
		return fmt.Errorf("file not found: %s", filename)
	}
	delete(m.files, filename)
	return nil
}

// Exists checks if a file exists in the mock Azure storage.
func (m *MockAzureStorage) Exists(filename string) bool {
	_, exists := m.files[filename]
	return exists
}

// GetSize returns the size of a file in mock Azure storage.
func (m *MockAzureStorage) GetSize(filename string) (int64, error) {
	if data, exists := m.files[filename]; exists {
		return int64(len(data)), nil
	}
	return 0, fmt.Errorf("file not found: %s", filename)
}

// ListFiles returns a list of all files in mock Azure storage.
func (m *MockAzureStorage) ListFiles() ([]string, error) {
	files := make([]string, 0, len(m.files))
	for filename := range m.files {
		files = append(files, filename)
	}
	return files, nil
}

// GetSignedURL generates a mock signed URL for mock Azure storage.
func (m *MockAzureStorage) GetSignedURL(filename string, expiration time.Duration) (string, error) {
	if _, exists := m.files[filename]; !exists {
		return "", fmt.Errorf("file not found: %s", filename)
	}
	return fmt.Sprintf("https://testaccount.blob.core.windows.net/test-container/%s?signature=mock", filename), nil
}

// GetReader returns a reader for the specified file in mock Azure storage.
func (m *MockAzureStorage) GetReader(filename string) (io.ReadCloser, error) {
	if data, exists := m.files[filename]; exists {
		return io.NopCloser(strings.NewReader(string(data))), nil
	}
	return nil, fmt.Errorf("file not found: %s", filename)
}

// GetBucketInfo returns mock bucket information for mock Azure storage.
func (m *MockAzureStorage) GetBucketInfo() (map[string]interface{}, error) {
	totalSize := int64(0)
	for _, data := range m.files {
		totalSize += int64(len(data))
	}

	return map[string]interface{}{
		"type":      "mock-azure",
		"container": "test-container",
		"fileCount": len(m.files),
		"totalSize": totalSize,
	}, nil
}

// Close performs cleanup operations for mock Azure storage.
func (m *MockAzureStorage) Close() error {
	return nil
}
