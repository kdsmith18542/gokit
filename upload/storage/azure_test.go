package storage

import (
	"strings"
	"testing"
	"time"
)

func TestAzureStorage_NewAzureBlob(t *testing.T) {
	// Test Azure constructor with valid config
	config := AzureConfig{
		AccountName: "testaccount",
		AccountKey:  "test-key",
		Container:   "test-container",
	}

	storage, err := NewAzureBlob(config)
	// Should fail with invalid credentials
	if err == nil {
		t.Error("Expected error for invalid Azure credentials")
	}
	if storage != nil {
		t.Error("Storage should be nil when error occurs")
	}
}

func TestAzureStorage_NewAzureBlobWithInvalidConfig(t *testing.T) {
	// Test Azure constructor with invalid config
	config := AzureConfig{
		AccountName: "",
		AccountKey:  "",
		Container:   "",
	}

	storage, err := NewAzureBlob(config)
	// Should fail due to invalid config
	if err == nil {
		t.Error("Expected error for invalid Azure config")
	}
	if storage != nil {
		t.Error("Storage should be nil when error occurs")
	}
}

func TestAzureStorage_GetURL(t *testing.T) {
	config := AzureConfig{
		AccountName: "testaccount",
		AccountKey:  "test-key",
		Container:   "test-container",
	}

	_, err := NewAzureBlob(config)
	// Should fail with invalid credentials
	if err == nil {
		t.Error("Expected error for invalid Azure credentials")
		return
	}
	// Test is skipped if storage creation fails
	t.Skip("Skipping test due to invalid credentials")
}

func TestAzureStorage_GetSignedURL(t *testing.T) {
	// Use mock Azure for testing
	storage := NewMockAzure()

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
	if !strings.Contains(signedURL, "blob.core.windows.net") {
		t.Errorf("Expected mock Azure URL, got '%s'", signedURL)
	}
}

func TestAzureStorage_GetBucketInfo(t *testing.T) {
	// Use mock Azure for testing
	storage := NewMockAzure()

	// Test GetBucketInfo with mock storage
	info, err := storage.GetBucketInfo()
	if err != nil {
		t.Errorf("GetBucketInfo failed: %v", err)
	}
	if info == nil {
		t.Error("Bucket info should not be nil")
	}
	if info["type"] != "mock-azure" {
		t.Errorf("Expected type 'mock-azure', got '%v'", info["type"])
	}
}

func TestAzureStorage_Store(t *testing.T) {
	// Use mock Azure for testing
	storage := NewMockAzure()

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

func TestAzureStorage_Delete(t *testing.T) {
	// Use mock Azure for testing
	storage := NewMockAzure()

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

func TestAzureStorage_Exists(t *testing.T) {
	// Use mock Azure for testing
	storage := NewMockAzure()

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

func TestAzureStorage_GetSize(t *testing.T) {
	// Use mock Azure for testing
	storage := NewMockAzure()

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

func TestAzureStorage_ListFiles(t *testing.T) {
	// Use mock Azure for testing
	storage := NewMockAzure()

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

func TestAzureStorage_Close(t *testing.T) {
	// Use mock Azure for testing
	storage := NewMockAzure()

	// Test Close with mock storage
	err := storage.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestAzureStorage_ErrorPaths(t *testing.T) {
	// Missing container
	config := AzureConfig{
		AccountName: "testaccount",
		AccountKey:  "test-key",
	}
	storage, err := NewAzureBlob(config)
	if err == nil {
		t.Error("Expected error for missing container")
	}
	if storage != nil {
		t.Error("Storage should be nil when error occurs")
	}

	// Invalid account key (simulate by passing invalid base64)
	config = AzureConfig{
		AccountName: "testaccount",
		AccountKey:  "not-base64!",
		Container:   "test-container",
	}
	storage, err = NewAzureBlob(config)
	if err == nil {
		t.Error("Expected error for invalid account key")
	}
	if storage != nil {
		t.Error("Storage should be nil when error occurs")
	}
}
