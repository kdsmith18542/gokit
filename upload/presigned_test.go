package upload

import (
	"context"
	"testing"
	"time"

	"github.com/kdsmith18542/gokit/upload/storage"
)

func TestPreSignedURL(t *testing.T) {
	// Create a mock storage for testing
	mockStorage := storage.NewMockStorage()
	processor := NewProcessor(mockStorage, Options{
		MaxFileSize: 10 * 1024 * 1024,
	})

	ctx := context.Background()

	// Test GenerateUploadURL
	opts := PresignedOptions{
		Filename:    "test.txt",
		ContentType: "text/plain",
		Expiration:  15 * time.Minute,
	}

	result, err := processor.GenerateUploadURL(ctx, opts)
	// Mock storage doesn't support pre-signed URLs, so expect an error
	if err == nil {
		t.Error("Expected error for mock storage pre-signed URL generation")
	}
	if result != nil {
		t.Error("Result should be nil when error occurs")
	}

	// Test GenerateUploadURLs
	results, err := processor.GenerateUploadURLs(ctx, []PresignedOptions{opts})
	// Mock storage doesn't support pre-signed URLs, so expect an error
	if err == nil {
		t.Error("Expected error for mock storage pre-signed URL generation")
	}
	if results != nil {
		t.Error("Results should be nil when error occurs")
	}

	// Test ValidatePreSignedUpload
	err = processor.ValidatePreSignedUpload(ctx, "test.txt", 1024, "text/plain")
	if err != nil {
		t.Errorf("ValidatePreSignedUpload failed: %v", err)
	}

	// Test GetStatus
	status, err := processor.GetStatus(ctx, "test.txt")
	if err != nil {
		t.Errorf("GetStatus failed: %v", err)
	}
	if status == nil {
		t.Error("Upload status should not be nil")
	}
}

func TestPresignedOptions(t *testing.T) {
	opts := PresignedOptions{
		Filename:    "test.txt",
		ContentType: "text/plain",
		Expiration:  15 * time.Minute,
		MaxFileSize: 1024 * 1024,
		Metadata: map[string]string{
			"user-id": "123",
		},
	}

	if opts.Filename != "test.txt" {
		t.Error("Filename not set correctly")
	}
	if opts.ContentType != "text/plain" {
		t.Error("ContentType not set correctly")
	}
	if opts.Expiration != 15*time.Minute {
		t.Error("Expiration not set correctly")
	}
	if opts.MaxFileSize != 1024*1024 {
		t.Error("MaxFileSize not set correctly")
	}
	if opts.Metadata["user-id"] != "123" {
		t.Error("Metadata not set correctly")
	}
}
