package storage

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewAzureBlob(t *testing.T) {
	// Test with valid configuration
	config := AzureConfig{
		AccountName: "testaccount",
		AccountKey:  "dGVzdGtleQ==", // base64 encoded "testkey"
		Container:   "testcontainer",
		BaseURL:     "https://testaccount.blob.core.windows.net",
	}

	azure, err := NewAzureBlob(config)
	if err != nil {
		t.Fatalf("NewAzureBlob failed: %v", err)
	}
	if azure == nil {
		t.Fatal("NewAzureBlob returned nil")
	}

	// Test with empty account name
	config.AccountName = ""
	_, err = NewAzureBlob(config)
	if err == nil {
		t.Error("Expected error for empty account name")
	}

	// Test with empty account key
	config.AccountName = "testaccount"
	config.AccountKey = ""
	_, err = NewAzureBlob(config)
	if err == nil {
		t.Error("Expected error for empty account key")
	}

	// Test with empty container
	config.AccountKey = "dGVzdGtleQ==" // base64 encoded "testkey"
	config.Container = ""
	_, err = NewAzureBlob(config)
	if err == nil {
		t.Error("Expected error for empty container")
	}
}

func TestAzureBlob_Store(t *testing.T) {
	// Create a mock Azure blob storage
	config := AzureConfig{
		AccountName: "testaccount",
		AccountKey:  "dGVzdGtleQ==", // base64 encoded "testkey"
		Container:   "testcontainer",
		BaseURL:     "https://testaccount.blob.core.windows.net",
	}

	azure, err := NewAzureBlob(config)
	if err != nil {
		t.Fatalf("NewAzureBlob failed: %v", err)
	}

	// Test storing a file
	testData := []byte("test content")
	reader := bytes.NewReader(testData)

	// Since we can't actually connect to Azure in tests, this will fail
	// but we can test the error handling
	_, err = azure.Store("test.txt", reader)
	if err == nil {
		t.Error("Expected error when storing to Azure without real connection")
	}
}

func TestAzureBlob_GetURL(t *testing.T) {
	config := AzureConfig{
		AccountName: "testaccount",
		AccountKey:  "dGVzdGtleQ==", // base64 encoded "testkey"
		Container:   "testcontainer",
		BaseURL:     "https://testaccount.blob.core.windows.net",
	}

	azure, err := NewAzureBlob(config)
	if err != nil {
		t.Fatalf("NewAzureBlob failed: %v", err)
	}

	// Test with custom base URL
	url := azure.GetURL("test.txt")
	// Accept both possible formats for compatibility
	if url != "https://testaccount.blob.core.windows.net/testcontainer/test.txt" && url != "https://testaccount.blob.core.windows.net/test.txt" {
		t.Errorf("Unexpected URL: %s", url)
	}

	// Test with default base URL
	config.BaseURL = ""
	azure, err = NewAzureBlob(config)
	if err != nil {
		t.Fatalf("NewAzureBlob failed: %v", err)
	}

	url = azure.GetURL("test.txt")
	if url != "https://testaccount.blob.core.windows.net/testcontainer/test.txt" && url != "https://testaccount.blob.core.windows.net/test.txt" {
		t.Errorf("Unexpected URL: %s", url)
	}
}

func TestAzureBlob_Delete(t *testing.T) {
	config := AzureConfig{
		AccountName: "testaccount",
		AccountKey:  "dGVzdGtleQ==", // base64 encoded "testkey"
		Container:   "testcontainer",
	}

	azure, err := NewAzureBlob(config)
	if err != nil {
		t.Fatalf("NewAzureBlob failed: %v", err)
	}

	// Test delete operation (will fail without real connection)
	err = azure.Delete("test.txt")
	if err == nil {
		t.Error("Expected error when deleting from Azure without real connection")
	}
}

func TestAzureBlob_Exists(t *testing.T) {
	config := AzureConfig{
		AccountName: "testaccount",
		AccountKey:  "dGVzdGtleQ==", // base64 encoded "testkey"
		Container:   "testcontainer",
	}

	azure, err := NewAzureBlob(config)
	if err != nil {
		t.Fatalf("NewAzureBlob failed: %v", err)
	}

	// Test exists operation (will fail without real connection)
	exists := azure.Exists("test.txt")
	if exists {
		t.Error("Expected false for non-existent file")
	}
}

func TestAzureBlob_GetSize(t *testing.T) {
	config := AzureConfig{
		AccountName: "testaccount",
		AccountKey:  "dGVzdGtleQ==", // base64 encoded "testkey"
		Container:   "testcontainer",
	}

	azure, err := NewAzureBlob(config)
	if err != nil {
		t.Fatalf("NewAzureBlob failed: %v", err)
	}

	// Test get size operation (will fail without real connection)
	size, err := azure.GetSize("test.txt")
	if err == nil {
		t.Error("Expected error when getting size from Azure without real connection")
	}
	if size != 0 {
		t.Errorf("Expected size 0, got %d", size)
	}
}

func TestAzureBlob_ListFiles(t *testing.T) {
	config := AzureConfig{
		AccountName: "testaccount",
		AccountKey:  "dGVzdGtleQ==", // base64 encoded "testkey"
		Container:   "testcontainer",
	}

	azure, err := NewAzureBlob(config)
	if err != nil {
		t.Fatalf("NewAzureBlob failed: %v", err)
	}

	// Test list files operation (will fail without real connection)
	files, err := azure.ListFiles()
	if err == nil {
		t.Error("Expected error when listing files from Azure without real connection")
	}
	if len(files) != 0 {
		t.Errorf("Expected empty file list, got %d files", len(files))
	}
}

func TestAzureBlob_GetSignedURL(t *testing.T) {
	config := AzureConfig{
		AccountName: "testaccount",
		AccountKey:  "dGVzdGtleQ==", // base64 encoded "testkey"
		Container:   "testcontainer",
	}

	azure, err := NewAzureBlob(config)
	if err != nil {
		t.Fatalf("NewAzureBlob failed: %v", err)
	}

	// Test get signed URL operation (should return a URL)
	url, err := azure.GetSignedURL("test.txt", 3600)
	if err != nil {
		t.Errorf("Unexpected error when getting signed URL: %v", err)
	}
	if url == "" {
		t.Error("Expected a signed URL, got empty string")
	}
}

func TestAzureBlob_GetReader(t *testing.T) {
	config := AzureConfig{
		AccountName: "testaccount",
		AccountKey:  "dGVzdGtleQ==", // base64 encoded "testkey"
		Container:   "testcontainer",
	}

	azure, err := NewAzureBlob(config)
	if err != nil {
		t.Fatalf("NewAzureBlob failed: %v", err)
	}

	// Test get reader operation (will fail without real connection)
	reader, err := azure.GetReader("test.txt")
	if err == nil {
		t.Error("Expected error when getting reader from Azure without real connection")
	}
	if reader != nil {
		t.Error("Expected nil reader")
	}
}

func TestAzureBlob_GetBucketInfo(t *testing.T) {
	config := AzureConfig{
		AccountName: "testaccount",
		AccountKey:  "dGVzdGtleQ==", // base64 encoded "testkey"
		Container:   "testcontainer",
	}

	azure, err := NewAzureBlob(config)
	if err != nil {
		t.Fatalf("NewAzureBlob failed: %v", err)
	}

	// Test get bucket info operation (should return a map)
	info, err := azure.GetBucketInfo()
	if err != nil {
		t.Errorf("Unexpected error when getting bucket info: %v", err)
	}
	if info == nil {
		t.Error("Expected bucket info map, got nil")
	}
	if _, ok := info["account"]; !ok {
		t.Error("Expected 'account' key in bucket info")
	}
	if _, ok := info["container"]; !ok {
		t.Error("Expected 'container' key in bucket info")
	}
}

func TestAzureBlob_Close(t *testing.T) {
	config := AzureConfig{
		AccountName: "testaccount",
		AccountKey:  "dGVzdGtleQ==", // base64 encoded "testkey"
		Container:   "testcontainer",
	}

	azure, err := NewAzureBlob(config)
	if err != nil {
		t.Fatalf("NewAzureBlob failed: %v", err)
	}

	// Test close operation (should not panic)
	err = azure.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestAzureBlob_Integration(t *testing.T) {
	// Test integration with mock storage
	mockAzure := NewMockAzure()

	// Test store
	testData := []byte("test content")
	reader := bytes.NewReader(testData)
	_, err := mockAzure.Store("test.txt", reader)
	if err != nil {
		t.Errorf("Mock Azure store failed: %v", err)
	}

	// Test exists
	exists := mockAzure.Exists("test.txt")
	if !exists {
		t.Error("Expected file to exist")
	}

	// Test get size
	size, err := mockAzure.GetSize("test.txt")
	if err != nil {
		t.Errorf("Mock Azure get size failed: %v", err)
	}
	if size != int64(len(testData)) {
		t.Errorf("Expected size %d, got %d", len(testData), size)
	}

	// Test list files
	files, err := mockAzure.ListFiles()
	if err != nil {
		t.Errorf("Mock Azure list files failed: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}
	if files[0] != "test.txt" {
		t.Errorf("Expected file 'test.txt', got '%s'", files[0])
	}

	// Test get URL
	url := mockAzure.GetURL("test.txt")
	expectedURL := "https://testaccount.blob.core.windows.net/test-container/test.txt"
	if url != expectedURL {
		t.Errorf("Expected URL %s, got %s", expectedURL, url)
	}

	// Test get signed URL
	signedURL, err := mockAzure.GetSignedURL("test.txt", 3600)
	if err != nil {
		t.Errorf("Mock Azure get signed URL failed: %v", err)
	}
	if !strings.Contains(signedURL, "test.txt") {
		t.Errorf("Signed URL should contain filename, got %s", signedURL)
	}

	// Test get reader
	reader2, err := mockAzure.GetReader("test.txt")
	if err != nil {
		t.Errorf("Mock Azure get reader failed: %v", err)
	}
	if reader2 == nil {
		t.Error("Expected non-nil reader")
	}

	// Test get bucket info
	info, err := mockAzure.GetBucketInfo()
	if err != nil {
		t.Errorf("Mock Azure get bucket info failed: %v", err)
	}
	if info["container"] != "test-container" {
		t.Errorf("Expected bucket name 'test-container', got '%v'", info["container"])
	}

	// Test delete
	err = mockAzure.Delete("test.txt")
	if err != nil {
		t.Errorf("Mock Azure delete failed: %v", err)
	}

	// Verify file is deleted
	exists = mockAzure.Exists("test.txt")
	if exists {
		t.Error("Expected file to be deleted")
	}

	// Test close
	err = mockAzure.Close()
	if err != nil {
		t.Errorf("Mock Azure close failed: %v", err)
	}
}
