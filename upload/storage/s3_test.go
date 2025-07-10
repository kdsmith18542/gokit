package storage

import (
	"strings"
	"testing"
	"time"
)

func TestS3Storage_NewS3(t *testing.T) {
	// Test S3 constructor with valid config
	config := S3Config{
		Region:          "us-east-1",
		Bucket:          "test-bucket",
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret-key",
	}

	storage, err := NewS3(config)
	if err != nil {
		t.Errorf("NewS3 failed: %v", err)
	}
	if storage == nil {
		t.Error("Storage should not be nil")
	}
}

func TestS3Storage_NewS3WithEndpoint(t *testing.T) {
	// Test S3 constructor with custom endpoint
	config := S3Config{
		Region:          "us-east-1",
		Bucket:          "test-bucket",
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret-key",
		Endpoint:        "http://localhost:9000",
		ForcePathStyle:  true,
	}

	storage, err := NewS3(config)
	if err != nil {
		t.Errorf("NewS3 with endpoint failed: %v", err)
	}
	if storage == nil {
		t.Error("Storage should not be nil")
	}
}

func TestS3Storage_NewS3WithInvalidConfig(t *testing.T) {
	// Test S3 constructor with invalid config
	config := S3Config{
		Region:          "",
		Bucket:          "",
		AccessKeyID:     "",
		SecretAccessKey: "",
	}

	storage, err := NewS3(config)
	// Should fail due to invalid config
	if err == nil {
		t.Error("Expected error for invalid S3 config")
	}
	if storage != nil {
		t.Error("Storage should be nil when error occurs")
	}
}

func TestS3Storage_ErrorPaths(t *testing.T) {
	// Missing bucket
	config := S3Config{
		Region: "us-east-1",
	}
	storage, err := NewS3(config)
	if err == nil {
		t.Error("Expected error for missing bucket")
	}
	if storage != nil {
		t.Error("Storage should be nil when error occurs")
	}

	// Invalid region (simulate by passing empty string)
	config = S3Config{
		Bucket: "test-bucket",
		Region: "",
	}
	storage, err = NewS3(config)
	if err != nil {
		t.Errorf("Should not error for empty region (uses AWS default): %v", err)
	}
	if storage == nil {
		t.Error("Storage should not be nil for valid config")
	}
}

func TestS3Storage_GetURL(t *testing.T) {
	config := S3Config{
		Region:          "us-east-1",
		Bucket:          "test-bucket",
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret-key",
	}

	storage, err := NewS3(config)
	if err != nil {
		t.Fatalf("NewS3 failed: %v", err)
	}

	url := storage.GetURL("test.txt")
	expected := "https://test-bucket.s3.us-east-1.amazonaws.com/test.txt"
	if url != expected {
		t.Errorf("Expected URL '%s', got '%s'", expected, url)
	}
}

func TestS3Storage_GetSignedURL(t *testing.T) {
	// Use mock S3 for testing
	storage := NewMockS3()

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
	if !strings.Contains(signedURL, "mock-s3.amazonaws.com") {
		t.Errorf("Expected mock S3 URL, got '%s'", signedURL)
	}
}

func TestS3Storage_GetBucketInfo(t *testing.T) {
	// Use mock S3 for testing
	storage := NewMockS3()

	// Test GetBucketInfo with mock storage
	info, err := storage.GetBucketInfo()
	if err != nil {
		t.Errorf("GetBucketInfo failed: %v", err)
	}
	if info == nil {
		t.Error("Bucket info should not be nil")
	}
	if info["type"] != "mock-s3" {
		t.Errorf("Expected type 'mock-s3', got '%v'", info["type"])
	}
}

func TestS3Storage_Store(t *testing.T) {
	// Use mock S3 for testing
	storage := NewMockS3()

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

func TestS3Storage_Delete(t *testing.T) {
	// Use mock S3 for testing
	storage := NewMockS3()

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

func TestS3Storage_Exists(t *testing.T) {
	// Use mock S3 for testing
	storage := NewMockS3()

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

func TestS3Storage_GetSize(t *testing.T) {
	// Use mock S3 for testing
	storage := NewMockS3()

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

func TestS3Storage_ListFiles(t *testing.T) {
	// Use mock S3 for testing
	storage := NewMockS3()

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

func TestS3Storage_Close(t *testing.T) {
	// Use mock S3 for testing
	storage := NewMockS3()

	// Test Close with mock storage
	err := storage.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
