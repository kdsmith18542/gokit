package storage

import (
	"strings"
	"testing"
	"time"
)

func TestGCSStorage_NewGCS(t *testing.T) {
	// Test GCS constructor with valid config
	config := GCSConfig{
		Bucket:          "test-bucket",
		ProjectID:       "test-project",
		CredentialsFile: "test-credentials.json",
	}

	storage, err := NewGCS(config)
	// Should fail with invalid credentials
	if err == nil {
		t.Error("Expected error for invalid GCS credentials")
	}
	if storage != nil {
		t.Error("Storage should be nil when error occurs")
	}
}

func TestGCSStorage_NewGCSWithInvalidConfig(t *testing.T) {
	// Test GCS constructor with invalid config
	config := GCSConfig{
		Bucket:          "",
		ProjectID:       "",
		CredentialsFile: "",
	}

	storage, err := NewGCS(config)
	// Should fail due to invalid config
	if err == nil {
		t.Error("Expected error for invalid GCS config")
	}
	if storage != nil {
		t.Error("Storage should be nil when error occurs")
	}
}

func TestGCSStorage_ErrorPaths(t *testing.T) {
	// Missing bucket
	config := GCSConfig{
		ProjectID:       "test-project",
		CredentialsFile: "test-credentials.json",
	}
	storage, err := NewGCS(config)
	if err == nil {
		t.Error("Expected error for missing bucket")
	}
	if storage != nil {
		t.Error("Storage should be nil when error occurs")
	}

	// Invalid credentials file (simulate by passing non-existent file)
	config = GCSConfig{
		Bucket:          "test-bucket",
		ProjectID:       "test-project",
		CredentialsFile: "/nonexistent/file.json",
	}
	storage, err = NewGCS(config)
	if err == nil {
		t.Error("Expected error for invalid credentials file")
	}
	if storage != nil {
		t.Error("Storage should be nil when error occurs")
	}
}

func TestGCSStorage_GetURL(t *testing.T) {
	config := GCSConfig{
		Bucket:          "test-bucket",
		ProjectID:       "test-project",
		CredentialsFile: "test-credentials.json",
	}

	_, err := NewGCS(config)
	// Should fail with invalid credentials
	if err == nil {
		t.Error("Expected error for invalid GCS credentials")
		return
	}
	// Test is skipped if storage creation fails
	t.Skip("Skipping test due to invalid credentials")
}

func TestGCSStorage_GetSignedURL(t *testing.T) {
	// Use mock GCS for testing
	storage := NewMockGCS()

	// Test GetSignedURL with mock storage
	signedURL, err := storage.GetSignedURL("test.txt", 15*time.Minute)
	if err == nil {
		t.Error("Expected error for GetSignedURL with non-existent file")
	}
	if signedURL != "" {
		t.Error("Signed URL should be empty when error occurs")
	}

	// Store a file and test again
	data := []byte("test data")
	reader := strings.NewReader(string(data))
	_, err = storage.Store("test.txt", reader)
	if err != nil {
		t.Fatalf("Failed to store file: %v", err)
	}

	signedURL, err = storage.GetSignedURL("test.txt", 15*time.Minute)
	if err != nil {
		t.Errorf("GetSignedURL failed: %v", err)
	}
	if !strings.Contains(signedURL, "storage.googleapis.com") {
		t.Errorf("Expected mock GCS URL, got '%s'", signedURL)
	}
}

func TestGCSStorage_GetBucketInfo(t *testing.T) {
	// Use mock GCS for testing
	storage := NewMockGCS()

	// Test GetBucketInfo with mock storage
	info, err := storage.GetBucketInfo()
	if err != nil {
		t.Errorf("GetBucketInfo failed: %v", err)
	}
	if info == nil {
		t.Error("Bucket info should not be nil")
	}
	if info["type"] != "mock-gcs" {
		t.Errorf("Expected type 'mock-gcs', got '%v'", info["type"])
	}
}

func TestGCSStorage_Store(t *testing.T) {
	// Use mock GCS for testing
	storage := NewMockGCS()

	// Test Store with mock storage
	data := []byte("test data")
	reader := strings.NewReader(string(data))

	filename, err := storage.Store("test.txt", reader)
	if err != nil {
		t.Errorf("Store failed: %v", err)
	}
	if filename != "test.txt" {
		t.Errorf("Expected filename 'test.txt', got '%s'", filename)
	}
}

func TestGCSStorage_Delete(t *testing.T) {
	// Use mock GCS for testing
	storage := NewMockGCS()

	// Store a file first
	data := []byte("test data")
	reader := strings.NewReader(string(data))
	_, err := storage.Store("test.txt", reader)
	if err != nil {
		t.Fatalf("Failed to store file: %v", err)
	}

	// Test Delete with mock storage
	err = storage.Delete("test.txt")
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}
}

func TestGCSStorage_Exists(t *testing.T) {
	// Use mock GCS for testing
	storage := NewMockGCS()

	// Test Exists with mock storage
	exists := storage.Exists("test.txt")
	if exists {
		t.Error("File should not exist")
	}

	// Store a file and test again
	data := []byte("test data")
	reader := strings.NewReader(string(data))
	_, err := storage.Store("test.txt", reader)
	if err != nil {
		t.Fatalf("Failed to store file: %v", err)
	}

	exists = storage.Exists("test.txt")
	if !exists {
		t.Error("File should exist")
	}
}

func TestGCSStorage_GetSize(t *testing.T) {
	// Use mock GCS for testing
	storage := NewMockGCS()

	// Store a file first
	data := []byte("test data")
	reader := strings.NewReader(string(data))
	_, err := storage.Store("test.txt", reader)
	if err != nil {
		t.Fatalf("Failed to store file: %v", err)
	}

	// Test GetSize with mock storage
	size, err := storage.GetSize("test.txt")
	if err != nil {
		t.Errorf("GetSize failed: %v", err)
	}
	if size != int64(len(data)) {
		t.Errorf("Expected size %d, got %d", len(data), size)
	}
}

func TestGCSStorage_ListFiles(t *testing.T) {
	// Use mock GCS for testing
	storage := NewMockGCS()

	// Store some files first
	files := []string{"file1.txt", "file2.txt"}
	for _, filename := range files {
		data := []byte("test data")
		reader := strings.NewReader(string(data))
		_, err := storage.Store(filename, reader)
		if err != nil {
			t.Fatalf("Failed to store file %s: %v", filename, err)
		}
	}

	// Test ListFiles with mock storage
	listedFiles, err := storage.ListFiles()
	if err != nil {
		t.Errorf("ListFiles failed: %v", err)
	}
	if len(listedFiles) != len(files) {
		t.Errorf("Expected %d files, got %d", len(files), len(listedFiles))
	}
}

func TestGCSStorage_Close(t *testing.T) {
	// Use mock GCS for testing
	storage := NewMockGCS()

	// Test Close with mock storage
	err := storage.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
