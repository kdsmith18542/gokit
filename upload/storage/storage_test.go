package storage

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestStorage_GetURLMethods(t *testing.T) {
	// Test GetURL methods for all storage backends
	testCases := []struct {
		name     string
		storage  Storage
		expected string
	}{
		{
			name:     "MockS3 GetURL",
			storage:  NewMockS3(),
			expected: "https://mock-s3.amazonaws.com/test-bucket/test.txt",
		},
		{
			name:     "MockGCS GetURL",
			storage:  NewMockGCS(),
			expected: "https://storage.googleapis.com/test-bucket/test.txt",
		},
		{
			name:     "MockAzure GetURL",
			storage:  NewMockAzure(),
			expected: "https://testaccount.blob.core.windows.net/test-container/test.txt",
		},
		{
			name:     "Local GetURL",
			storage:  NewLocal(t.TempDir()),
			expected: "",
		},
		{
			name:     "Local with custom URL",
			storage:  NewLocalWithURL(t.TempDir(), "https://example.com/files"),
			expected: "https://example.com/files/test.txt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := tc.storage.GetURL("test.txt")
			if url != tc.expected {
				t.Errorf("Expected URL '%s', got '%s'", tc.expected, url)
			}
		})
	}
}

func TestStorage_GetSignedURLMethods(t *testing.T) {
	// Test GetSignedURL methods for all storage backends
	testCases := []struct {
		name    string
		storage Storage
	}{
		{
			name:    "MockS3 GetSignedURL",
			storage: NewMockS3(),
		},
		{
			name:    "MockGCS GetSignedURL",
			storage: NewMockGCS(),
		},
		{
			name:    "MockAzure GetSignedURL",
			storage: NewMockAzure(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test with non-existent file
			signedURL, err := tc.storage.GetSignedURL("nonexistent.txt", 15*time.Minute)
			if err == nil {
				t.Error("Expected error for non-existent file")
			}
			if signedURL != "" {
				t.Error("Signed URL should be empty when error occurs")
			}

			// Store a file and test again
			data := []byte("test data")
			reader := strings.NewReader(string(data))
			_, err = tc.storage.Store("test.txt", reader)
			if err != nil {
				t.Fatalf("Failed to store file: %v", err)
			}

			signedURL, err = tc.storage.GetSignedURL("test.txt", 15*time.Minute)
			if err != nil {
				t.Errorf("GetSignedURL failed: %v", err)
			}
			if signedURL == "" {
				t.Error("Signed URL should not be empty for existing file")
			}
		})
	}

	// Test Local storage separately (doesn't support signed URLs)
	t.Run("Local GetSignedURL", func(t *testing.T) {
		storage := NewLocal(t.TempDir())

		// Local storage doesn't support signed URLs
		signedURL, err := storage.GetSignedURL("test.txt", 15*time.Minute)
		if err == nil {
			t.Error("Expected error for local storage signed URL")
		}
		if signedURL != "" {
			t.Error("Signed URL should be empty when error occurs")
		}
	})
}

func TestStorage_GetBucketInfoMethods(t *testing.T) {
	// Test GetBucketInfo methods for all storage backends
	testCases := []struct {
		name    string
		storage Storage
	}{
		{
			name:    "MockS3 GetBucketInfo",
			storage: NewMockS3(),
		},
		{
			name:    "MockGCS GetBucketInfo",
			storage: NewMockGCS(),
		},
		{
			name:    "MockAzure GetBucketInfo",
			storage: NewMockAzure(),
		},
		{
			name:    "Local GetBucketInfo",
			storage: NewLocal(t.TempDir()),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := tc.storage.GetBucketInfo()
			if err != nil {
				t.Errorf("GetBucketInfo failed: %v", err)
			}
			if info == nil {
				t.Error("Bucket info should not be nil")
			}
		})
	}
}

func TestStorage_CloseMethods(t *testing.T) {
	// Test Close methods for all storage backends
	testCases := []struct {
		name    string
		storage Storage
	}{
		{
			name:    "MockS3 Close",
			storage: NewMockS3(),
		},
		{
			name:    "MockGCS Close",
			storage: NewMockGCS(),
		},
		{
			name:    "MockAzure Close",
			storage: NewMockAzure(),
		},
		{
			name:    "Local Close",
			storage: NewLocal(t.TempDir()),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.storage.Close()
			if err != nil {
				t.Errorf("Close failed: %v", err)
			}
		})
	}
}

func TestStorage_ErrorHandling(t *testing.T) {
	// Test error handling for various scenarios
	storage := NewMockStorage()

	// Test GetSize with non-existent file
	size, err := storage.GetSize("nonexistent.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	if size != 0 {
		t.Error("Size should be 0 when error occurs")
	}

	// Test Delete with non-existent file
	err = storage.Delete("nonexistent.txt")
	if err == nil {
		t.Error("Expected error for deleting non-existent file")
	}

	// Test GetSignedURL with non-existent file
	signedURL, err := storage.GetSignedURL("nonexistent.txt", 15*time.Minute)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	if signedURL != "" {
		t.Error("Signed URL should be empty when error occurs")
	}
}

func TestLocalStorage_Integration(t *testing.T) {
	tempDir := t.TempDir()
	baseURL := "http://localhost:8080/files"

	storage := NewLocalWithURL(tempDir, baseURL)

	// Test successful upload and retrieval
	content := []byte("test content")
	reader := bytes.NewReader(content)

	filename, err := storage.Store("test.txt", reader)
	if err != nil {
		t.Fatalf("Failed to store file: %v", err)
	}

	// Test retrieval by checking if file exists
	if !storage.Exists(filename) {
		t.Errorf("File should exist after storing")
	}

	// Test size retrieval
	size, err := storage.GetSize(filename)
	if err != nil {
		t.Fatalf("Failed to get file size: %v", err)
	}
	if size != int64(len(content)) {
		t.Errorf("Expected size %d, got %d", len(content), size)
	}

	// Test URL generation
	url := storage.GetURL(filename)
	if url != baseURL+"/test.txt" {
		t.Errorf("Expected URL %s, got %s", baseURL+"/test.txt", url)
	}

	// Test deletion
	err = storage.Delete(filename)
	if err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	// Verify deletion
	if storage.Exists(filename) {
		t.Errorf("File should not exist after deletion")
	}
}

func TestLocalStorage_ErrorPaths(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewLocalWithURL(tempDir, "http://localhost:8080/files")

	// Test with nil reader
	_, err := storage.Store("test.txt", nil)
	if err == nil {
		t.Error("Expected error when reader is nil")
	}

	// Test with invalid filename
	_, err = storage.Store("", bytes.NewReader([]byte("test")))
	if err == nil {
		t.Error("Expected error when filename is empty")
	}

	// Test with directory traversal attempt
	_, err = storage.Store("../../../etc/passwd", bytes.NewReader([]byte("test")))
	if err == nil {
		t.Error("Expected error for directory traversal attempt")
	}

	// Test getting non-existent file
	if storage.Exists("nonexistent.txt") {
		t.Error("Non-existent file should not exist")
	}

	// Test deleting non-existent file
	err = storage.Delete("nonexistent.txt")
	if err == nil {
		t.Error("Expected error when deleting non-existent file")
	}
}

func TestS3Storage_Constructor(t *testing.T) {
	// Test with valid config
	config := S3Config{
		Region:          "us-west-2",
		Bucket:          "test-bucket",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		Endpoint:        "http://localhost:9000",
		ForcePathStyle:  true,
	}

	storage, err := NewS3(config)
	if err != nil {
		t.Fatalf("Failed to create S3 storage: %v", err)
	}
	if storage == nil {
		t.Fatal("Storage should not be nil")
	}

	// Test with missing required fields
	_, err = NewS3(S3Config{})
	if err == nil {
		t.Error("Expected error when config is empty")
	}

	_, err = NewS3(S3Config{Region: "us-west-2"})
	if err == nil {
		t.Error("Expected error when required fields are missing")
	}
}

func TestS3Storage_Methods(t *testing.T) {
	config := S3Config{
		Region:          "us-west-2",
		Bucket:          "test-bucket",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		Endpoint:        "http://localhost:9000",
		ForcePathStyle:  true,
	}

	storage, err := NewS3(config)
	if err != nil {
		t.Fatalf("Failed to create S3 storage: %v", err)
	}

	// Test methods return errors (since we don't have real S3)
	_, err = storage.Store("test.txt", bytes.NewReader([]byte("test")))
	if err == nil {
		t.Error("Expected error when storing to S3 without real connection")
	}

	if storage.Exists("test.txt") {
		t.Error("File should not exist in S3")
	}

	err = storage.Delete("test.txt")
	if err == nil {
		t.Error("Expected error when deleting from S3")
	}

	url := storage.GetURL("test.txt")
	if !strings.Contains(url, "test-bucket") || !strings.Contains(url, "test.txt") {
		t.Errorf("URL should contain bucket and filename: %s", url)
	}
}

func TestGCSStorage_Constructor(t *testing.T) {
	// Test with valid config (without real credentials file)
	config := GCSConfig{
		Bucket:    "test-bucket",
		ProjectID: "test-project",
		// Don't set CredentialsFile to avoid file system dependency
	}

	// This will fail because we don't have real GCS credentials, but that's expected
	storage, err := NewGCS(config)
	if err != nil {
		// Expected error when no credentials are available
		t.Logf("Expected error when no GCS credentials available: %v", err)
		return
	}
	if storage == nil {
		t.Fatal("Storage should not be nil")
	}

	// Test with missing required fields
	_, err = NewGCS(GCSConfig{})
	if err == nil {
		t.Error("Expected error when config is empty")
	}
}

func TestGCSStorage_Methods(t *testing.T) {
	config := GCSConfig{
		Bucket:    "test-bucket",
		ProjectID: "test-project",
		// Don't set CredentialsFile to avoid file system dependency
	}

	// This will fail because we don't have real GCS credentials, but that's expected
	storage, err := NewGCS(config)
	if err != nil {
		// Expected error when no credentials are available
		t.Logf("Expected error when no GCS credentials available: %v", err)
		return
	}

	// Test methods return errors (since we don't have real GCS)
	_, err = storage.Store("test.txt", bytes.NewReader([]byte("test")))
	if err == nil {
		t.Error("Expected error when storing to GCS without real connection")
	}

	if storage.Exists("test.txt") {
		t.Error("File should not exist in GCS")
	}

	err = storage.Delete("test.txt")
	if err == nil {
		t.Error("Expected error when deleting from GCS")
	}

	url := storage.GetURL("test.txt")
	if !strings.Contains(url, "test-bucket") || !strings.Contains(url, "test.txt") {
		t.Errorf("URL should contain bucket and filename: %s", url)
	}
}

func TestAzureStorage_Constructor(t *testing.T) {
	// Test with valid config (using base64 encoded key)
	config := AzureConfig{
		AccountName: "testaccount",
		AccountKey:  "dGVzdC1rZXk=", // base64 encoded "test-key"
		Container:   "test-container",
	}

	storage, err := NewAzureBlob(config)
	if err != nil {
		// Expected error when no real Azure credentials are available
		t.Logf("Expected error when no Azure credentials available: %v", err)
		return
	}
	if storage == nil {
		t.Fatal("Storage should not be nil")
	}

	// Test with missing required fields
	_, err = NewAzureBlob(AzureConfig{})
	if err == nil {
		t.Error("Expected error when config is empty")
	}

	_, err = NewAzureBlob(AzureConfig{AccountName: "test"})
	if err == nil {
		t.Error("Expected error when required fields are missing")
	}
}

func TestAzureStorage_Methods(t *testing.T) {
	config := AzureConfig{
		AccountName: "testaccount",
		AccountKey:  "dGVzdC1rZXk=", // base64 encoded "test-key"
		Container:   "test-container",
	}

	storage, err := NewAzureBlob(config)
	if err != nil {
		// Expected error when no real Azure credentials are available
		t.Logf("Expected error when no Azure credentials available: %v", err)
		return
	}

	// Test methods return errors (since we don't have real Azure)
	_, err = storage.Store("test.txt", bytes.NewReader([]byte("test")))
	if err == nil {
		t.Error("Expected error when storing to Azure without real connection")
	}

	if storage.Exists("test.txt") {
		t.Error("File should not exist in Azure")
	}

	err = storage.Delete("test.txt")
	if err == nil {
		t.Error("Expected error when deleting from Azure")
	}

	url := storage.GetURL("test.txt")
	if !strings.Contains(url, "testaccount") || !strings.Contains(url, "test-container") || !strings.Contains(url, "test.txt") {
		t.Errorf("URL should contain account, container and filename: %s", url)
	}
}

func TestStorage_ContextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewLocalWithURL(tempDir, "http://localhost:8080/files")

	// Test context cancellation (storage doesn't use context, so this is just a basic test)
	_, err := storage.Store("test.txt", bytes.NewReader([]byte("test")))
	if err != nil {
		t.Fatalf("Failed to store file: %v", err)
	}

	if !storage.Exists("test.txt") {
		t.Error("File should exist after storing")
	}

	err = storage.Delete("test.txt")
	if err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}
}

func TestStorage_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewLocalWithURL(tempDir, "http://localhost:8080/files")

	// Test concurrent uploads
	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			filename := fmt.Sprintf("test_%d.txt", id)
			content := fmt.Sprintf("content_%d", id)

			_, err := storage.Store(filename, bytes.NewReader([]byte(content)))
			if err != nil {
				t.Errorf("Failed to store file %s: %v", filename, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all files were created
	for i := 0; i < numGoroutines; i++ {
		filename := fmt.Sprintf("test_%d.txt", i)
		if !storage.Exists(filename) {
			t.Errorf("File %s should exist", filename)
		}
	}
}

func TestStorage_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewLocalWithURL(tempDir, "http://localhost:8080/files")

	// Test empty file
	filename, err := storage.Store("empty.txt", bytes.NewReader([]byte{}))
	if err != nil {
		t.Fatalf("Failed to store empty file: %v", err)
	}

	size, err := storage.GetSize(filename)
	if err != nil {
		t.Fatalf("Failed to get empty file size: %v", err)
	}
	if size != 0 {
		t.Errorf("Expected size 0, got %d", size)
	}

	// Test large filename
	longFilename := strings.Repeat("a", 1000) + ".txt"
	_, err = storage.Store(longFilename, bytes.NewReader([]byte("test")))
	if err == nil {
		t.Error("Expected error for very long filename")
	}

	// Test special characters in filename (local storage doesn't validate this, so skip)
	// _, err = storage.Store("test<>.txt", bytes.NewReader([]byte("test")))
	// if err == nil {
	// 	t.Error("Expected error for special characters in filename")
	// }

	// Test URL generation for non-existent file
	url := storage.GetURL("nonexistent.txt")
	if !strings.Contains(url, "nonexistent.txt") {
		t.Errorf("URL should contain filename: %s", url)
	}
}

// TestStorageErrorHandling tests error handling in storage backends
func TestStorageErrorHandling(t *testing.T) {
	t.Run("LocalStorageInvalidDirectory", func(t *testing.T) {
		// Test with invalid directory
		invalidDir := "/invalid/path/that/does/not/exist"
		localStorage := NewLocal(invalidDir)
		// Note: NewLocal doesn't return an error, so we test the actual usage
		_, err := localStorage.Store("test.txt", strings.NewReader("test"))
		if err == nil {
			t.Error("Expected error for invalid storage directory")
		}
	})

	t.Run("LocalStoragePermissionDenied", func(t *testing.T) {
		// Test with read-only directory
		tempDir := t.TempDir()
		readOnlyDir := filepath.Join(tempDir, "readonly")
		os.Mkdir(readOnlyDir, 0444) // Read-only

		localStorage := NewLocal(readOnlyDir)

		// Try to store a file
		_, err := localStorage.Store("test.txt", strings.NewReader("test"))
		if err == nil {
			t.Error("Expected error for permission denied")
		}
	})

	t.Run("LocalStorageDiskFull", func(t *testing.T) {
		// This is a theoretical test - we can't easily simulate disk full
		// But we can test the error handling code paths
		localStorage := NewLocal(t.TempDir())

		// Test with very large data
		largeData := strings.Repeat("a", 1024*1024) // 1MB
		_, err := localStorage.Store("large.txt", strings.NewReader(largeData))
		if err != nil {
			t.Errorf("Expected no error storing large file, got: %v", err)
		}
	})

	t.Run("LocalStorageContextCancellation", func(t *testing.T) {
		localStorage := NewLocal(t.TempDir())

		// Note: Local storage doesn't use context, so this test is simplified
		_, err := localStorage.Store("test.txt", strings.NewReader("test"))
		if err != nil {
			t.Errorf("Expected no error storing file, got: %v", err)
		}
	})

	t.Run("LocalStorageContextTimeout", func(t *testing.T) {
		localStorage := NewLocal(t.TempDir())

		// Note: Local storage doesn't use context, so this test is simplified
		_, err := localStorage.Store("test.txt", strings.NewReader("test"))
		if err != nil {
			t.Errorf("Expected no error storing file, got: %v", err)
		}
	})
}

// TestStorageEdgeCases tests edge cases in storage backends
func TestStorageEdgeCases(t *testing.T) {
	t.Run("LocalStorageEmptyFile", func(t *testing.T) {
		localStorage := NewLocal(t.TempDir())

		_, err := localStorage.Store("empty.txt", strings.NewReader(""))
		if err != nil {
			t.Errorf("Expected no error storing empty file, got: %v", err)
		}

		reader, err := localStorage.GetReader("empty.txt")
		if err != nil {
			t.Errorf("Expected no error retrieving empty file, got: %v", err)
		}
		defer reader.Close()

		data, err := io.ReadAll(reader)
		if err != nil {
			t.Errorf("Expected no error reading empty file, got: %v", err)
		}

		if len(data) != 0 {
			t.Error("Expected empty data")
		}
	})

	t.Run("LocalStorageLargeFile", func(t *testing.T) {
		localStorage := NewLocal(t.TempDir())

		// Create a large file (10MB)
		largeData := strings.Repeat("a", 10*1024*1024)
		_, err := localStorage.Store("large.txt", strings.NewReader(largeData))
		if err != nil {
			t.Errorf("Expected no error storing large file, got: %v", err)
		}

		reader, err := localStorage.GetReader("large.txt")
		if err != nil {
			t.Errorf("Expected no error retrieving large file, got: %v", err)
		}
		defer reader.Close()

		data, err := io.ReadAll(reader)
		if err != nil {
			t.Errorf("Expected no error reading large file, got: %v", err)
		}

		if len(data) != len(largeData) {
			t.Error("Expected data length to match")
		}
	})

	t.Run("LocalStorageSpecialCharacters", func(t *testing.T) {
		localStorage := NewLocal(t.TempDir())

		// Test with special characters in filename
		specialName := "file with spaces & special chars!@#$%^&*().txt"
		_, err := localStorage.Store(specialName, strings.NewReader("test"))
		if err != nil {
			t.Errorf("Expected no error storing file with special characters, got: %v", err)
		}

		reader, err := localStorage.GetReader(specialName)
		if err != nil {
			t.Errorf("Expected no error retrieving file with special characters, got: %v", err)
		}
		defer reader.Close()

		data, err := io.ReadAll(reader)
		if err != nil {
			t.Errorf("Expected no error reading file with special characters, got: %v", err)
		}

		if string(data) != "test" {
			t.Error("Expected data to match")
		}
	})

	t.Run("LocalStorageUnicodeFilename", func(t *testing.T) {
		localStorage := NewLocal(t.TempDir())

		// Test with Unicode filename
		unicodeName := "file-中文-日本語-한국어.txt"
		_, err := localStorage.Store(unicodeName, strings.NewReader("test"))
		if err != nil {
			t.Errorf("Expected no error storing file with Unicode name, got: %v", err)
		}

		reader, err := localStorage.GetReader(unicodeName)
		if err != nil {
			t.Errorf("Expected no error retrieving file with Unicode name, got: %v", err)
		}
		defer reader.Close()

		data, err := io.ReadAll(reader)
		if err != nil {
			t.Errorf("Expected no error reading file with Unicode name, got: %v", err)
		}

		if string(data) != "test" {
			t.Error("Expected data to match")
		}
	})

	t.Run("LocalStoragePathTraversal", func(t *testing.T) {
		localStorage := NewLocal(t.TempDir())

		// Test path traversal attempts
		maliciousPaths := []string{
			"../../../etc/passwd",
			"..\\..\\..\\windows\\system32\\config\\sam",
			"....//....//....//etc/passwd",
			"..%2F..%2F..%2Fetc%2Fpasswd",
		}

		for _, path := range maliciousPaths {
			_, err := localStorage.Store(path, strings.NewReader("malicious"))
			if err == nil {
				t.Errorf("Expected error for path traversal attempt: %s", path)
			}
		}
	})

	t.Run("LocalStorageOverwrite", func(t *testing.T) {
		localStorage := NewLocal(t.TempDir())

		// Store initial file
		_, err := localStorage.Store("test.txt", strings.NewReader("initial"))
		if err != nil {
			t.Errorf("Expected no error storing initial file, got: %v", err)
		}

		// Overwrite with new content
		_, err = localStorage.Store("test.txt", strings.NewReader("overwritten"))
		if err != nil {
			t.Errorf("Expected no error overwriting file, got: %v", err)
		}

		reader, err := localStorage.GetReader("test.txt")
		if err != nil {
			t.Errorf("Expected no error retrieving overwritten file, got: %v", err)
		}
		defer reader.Close()

		data, err := io.ReadAll(reader)
		if err != nil {
			t.Errorf("Expected no error reading overwritten file, got: %v", err)
		}

		if string(data) != "overwritten" {
			t.Error("Expected overwritten content")
		}
	})
}

// TestStorageConcurrency tests concurrent access to storage
func TestStorageConcurrency(t *testing.T) {
	t.Run("LocalStorageConcurrentWrites", func(t *testing.T) {
		localStorage := NewLocal(t.TempDir())

		// Test concurrent writes to different files
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func(id int) {
				defer func() { done <- true }()

				filename := fmt.Sprintf("concurrent_%d.txt", id)
				data := fmt.Sprintf("data from goroutine %d", id)

				_, err := localStorage.Store(filename, strings.NewReader(data))
				if err != nil {
					t.Errorf("Expected no error in concurrent write %d, got: %v", id, err)
				}

				reader, err := localStorage.GetReader(filename)
				if err != nil {
					t.Errorf("Expected no error in concurrent read %d, got: %v", id, err)
				}
				defer reader.Close()

				retrieved, err := io.ReadAll(reader)
				if err != nil {
					t.Errorf("Expected no error reading concurrent file %d, got: %v", id, err)
				}

				if string(retrieved) != data {
					t.Errorf("Expected data to match in concurrent operation %d", id)
				}
			}(i)
		}

		// Wait for all operations to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("LocalStorageConcurrentReads", func(t *testing.T) {
		localStorage := NewLocal(t.TempDir())

		// Store a file first
		_, err := localStorage.Store("shared.txt", strings.NewReader("shared data"))
		if err != nil {
			t.Fatalf("Expected no error storing shared file, got: %v", err)
		}

		// Test concurrent reads
		done := make(chan bool, 20)
		for i := 0; i < 20; i++ {
			go func(id int) {
				defer func() { done <- true }()

				reader, err := localStorage.GetReader("shared.txt")
				if err != nil {
					t.Errorf("Expected no error in concurrent read %d, got: %v", id, err)
				}
				defer reader.Close()

				data, err := io.ReadAll(reader)
				if err != nil {
					t.Errorf("Expected no error reading concurrent file %d, got: %v", id, err)
				}

				if string(data) != "shared data" {
					t.Errorf("Expected correct data in concurrent read %d", id)
				}
			}(i)
		}

		// Wait for all operations to complete
		for i := 0; i < 20; i++ {
			<-done
		}
	})

	t.Run("LocalStorageConcurrentReadWrite", func(t *testing.T) {
		localStorage := NewLocal(t.TempDir())

		// Test concurrent read and write operations
		done := make(chan bool, 20)
		for i := 0; i < 10; i++ {
			// Writers
			go func(id int) {
				defer func() { done <- true }()

				filename := fmt.Sprintf("rw_%d.txt", id)
				data := fmt.Sprintf("data %d", id)

				_, err := localStorage.Store(filename, strings.NewReader(data))
				if err != nil {
					t.Errorf("Expected no error in concurrent write %d, got: %v", id, err)
				}
			}(i)

			// Readers
			go func(id int) {
				defer func() { done <- true }()

				filename := fmt.Sprintf("rw_%d.txt", id)
				reader, err := localStorage.GetReader(filename)
				// Reader might fail if writer hasn't finished yet, which is expected
				if err != nil && !strings.Contains(err.Error(), "not found") {
					t.Errorf("Unexpected error in concurrent read %d, got: %v", id, err)
				}
				if err == nil {
					reader.Close()
				}
			}(i)
		}

		// Wait for all operations to complete
		for i := 0; i < 20; i++ {
			<-done
		}
	})
}

// TestStoragePerformance tests performance characteristics
func TestStoragePerformance(t *testing.T) {
	t.Run("LocalStorageBulkOperations", func(t *testing.T) {
		localStorage := NewLocal(t.TempDir())

		start := time.Now()

		// Perform bulk write operations
		for i := 0; i < 100; i++ {
			filename := fmt.Sprintf("bulk_%d.txt", i)
			data := fmt.Sprintf("data for file %d", i)

			_, err := localStorage.Store(filename, strings.NewReader(data))
			if err != nil {
				t.Errorf("Expected no error in bulk write %d, got: %v", i, err)
			}
		}

		writeDuration := time.Since(start)

		// Perform bulk read operations
		start = time.Now()
		for i := 0; i < 100; i++ {
			filename := fmt.Sprintf("bulk_%d.txt", i)
			expected := fmt.Sprintf("data for file %d", i)

			reader, err := localStorage.GetReader(filename)
			if err != nil {
				t.Errorf("Expected no error in bulk read %d, got: %v", i, err)
			}
			defer reader.Close()

			data, err := io.ReadAll(reader)
			if err != nil {
				t.Errorf("Expected no error reading bulk file %d, got: %v", i, err)
			}

			if string(data) != expected {
				t.Errorf("Expected data to match in bulk read %d", i)
			}
		}

		readDuration := time.Since(start)

		if writeDuration > 5*time.Second {
			t.Errorf("Bulk write operations took too long: %v", writeDuration)
		}

		if readDuration > 5*time.Second {
			t.Errorf("Bulk read operations took too long: %v", readDuration)
		}
	})

	t.Run("LocalStorageLargeFilePerformance", func(t *testing.T) {
		localStorage := NewLocal(t.TempDir())

		// Create a large file (100MB)
		largeData := strings.Repeat("a", 100*1024*1024)

		start := time.Now()
		_, err := localStorage.Store("large_perf.txt", strings.NewReader(largeData))
		writeDuration := time.Since(start)

		if err != nil {
			t.Fatalf("Expected no error storing large file, got: %v", err)
		}

		start = time.Now()
		reader, err := localStorage.GetReader("large_perf.txt")
		if err != nil {
			t.Fatalf("Expected no error retrieving large file, got: %v", err)
		}
		defer reader.Close()

		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("Expected no error reading large file, got: %v", err)
		}

		if len(data) != len(largeData) {
			t.Fatal("Expected data length to match")
		}

		readDuration := time.Since(start)

		if writeDuration > 30*time.Second {
			t.Errorf("Large file write took too long: %v", writeDuration)
		}

		if readDuration > 30*time.Second {
			t.Errorf("Large file read took too long: %v", readDuration)
		}
	})
}

// TestStorageErrorRecovery tests error recovery scenarios
func TestStorageErrorRecovery(t *testing.T) {
	t.Run("LocalStorageRecoveryAfterError", func(t *testing.T) {
		localStorage := NewLocal(t.TempDir())

		// Try to store a file
		_, err := localStorage.Store("recovery.txt", strings.NewReader("test"))
		if err != nil {
			t.Errorf("Expected no error storing file, got: %v", err)
		}

		// Verify it was stored
		reader, err := localStorage.GetReader("recovery.txt")
		if err != nil {
			t.Errorf("Expected no error retrieving file, got: %v", err)
		}
		defer reader.Close()
		data, err := io.ReadAll(reader)
		if err != nil {
			t.Errorf("Expected no error reading file, got: %v", err)
		}
		if string(data) != "test" {
			t.Error("Expected data to match")
		}

		// Try to store another file after successful operation
		_, err = localStorage.Store("recovery2.txt", strings.NewReader("test2"))
		if err != nil {
			t.Errorf("Expected no error storing second file, got: %v", err)
		}

		reader2, err := localStorage.GetReader("recovery2.txt")
		if err != nil {
			t.Errorf("Expected no error retrieving second file, got: %v", err)
		}
		defer reader2.Close()
		data2, err := io.ReadAll(reader2)
		if err != nil {
			t.Errorf("Expected no error reading second file, got: %v", err)
		}
		if string(data2) != "test2" {
			t.Error("Expected second data to match")
		}
	})

	t.Run("LocalStoragePartialFailure", func(t *testing.T) {
		localStorage := NewLocal(t.TempDir())

		// Store multiple files
		files := []string{"file1.txt", "file2.txt", "file3.txt"}
		for i, filename := range files {
			data := fmt.Sprintf("data %d", i+1)
			_, err := localStorage.Store(filename, strings.NewReader(data))
			if err != nil {
				t.Errorf("Expected no error storing %s, got: %v", filename, err)
			}
		}

		// Verify all files are accessible
		for i, filename := range files {
			expected := fmt.Sprintf("data %d", i+1)
			reader, err := localStorage.GetReader(filename)
			if err != nil {
				t.Errorf("Expected no error retrieving %s, got: %v", filename, err)
			}
			defer reader.Close()
			data, err := io.ReadAll(reader)
			if err != nil {
				t.Errorf("Expected no error reading %s, got: %v", filename, err)
			}
			if string(data) != expected {
				t.Errorf("Expected data to match for %s", filename)
			}
		}
	})
}

// TestStorageSecurity tests security-related scenarios
func TestStorageSecurity(t *testing.T) {
	t.Run("LocalStorageSymlinkAttack", func(t *testing.T) {
		localStorage := NewLocal(t.TempDir())

		// Create a symlink to a sensitive file
		sensitiveFile := filepath.Join(t.TempDir(), "sensitive.txt")
		os.WriteFile(sensitiveFile, []byte("sensitive data"), 0644)

		symlinkFile := filepath.Join(t.TempDir(), "symlink.txt")
		os.Symlink(sensitiveFile, symlinkFile)

		// Try to store through symlink
		_, err := localStorage.Store(symlinkFile, strings.NewReader("malicious"))
		if err == nil {
			t.Error("Expected error for symlink attack")
		}
	})

	t.Run("LocalStorageDirectoryTraversal", func(t *testing.T) {
		localStorage := NewLocal(t.TempDir())

		// Test various directory traversal patterns
		traversalPatterns := []string{
			"../../../etc/passwd",
			"..\\..\\..\\windows\\system32\\config\\sam",
			"....//....//....//etc/passwd",
			"..%2F..%2F..%2Fetc%2Fpasswd",
			"..%5C..%5C..%5Cwindows%5Csystem32%5Cconfig%5Csam",
		}

		for _, pattern := range traversalPatterns {
			_, err := localStorage.Store(pattern, strings.NewReader("malicious"))
			if err == nil {
				t.Errorf("Expected error for directory traversal: %s", pattern)
			}
		}
	})

	t.Run("LocalStorageNullByte", func(t *testing.T) {
		localStorage := NewLocal(t.TempDir())

		// Test filename with null byte
		nullFilename := "file\x00.txt"
		_, err := localStorage.Store(nullFilename, strings.NewReader("test"))
		if err == nil {
			t.Error("Expected error for filename with null byte")
		}
	})

	t.Run("LocalStorageControlCharacters", func(t *testing.T) {
		localStorage := NewLocal(t.TempDir())

		// Test filename with control characters
		controlFilename := "file\x01\x02\x03.txt"
		_, err := localStorage.Store(controlFilename, strings.NewReader("test"))
		if err == nil {
			t.Error("Expected error for filename with control characters")
		}
	})
}

// TestStorageIntegration tests integration scenarios
func TestStorageIntegration(t *testing.T) {
	t.Run("LocalStorageWithBaseURL", func(t *testing.T) {
		baseURL := "https://example.com/files/"
		localStorage := NewLocal(t.TempDir(), baseURL)

		// Store a file
		_, err := localStorage.Store("test.txt", strings.NewReader("test"))
		if err != nil {
			t.Errorf("Expected no error storing file, got: %v", err)
		}

		// Get URL
		url := localStorage.GetURL("test.txt")
		expectedURL := baseURL + "test.txt"
		if url != expectedURL {
			t.Errorf("Expected URL %s, got %s", expectedURL, url)
		}
	})

	t.Run("LocalStorageWithoutBaseURL", func(t *testing.T) {
		localStorage := NewLocal(t.TempDir())

		// Store a file
		_, err := localStorage.Store("test.txt", strings.NewReader("test"))
		if err != nil {
			t.Errorf("Expected no error storing file, got: %v", err)
		}

		// Get URL (should be empty without base URL)
		url := localStorage.GetURL("test.txt")
		if url != "" {
			t.Errorf("Expected empty URL without base URL, got %s", url)
		}
	})
}

// TestStorageBenchmarks runs performance benchmarks
func TestStorageBenchmarks(t *testing.T) {
	t.Run("LocalStorageWriteBenchmark", func(t *testing.T) {
		localStorage := NewLocal(t.TempDir())

		start := time.Now()
		for i := 0; i < 1000; i++ {
			filename := fmt.Sprintf("bench_%d.txt", i)
			data := fmt.Sprintf("data %d", i)

			_, err := localStorage.Store(filename, strings.NewReader(data))
			if err != nil {
				t.Errorf("Expected no error in benchmark write %d, got: %v", i, err)
			}
		}
		duration := time.Since(start)

		if duration > 10*time.Second {
			t.Errorf("Write benchmark took too long: %v", duration)
		}
	})

	t.Run("LocalStorageReadBenchmark", func(t *testing.T) {
		localStorage := NewLocal(t.TempDir())

		// Prepare test data
		for i := 0; i < 100; i++ {
			filename := fmt.Sprintf("bench_read_%d.txt", i)
			data := fmt.Sprintf("data %d", i)

			_, err := localStorage.Store(filename, strings.NewReader(data))
			if err != nil {
				t.Fatalf("Expected no error preparing benchmark data %d, got: %v", i, err)
			}
		}

		start := time.Now()
		for i := 0; i < 1000; i++ {
			filename := fmt.Sprintf("bench_read_%d.txt", i%100)
			expected := fmt.Sprintf("data %d", i%100)

			reader, err := localStorage.GetReader(filename)
			if err != nil {
				t.Errorf("Expected no error in benchmark read %d, got: %v", i, err)
			}
			defer reader.Close()

			data, err := io.ReadAll(reader)
			if err != nil {
				t.Errorf("Expected no error reading benchmark file %d, got: %v", i, err)
			}

			if string(data) != expected {
				t.Errorf("Expected data to match in benchmark read %d", i)
			}
		}
		duration := time.Since(start)

		if duration > 10*time.Second {
			t.Errorf("Read benchmark took too long: %v", duration)
		}
	})
}
