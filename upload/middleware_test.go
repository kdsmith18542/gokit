package upload

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kdsmith18542/gokit/upload/storage"
)

func TestUploadMiddleware(t *testing.T) {
	// Create mock storage
	mockStorage := storage.NewMockStorage()

	// Create upload processor
	processor := NewProcessor(mockStorage, Options{
		MaxFileSize:      10 * 1024 * 1024, // 10MB
		AllowedMIMETypes: []string{"image/jpeg", "image/png", "application/octet-stream"},
	})

	// Test successful upload
	t.Run("successful upload", func(t *testing.T) {
		middleware := UploadMiddleware(processor, "file", nil)

		// Create multipart request
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
		w := httptest.NewRecorder()

		var capturedResults []Result
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedResults = UploadResultsFromContext(r.Context())
		})

		middleware(handler).ServeHTTP(w, req)

		// Check response
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Check results were captured
		if len(capturedResults) == 0 {
			t.Error("Upload results not found in context")
		} else {
			result := capturedResults[0]
			if result.OriginalName != "test.jpg" {
				t.Errorf("Expected original name 'test.jpg', got '%s'", result.OriginalName)
			}
			if result.Size != 17 {
				t.Errorf("Expected size 17, got %d", result.Size)
			}
		}
	})

	// Test upload error
	t.Run("upload error", func(t *testing.T) {
		// Create processor with restrictive options
		restrictiveProcessor := NewProcessor(mockStorage, Options{
			MaxFileSize:      1024, // 1KB limit
			AllowedMIMETypes: []string{"image/jpeg", "image/png", "application/octet-stream"},
		})

		middleware := UploadMiddleware(restrictiveProcessor, "file", nil)

		// Create large file
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		part, err := writer.CreateFormFile("file", "large.jpg")
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}
		if _, err := part.Write(make([]byte, 2048)); err != nil {
			t.Fatalf("Failed to write large file content: %v", err)
		}
		writer.Close()

		req := httptest.NewRequest("POST", "/upload", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called for upload error")
		})

		middleware(handler).ServeHTTP(w, req)

		// Check error response
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		// Check JSON response
		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}

		if response["error"] != "Upload failed" {
			t.Errorf("Expected error 'Upload failed', got '%v'", response["error"])
		}
	})
}

func TestSingleUploadMiddleware(t *testing.T) {
	// Create mock storage
	mockStorage := storage.NewMockStorage()

	// Create upload processor
	processor := NewProcessor(mockStorage, Options{
		MaxFileSize:      10 * 1024 * 1024,
		AllowedMIMETypes: []string{"image/jpeg", "image/png", "application/octet-stream"},
	})

	middleware := SingleUploadMiddleware(processor, "file", nil)

	// Create multipart request
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
	w := httptest.NewRecorder()

	var capturedResult *Result
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedResult = SingleUploadResultFromContext(r.Context())
	})

	middleware(handler).ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check result was captured
	if capturedResult == nil {
		t.Error("Single upload result not found in context")
	} else {
		if capturedResult.OriginalName != "test.jpg" {
			t.Errorf("Expected original name 'test.jpg', got '%s'", capturedResult.OriginalName)
		}
		if capturedResult.Size != 17 {
			t.Errorf("Expected size 17, got %d", capturedResult.Size)
		}
	}
}

func TestUploadMiddlewareWithContext(t *testing.T) {
	// Create mock storage
	mockStorage := storage.NewMockStorage()

	// Create upload processor
	processor := NewProcessor(mockStorage, Options{
		MaxFileSize:      10 * 1024 * 1024,
		AllowedMIMETypes: []string{"image/jpeg", "image/png", "application/octet-stream"},
	})

	middleware := UploadMiddlewareWithContext(processor, "file", nil)

	// Create multipart request
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
	w := httptest.NewRecorder()

	var capturedResults []Result
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedResults = UploadResultsFromContext(r.Context())
	})

	middleware(handler).ServeHTTP(w, req)

	// Check results were captured
	if len(capturedResults) == 0 {
		t.Error("Upload results not found in context")
	}
}

func TestUploadResultsFromContext(t *testing.T) {
	// Test with nil context
	results := UploadResultsFromContext(context.Background())
	if results != nil {
		t.Error("Expected nil results for empty context")
	}

	// Test with context containing results
	testResults := []Result{
		{OriginalName: "test.jpg", Size: 1024},
	}
	ctx := context.WithValue(context.Background(), "upload_results", testResults)

	results = UploadResultsFromContext(ctx)
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestMustUploadResultsFromContext(t *testing.T) {
	// Test panic case
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when results not found in context")
		}
	}()

	MustUploadResultsFromContext(context.Background())
}

func TestMustUploadResultsFromContextSuccess(t *testing.T) {
	// Test success case
	testResults := []Result{
		{OriginalName: "test.jpg", Size: 1024},
	}
	ctx := context.WithValue(context.Background(), "upload_results", testResults)

	results := MustUploadResultsFromContext(ctx)
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestSingleUploadResultFromContext(t *testing.T) {
	// Test with nil context
	result := SingleUploadResultFromContext(context.Background())
	if result != nil {
		t.Error("Expected nil result for empty context")
	}

	// Test with context containing result
	testResult := &Result{OriginalName: "test.jpg", Size: 1024}
	ctx := context.WithValue(context.Background(), "upload_result", testResult)

	result = SingleUploadResultFromContext(ctx)
	if result != testResult {
		t.Error("Expected result from context")
	}
}

func TestMustSingleUploadResultFromContext(t *testing.T) {
	// Test panic case
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when result not found in context")
		}
	}()

	MustSingleUploadResultFromContext(context.Background())
}

func TestMustSingleUploadResultFromContextSuccess(t *testing.T) {
	// Test success case
	testResult := &Result{OriginalName: "test.jpg", Size: 1024}
	ctx := context.WithValue(context.Background(), "upload_result", testResult)

	result := MustSingleUploadResultFromContext(ctx)
	if result != testResult {
		t.Error("Expected result from context")
	}
}

func TestJSONUploadErrorHandler(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/upload", nil)

	JSONUploadErrorHandler(w, req, http.ErrServerClosed)

	// Check response
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status 422, got %d", w.Code)
	}

	// Check JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response["status"] != "error" {
		t.Errorf("Expected status 'error', got '%v'", response["status"])
	}

	if response["message"] != "Upload failed" {
		t.Errorf("Expected message 'Upload failed', got '%v'", response["message"])
	}
}

func TestHTMLUploadErrorHandler(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/upload", nil)

	HTMLUploadErrorHandler(w, req, http.ErrServerClosed)

	// Check response
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	body := w.Body.String()
	if !bytes.Contains([]byte(body), []byte("Upload Error")) {
		t.Error("Expected 'Upload Error' in HTML response")
	}
}

func TestUploadSuccessHandler(t *testing.T) {
	// Create mock storage
	mockStorage := storage.NewMockStorage()

	// Create upload processor
	processor := NewProcessor(mockStorage, Options{
		MaxFileSize:      10 * 1024 * 1024,
		AllowedMIMETypes: []string{"image/jpeg", "image/png", "application/octet-stream"},
	})

	// Create multipart request
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "test.jpg")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	part.Write([]byte("test file content"))
	writer.Close()

	req := httptest.NewRequest("POST", "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	// Create middleware chain
	uploadMiddleware := UploadMiddleware(processor, "file", nil)

	// Handler that doesn't write response
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do nothing - let success handler write response
	})

	// Apply upload middleware first, then success handler
	uploadMiddleware(UploadSuccessHandler(handler)).ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response["status"] != "success" {
		t.Errorf("Expected status 'success', got '%v'", response["status"])
	}

	if response["message"] != "Upload completed successfully" {
		t.Errorf("Expected message 'Upload completed successfully', got '%v'", response["message"])
	}
}

func TestCustomUploadErrorHandler(t *testing.T) {
	customHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("Custom upload error handler"))
	}

	// Create mock storage
	mockStorage := storage.NewMockStorage()

	// Create upload processor with restrictive options
	processor := NewProcessor(mockStorage, Options{
		MaxFileSize:      1024, // 1KB limit
		AllowedMIMETypes: []string{"image/jpeg", "image/png", "application/octet-stream"},
	})

	middleware := UploadMiddleware(processor, "file", customHandler)

	// Create large file
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "large.jpg")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	part.Write(make([]byte, 2048)) // 2KB file
	writer.Close()

	req := httptest.NewRequest("POST", "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for upload error")
	})

	middleware(handler).ServeHTTP(w, req)

	// Check custom response
	if w.Code != http.StatusTeapot {
		t.Errorf("Expected status 418, got %d", w.Code)
	}

	if w.Body.String() != "Custom upload error handler" {
		t.Errorf("Expected 'Custom upload error handler', got '%s'", w.Body.String())
	}
}
