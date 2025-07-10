package upload

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kdsmith18542/gokit/upload/storage"
)

func TestNewProcessor(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	options := Options{
		MaxFileSize:      10 * 1024 * 1024, // 10MB
		AllowedMIMETypes: []string{"image/jpeg", "image/png"},
	}

	processor := NewProcessor(mockStorage, options)

	if processor == nil {
		t.Fatal("Expected processor to be created")
	}

	if processor.storage != mockStorage {
		t.Error("Expected storage to be set")
	}

	if processor.options.MaxFileSize != options.MaxFileSize {
		t.Error("Expected options to be set")
	}

	if len(processor.onSuccess) != 0 {
		t.Error("Expected empty onSuccess hooks")
	}

	if len(processor.onError) != 0 {
		t.Error("Expected empty onError hooks")
	}
}

func TestOnSuccessHook(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	processor := NewProcessor(mockStorage, Options{})

	var hookCalled bool
	var capturedResult Result

	processor.OnSuccess(func(ctx context.Context, result Result) {
		hookCalled = true
		capturedResult = result
	})

	if len(processor.onSuccess) != 1 {
		t.Error("Expected one success hook to be registered")
	}

	// Test hook execution
	ctx := context.Background()
	testResult := Result{
		OriginalName: "test.jpg",
		Size:         1024,
		URL:          "http://example.com/test.jpg",
	}

	processor.onSuccess[0](ctx, testResult)

	if !hookCalled {
		t.Error("Expected hook to be called")
	}

	if capturedResult.OriginalName != testResult.OriginalName {
		t.Errorf("Expected original name %s, got %s", testResult.OriginalName, capturedResult.OriginalName)
	}
}

func TestOnErrorHook(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	processor := NewProcessor(mockStorage, Options{})

	var hookCalled bool
	var capturedResult Result
	var capturedError error

	processor.OnError(func(ctx context.Context, result Result, err error) {
		hookCalled = true
		capturedResult = result
		capturedError = err
	})

	if len(processor.onError) != 1 {
		t.Error("Expected one error hook to be registered")
	}

	// Test hook execution
	ctx := context.Background()
	testResult := Result{
		OriginalName: "test.jpg",
		Size:         1024,
	}
	testError := fmt.Errorf("test error")

	processor.onError[0](ctx, testResult, testError)

	if !hookCalled {
		t.Error("Expected hook to be called")
	}

	if capturedResult.OriginalName != testResult.OriginalName {
		t.Errorf("Expected original name %s, got %s", testResult.OriginalName, capturedResult.OriginalName)
	}

	if capturedError != testError {
		t.Error("Expected error to be captured")
	}
}

func TestMultipleHooks(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	processor := NewProcessor(mockStorage, Options{})

	var hook1Called, hook2Called bool

	processor.OnSuccess(func(ctx context.Context, result Result) {
		hook1Called = true
	})

	processor.OnSuccess(func(ctx context.Context, result Result) {
		hook2Called = true
	})

	if len(processor.onSuccess) != 2 {
		t.Error("Expected two success hooks to be registered")
	}

	// Test hook execution
	ctx := context.Background()
	testResult := Result{OriginalName: "test.jpg"}

	processor.onSuccess[0](ctx, testResult)
	processor.onSuccess[1](ctx, testResult)

	if !hook1Called {
		t.Error("Expected first hook to be called")
	}

	if !hook2Called {
		t.Error("Expected second hook to be called")
	}
}

func TestProcessWithHooks(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	processor := NewProcessor(mockStorage, Options{
		MaxFileSize:      10 * 1024 * 1024,
		AllowedMIMETypes: []string{"image/jpeg", "application/octet-stream"},
	})

	var successHookCalled bool
	var errorHookCalled bool
	var capturedResult Result

	processor.OnSuccess(func(ctx context.Context, result Result) {
		successHookCalled = true
		capturedResult = result
	})

	processor.OnError(func(ctx context.Context, result Result, err error) {
		errorHookCalled = true
	})

	// Create a test request with a file
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "test.jpg")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	if _, err := part.Write([]byte("test file content")); err != nil {
		t.Fatalf("Failed to write test file content: %v", err)
	}
	writer.Close()

	req := httptest.NewRequest("POST", "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Process the upload
	results, err := processor.Process(req, "file")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !successHookCalled {
		t.Error("Expected success hook to be called")
	}

	if errorHookCalled {
		t.Error("Expected error hook not to be called")
	}

	if capturedResult.OriginalName != "test.jpg" {
		t.Errorf("Expected original name 'test.jpg', got '%s'", capturedResult.OriginalName)
	}
}

func TestProcessWithContext(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	processor := NewProcessor(mockStorage, Options{
		MaxFileSize:      10 * 1024 * 1024,
		AllowedMIMETypes: []string{"image/jpeg", "application/octet-stream"},
	})

	var hookCalled bool
	var capturedContext context.Context

	processor.OnSuccess(func(ctx context.Context, result Result) {
		hookCalled = true
		capturedContext = ctx
	})

	// Create a test request with a file
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "test.jpg")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	if _, err := part.Write([]byte("test file content")); err != nil {
		t.Fatalf("Failed to write test file content: %v", err)
	}
	writer.Close()

	req := httptest.NewRequest("POST", "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Process the upload with context
	results, err := processor.ProcessWithContext(ctx, req, "file")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !hookCalled {
		t.Error("Expected success hook to be called")
	}

	if capturedContext != ctx {
		t.Error("Expected hook to receive the same context")
	}
}

func TestProcessSingleWithContext(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	processor := NewProcessor(mockStorage, Options{
		MaxFileSize:      10 * 1024 * 1024,
		AllowedMIMETypes: []string{"image/jpeg", "application/octet-stream"},
	})

	var hookCalled bool

	processor.OnSuccess(func(ctx context.Context, result Result) {
		hookCalled = true
	})

	// Create a test request with a file
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "test.jpg")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	if _, err := part.Write([]byte("test file content")); err != nil {
		t.Fatalf("Failed to write test file content: %v", err)
	}
	writer.Close()

	req := httptest.NewRequest("POST", "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Create a context
	ctx := context.Background()

	// Process the upload with context
	result, err := processor.ProcessSingleWithContext(ctx, req, "file")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be returned")
	}

	if result.OriginalName != "test.jpg" {
		t.Errorf("Expected original name 'test.jpg', got '%s'", result.OriginalName)
	}

	if !hookCalled {
		t.Error("Expected success hook to be called")
	}
}

func TestHookExecutionOrder(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	processor := NewProcessor(mockStorage, Options{
		MaxFileSize:      10 * 1024 * 1024,
		AllowedMIMETypes: []string{"image/jpeg", "application/octet-stream"},
	})

	var executionOrder []int

	processor.OnSuccess(func(ctx context.Context, result Result) {
		executionOrder = append(executionOrder, 1)
	})

	processor.OnSuccess(func(ctx context.Context, result Result) {
		executionOrder = append(executionOrder, 2)
	})

	processor.OnSuccess(func(ctx context.Context, result Result) {
		executionOrder = append(executionOrder, 3)
	})

	// Create a test request with a file
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "test.jpg")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	if _, err := part.Write([]byte("test file content")); err != nil {
		t.Fatalf("Failed to write test file content: %v", err)
	}
	writer.Close()

	req := httptest.NewRequest("POST", "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Process the upload
	_, err = processor.Process(req, "file")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check execution order
	expectedOrder := []int{1, 2, 3}
	if len(executionOrder) != len(expectedOrder) {
		t.Errorf("Expected %d hook executions, got %d", len(expectedOrder), len(executionOrder))
	}

	for i, expected := range expectedOrder {
		if i >= len(executionOrder) {
			break
		}
		if executionOrder[i] != expected {
			t.Errorf("Expected execution order %d at position %d, got %d", expected, i, executionOrder[i])
		}
	}
}

func TestHookConcurrency(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	processor := NewProcessor(mockStorage, Options{
		MaxFileSize:      10 * 1024 * 1024,
		AllowedMIMETypes: []string{"image/jpeg", "application/octet-stream"},
	})

	var mu sync.Mutex
	var hookCallCount int

	processor.OnSuccess(func(ctx context.Context, result Result) {
		mu.Lock()
		hookCallCount++
		mu.Unlock()
		// Simulate some work
		time.Sleep(10 * time.Millisecond)
	})

	// Create multiple test requests
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			var buf bytes.Buffer
			writer := multipart.NewWriter(&buf)
			part, err := writer.CreateFormFile("file", "test.jpg")
			if err != nil {
				t.Errorf("Failed to create form file: %v", err)
				return
			}
			if _, err := part.Write([]byte("test file content")); err != nil {
				t.Errorf("Failed to write part content for TestHookConcurrency: %v", err)
				return
			}
			if err := writer.Close(); err != nil {
				t.Errorf("Failed to close writer for TestHookConcurrency: %v", err)
				return
			}

			req := httptest.NewRequest("POST", "/upload", &buf)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			_, err = processor.Process(req, "file")
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		}()
	}

	wg.Wait()

	mu.Lock()
	finalCount := hookCallCount
	mu.Unlock()

	if finalCount != 5 {
		t.Errorf("Expected 5 hook executions, got %d", finalCount)
	}
}

func TestProcessValidation(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	processor := NewProcessor(mockStorage, Options{
		MaxFileSize:      1024, // 1KB limit
		AllowedMIMETypes: []string{"image/jpeg"},
	})

	var errorHookCalled bool
	var capturedError error

	processor.OnError(func(ctx context.Context, result Result, err error) {
		errorHookCalled = true
		capturedError = err
	})

	// Create a test request with a file that's too large
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "test.jpg")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	// Write more than 1KB
	if _, err := part.Write(make([]byte, 2048)); err != nil {
		t.Fatalf("Failed to write part content for TestProcessValidation: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close writer for TestProcessValidation: %v", err)
	}

	req := httptest.NewRequest("POST", "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Process the upload
	_, err = processor.Process(req, "file")
	if err == nil {
		t.Fatal("Expected error for file too large")
	}

	if !errorHookCalled {
		t.Error("Expected error hook to be called")
	}

	if capturedError == nil {
		t.Error("Expected error to be captured in hook")
	}

	if !strings.Contains(capturedError.Error(), "file too large") {
		t.Errorf("Expected 'file too large' error, got: %v", capturedError)
	}
}

func TestProcessNoFiles(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	processor := NewProcessor(mockStorage, Options{})

	var errorHookCalled bool

	processor.OnError(func(ctx context.Context, result Result, err error) {
		errorHookCalled = true
	})

	// Create a test request without files
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close writer for TestProcessNoFiles: %v", err)
	}

	req := httptest.NewRequest("POST", "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Process the upload
	_, err := processor.Process(req, "file")
	if err == nil {
		t.Fatal("Expected error for no files")
	}

	// Error hooks are only called for file processing errors, not for "no files" errors
	if errorHookCalled {
		t.Error("Expected error hook not to be called for 'no files' error")
	}

	if !strings.Contains(err.Error(), "no files found") {
		t.Errorf("Expected 'no files found' error, got: %v", err)
	}
}
