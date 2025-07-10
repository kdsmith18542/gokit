package storage

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// MockStorage implements Storage interface for testing
type MockStorage struct {
	files map[string][]byte
	mu    sync.RWMutex
}

// NewMockStorage creates a new mock storage for testing
func NewMockStorage() *MockStorage {
	return &MockStorage{
		files: make(map[string][]byte),
	}
}

func (m *MockStorage) Store(filename string, reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	m.files[filename] = data
	m.mu.Unlock()
	return filename, nil
}

func (m *MockStorage) GetURL(path string) string {
	return "/uploads/" + path
}

func (m *MockStorage) Delete(filename string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.files[filename]; !exists {
		return fmt.Errorf("file not found: %s", filename)
	}
	delete(m.files, filename)
	return nil
}

func (m *MockStorage) Exists(filename string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.files[filename]
	return exists
}

func (m *MockStorage) GetSize(filename string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if data, exists := m.files[filename]; exists {
		return int64(len(data)), nil
	}
	return 0, fmt.Errorf("file not found: %s", filename)
}

func (m *MockStorage) ListFiles() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	files := make([]string, 0, len(m.files))
	for filename := range m.files {
		files = append(files, filename)
	}
	return files, nil
}

func (m *MockStorage) GetSignedURL(filename string, expiration time.Duration) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, exists := m.files[filename]; !exists {
		return "", fmt.Errorf("file not found: %s", filename)
	}
	return "/uploads/" + filename + "?signed=true", nil
}

func (m *MockStorage) GetReader(filename string) (io.ReadCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if data, exists := m.files[filename]; exists {
		return io.NopCloser(strings.NewReader(string(data))), nil
	}
	return nil, fmt.Errorf("file not found: %s", filename)
}

// GeneratePresignedPutURL generates a dummy pre-signed PUT URL for testing.
func (m *MockStorage) GeneratePresignedPutURL(filename string, expiration time.Duration, contentType string) (string, error) {
	return fmt.Sprintf("/uploads/%s?upload=true&expires=%d&content_type=%s", filename, int(expiration.Seconds()), contentType), nil
}

// GeneratePresignedGetURL generates a dummy pre-signed GET URL for testing.
func (m *MockStorage) GeneratePresignedGetURL(filename string, expiration time.Duration) (string, error) {
	return fmt.Sprintf("/uploads/%s?download=true&expires=%d", filename, int(expiration.Seconds())), nil
}

func (m *MockStorage) GetBucketInfo() (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return map[string]interface{}{
		"type":    "mock",
		"files":   len(m.files),
		"backend": "MockStorage",
	}, nil
}

func (m *MockStorage) Close() error {
	// Clear all files
	m.mu.Lock()
	m.files = make(map[string][]byte)
	m.mu.Unlock()
	return nil
}
