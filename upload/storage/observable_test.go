package storage

import (
	"strings"
	"testing"
	"time"
)

func TestObservableStorage_NewObservableStorage(t *testing.T) {
	// Create a mock storage backend
	mockStorage := NewMockStorage()

	// Create observable storage wrapper
	observableStorage := NewObservableStorage(mockStorage, "mock")
	if observableStorage == nil {
		t.Error("ObservableStorage should not be nil")
	}
}

func TestObservableStorage_Store(t *testing.T) {
	mockStorage := NewMockStorage()
	observableStorage := NewObservableStorage(mockStorage, "mock")

	// Test Store with observable wrapper
	data := []byte("test data")
	reader := strings.NewReader(string(data))

	filename, err := observableStorage.Store("test.txt", reader)
	if err != nil {
		t.Errorf("Store failed: %v", err)
	}
	if filename != "test.txt" {
		t.Errorf("Expected filename 'test.txt', got '%s'", filename)
	}
}

func TestObservableStorage_GetURL(t *testing.T) {
	mockStorage := NewMockStorage()
	observableStorage := NewObservableStorage(mockStorage, "mock")

	// Test GetURL with observable wrapper
	url := observableStorage.GetURL("test.txt")
	expected := "/uploads/test.txt"
	if url != expected {
		t.Errorf("Expected URL '%s', got '%s'", expected, url)
	}
}

func TestObservableStorage_Delete(t *testing.T) {
	mockStorage := NewMockStorage()
	observableStorage := NewObservableStorage(mockStorage, "mock")

	// Store a file first
	data := []byte("test data")
	reader := strings.NewReader(string(data))
	_, err := observableStorage.Store("test.txt", reader)
	if err != nil {
		t.Fatalf("Failed to store file: %v", err)
	}

	// Test Delete with observable wrapper
	err = observableStorage.Delete("test.txt")
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}
}

func TestObservableStorage_Exists(t *testing.T) {
	mockStorage := NewMockStorage()
	observableStorage := NewObservableStorage(mockStorage, "mock")

	// Test Exists with observable wrapper
	exists := observableStorage.Exists("test.txt")
	if exists {
		t.Error("File should not exist")
	}

	// Store a file and test again
	data := []byte("test data")
	reader := strings.NewReader(string(data))
	_, err := observableStorage.Store("test.txt", reader)
	if err != nil {
		t.Fatalf("Failed to store file: %v", err)
	}

	exists = observableStorage.Exists("test.txt")
	if !exists {
		t.Error("File should exist")
	}
}

func TestObservableStorage_GetSize(t *testing.T) {
	mockStorage := NewMockStorage()
	observableStorage := NewObservableStorage(mockStorage, "mock")

	// Store a file first
	data := []byte("test data")
	reader := strings.NewReader(string(data))
	_, err := observableStorage.Store("test.txt", reader)
	if err != nil {
		t.Fatalf("Failed to store file: %v", err)
	}

	// Test GetSize with observable wrapper
	size, err := observableStorage.GetSize("test.txt")
	if err != nil {
		t.Errorf("GetSize failed: %v", err)
	}
	if size != int64(len(data)) {
		t.Errorf("Expected size %d, got %d", len(data), size)
	}
}

func TestObservableStorage_ListFiles(t *testing.T) {
	mockStorage := NewMockStorage()
	observableStorage := NewObservableStorage(mockStorage, "mock")

	// Store some files first
	files := []string{"file1.txt", "file2.txt"}
	for _, filename := range files {
		data := []byte("test data")
		reader := strings.NewReader(string(data))
		_, err := observableStorage.Store(filename, reader)
		if err != nil {
			t.Fatalf("Failed to store file %s: %v", filename, err)
		}
	}

	// Test ListFiles with observable wrapper
	listedFiles, err := observableStorage.ListFiles()
	if err != nil {
		t.Errorf("ListFiles failed: %v", err)
	}
	if len(listedFiles) != len(files) {
		t.Errorf("Expected %d files, got %d", len(files), len(listedFiles))
	}
}

func TestObservableStorage_GetSignedURL(t *testing.T) {
	mockStorage := NewMockStorage()
	observableStorage := NewObservableStorage(mockStorage, "mock")

	// Test GetSignedURL with observable wrapper
	signedURL, err := observableStorage.GetSignedURL("test.txt", 15*time.Minute)
	if err == nil {
		t.Error("Expected error for GetSignedURL with mock storage")
	}
	if signedURL != "" {
		t.Error("Signed URL should be empty when error occurs")
	}
}

func TestObservableStorage_GetBucketInfo(t *testing.T) {
	mockStorage := NewMockStorage()
	observableStorage := NewObservableStorage(mockStorage, "mock")

	// Test GetBucketInfo with observable wrapper
	info, err := observableStorage.GetBucketInfo()
	if err != nil {
		t.Errorf("GetBucketInfo failed: %v", err)
	}
	if info == nil {
		t.Error("Bucket info should not be nil")
	}
}

func TestObservableStorage_Close(t *testing.T) {
	mockStorage := NewMockStorage()
	observableStorage := NewObservableStorage(mockStorage, "mock")

	// Test Close with observable wrapper
	err := observableStorage.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
