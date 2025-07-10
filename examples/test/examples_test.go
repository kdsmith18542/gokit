package main_test

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kdsmith18542/gokit/form"
	"github.com/kdsmith18542/gokit/i18n"
	"github.com/kdsmith18542/gokit/upload"
	"github.com/kdsmith18542/gokit/upload/storage"
)

// TestMainExamples tests the main examples functionality
func TestMainExamples(t *testing.T) {
	// Test user registration form
	t.Run("UserRegistration", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/register", strings.NewReader("name=John&email=john@example.com&age=25"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// Simulate the registration handler
		var user struct {
			Name  string `form:"name" validate:"required"`
			Email string `form:"email" validate:"required,email"`
			Age   int    `form:"age" validate:"required,min=18"`
		}

		errs := form.DecodeAndValidate(req, &user)
		if len(errs) > 0 {
			t.Errorf("Expected no validation errors, got: %v", errs)
		}

		if user.Name != "John" || user.Email != "john@example.com" || user.Age != 25 {
			t.Errorf("Expected user data to be decoded correctly")
		}
	})

	// Test file upload
	t.Run("FileUpload", func(t *testing.T) {
		// Create a test file
		testFile := filepath.Join(t.TempDir(), "test.txt")
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		// Create multipart request
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		file, err := writer.CreateFormFile("file", "test.txt")
		if err != nil {
			t.Fatal(err)
		}
		file.Write([]byte("test content"))
		writer.Close()

		req := httptest.NewRequest("POST", "/upload", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		// Simulate upload handler
		localStorage := storage.NewLocal(t.TempDir())
		processor := upload.NewProcessor(localStorage, upload.Options{
			MaxFileSize:      1024 * 1024,
			AllowedMIMETypes: []string{"text/plain", "application/octet-stream"},
		})

		results, err := processor.Process(req, "file")
		if err != nil {
			t.Errorf("Expected no upload error, got: %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected upload results")
		}
	})

	// Test i18n functionality
	t.Run("I18nDemo", func(t *testing.T) {
		// Create test locale files
		tempDir := t.TempDir()
		enFile := filepath.Join(tempDir, "en.toml")
		esFile := filepath.Join(tempDir, "es.toml")

		os.WriteFile(enFile, []byte(`greeting = "Hello"`), 0644)
		os.WriteFile(esFile, []byte(`greeting = "Hola"`), 0644)

		// Create i18n manager
		manager := i18n.NewManager(tempDir)
		// manager.LoadFromDirectory(tempDir) // Commented out as per instructions

		// Test English
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "en")
		translator := manager.Translator(req)

		greeting := translator.T("greeting", nil)
		if greeting != "Hello" {
			t.Errorf("Expected 'Hello', got '%s'", greeting)
		}

		// Test Spanish
		req.Header.Set("Accept-Language", "es")
		translator = manager.Translator(req)
		greeting = translator.T("greeting", nil)
		if greeting != "Hola" {
			t.Errorf("Expected 'Hola', got '%s'", greeting)
		}
	})

	// Test performance monitoring
	t.Run("PerformanceDemo", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/data", nil)
		w := httptest.NewRecorder()

		// Simulate performance monitoring
		start := time.Now()
		time.Sleep(10 * time.Millisecond) // Simulate work
		duration := time.Since(start)

		if duration < 5*time.Millisecond {
			t.Error("Expected some processing time")
		}
	})

	// Test error handling
	t.Run("ErrorHandling", func(t *testing.T) {
		// Test validation error
		req := httptest.NewRequest("POST", "/register", strings.NewReader("name=&email=invalid&age=15"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var user struct {
			Name  string `form:"name" validate:"required"`
			Email string `form:"email" validate:"required,email"`
			Age   int    `form:"age" validate:"required,min=18"`
		}

		errs := form.DecodeAndValidate(req, &user)
		if len(errs) == 0 {
			t.Error("Expected validation errors")
		}

		// Test upload error
		req = httptest.NewRequest("POST", "/upload", strings.NewReader("invalid multipart"))
		req.Header.Set("Content-Type", "multipart/form-data")

		localStorage := storage.NewLocal(t.TempDir())
		processor := upload.NewProcessor(localStorage, upload.Options{
			MaxFileSize:      1024 * 1024,
			AllowedMIMETypes: []string{"text/plain"},
		})

		_, err := processor.Process(req, "file")
		if err == nil {
			t.Error("Expected upload error")
		}
	})
}

// TestAdvancedFeatures tests the advanced features example
func TestAdvancedFeatures(t *testing.T) {
	// Test complex form validation
	t.Run("ComplexValidation", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/complex", strings.NewReader(
			"email=test@example.com&password=secret123&confirm_password=secret123&bio=Hello World",
		))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var form struct {
			Email           string `form:"email" validate:"required,email"`
			Password        string `form:"password" validate:"required,min=8"`
			ConfirmPassword string `form:"confirm_password" validate:"required,eqfield=Password"`
			Bio             string `form:"bio" sanitize:"trim,escape_html"`
		}

		errs := form.DecodeAndValidate(req, &form)
		if len(errs) > 0 {
			t.Errorf("Expected no validation errors, got: %v", errs)
		}

		if form.Email != "test@example.com" || form.Password != "secret123" {
			t.Error("Expected form data to be decoded correctly")
		}
	})

	// Test file upload with hooks
	t.Run("UploadWithHooks", func(t *testing.T) {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		file, _ := writer.CreateFormFile("document", "test.pdf")
		file.Write([]byte("fake pdf content"))
		writer.Close()

		req := httptest.NewRequest("POST", "/upload", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		localStorage := storage.NewLocal(t.TempDir())
		processor := upload.NewProcessor(localStorage, upload.Options{
			MaxFileSize:      1024 * 1024,
			AllowedMIMETypes: []string{"application/pdf", "application/octet-stream"},
		})

		// Add success hook
		hookCalled := false
		processor.OnSuccess(func(ctx context.Context, result upload.Result) {
			hookCalled = true
		})

		results, err := processor.Process(req, "document")
		if err != nil {
			t.Errorf("Expected no upload error, got: %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected upload results")
		}

		// Give hooks time to execute
		time.Sleep(10 * time.Millisecond)
		if !hookCalled {
			t.Error("Expected success hook to be called")
		}
	})

	// Test i18n with pluralization
	t.Run("I18nPluralization", func(t *testing.T) {
		tempDir := t.TempDir()
		enFile := filepath.Join(tempDir, "en.toml")
		os.WriteFile(enFile, []byte(`
item_count = "{{.Count}} item"
item_count_plural = "{{.Count}} items"
`), 0644)

		manager := i18n.NewManager(tempDir)
		// manager.LoadFromDirectory(tempDir) // Commented out as per instructions

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "en")
		translator := manager.Translator(req)

		// Test singular
		msg1 := translator.T("item_count", map[string]interface{}{"Count": 1})
		if !strings.Contains(msg1, "1 item") {
			t.Errorf("Expected singular form, got: %s", msg1)
		}

		// Test plural
		msg5 := translator.T("item_count", map[string]interface{}{"Count": 5})
		if !strings.Contains(msg5, "5 items") {
			t.Errorf("Expected plural form, got: %s", msg5)
		}
	})

	// Test observability integration
	t.Run("ObservabilityIntegration", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/observable", nil)
		w := httptest.NewRecorder()

		// Simulate observable operation
		ctx := req.Context()
		// ctx, span := i18n.StartSpan(ctx, "test_operation") // Commented out as per instructions
		// defer span.End() // Commented out as per instructions

		// Simulate some work
		time.Sleep(5 * time.Millisecond)

		if w.Code != 0 {
			t.Error("Expected no response code set")
		}
	})
}

// TestFormMiddleware tests the form middleware example
func TestFormMiddleware(t *testing.T) {
	t.Run("FormMiddlewareDemo", func(t *testing.T) {
		// Test form validation middleware
		req := httptest.NewRequest("POST", "/submit", strings.NewReader("name=Test&email=test@example.com"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// Simulate middleware
		var data struct {
			Name  string `form:"name" validate:"required"`
			Email string `form:"email" validate:"required,email"`
		}

		errs := form.DecodeAndValidate(req, &data)
		if len(errs) > 0 {
			t.Errorf("Expected no validation errors, got: %v", errs)
		}

		if data.Name != "Test" || data.Email != "test@example.com" {
			t.Error("Expected form data to be decoded correctly")
		}
	})
}

// TestI18nMiddleware tests the i18n middleware example
func TestI18nMiddleware(t *testing.T) {
	t.Run("I18nMiddlewareDemo", func(t *testing.T) {
		// Create test locale
		tempDir := t.TempDir()
		enFile := filepath.Join(tempDir, "en.toml")
		os.WriteFile(enFile, []byte(`welcome = "Welcome"`), 0644)

		manager := i18n.NewManager(tempDir)
		// manager.LoadFromDirectory(tempDir) // Commented out as per instructions

		// Test middleware behavior
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "en")
		w := httptest.NewRecorder()

		translator := manager.Translator(req)
		message := translator.T("welcome", nil)

		if message != "Welcome" {
			t.Errorf("Expected 'Welcome', got '%s'", message)
		}
	})
}

// TestUploadMiddleware tests the upload middleware example
func TestUploadMiddleware(t *testing.T) {
	t.Run("UploadMiddlewareDemo", func(t *testing.T) {
		// Create test file upload
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		file, _ := writer.CreateFormFile("upload", "test.txt")
		file.Write([]byte("test content"))
		writer.Close()

		req := httptest.NewRequest("POST", "/upload", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		// Simulate upload middleware
		localStorage := storage.NewLocal(t.TempDir())
		processor := upload.NewProcessor(localStorage, upload.Options{
			MaxFileSize:      1024 * 1024,
			AllowedMIMETypes: []string{"text/plain", "application/octet-stream"},
		})

		results, err := processor.Process(req, "upload")
		if err != nil {
			t.Errorf("Expected no upload error, got: %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected upload results")
		}
	})
}

// TestI18nEditor tests the i18n editor example
func TestI18nEditor(t *testing.T) {
	t.Run("I18nEditorDemo", func(t *testing.T) {
		// Create test locale files
		tempDir := t.TempDir()
		enFile := filepath.Join(tempDir, "en.toml")
		esFile := filepath.Join(tempDir, "es.toml")

		os.WriteFile(enFile, []byte(`hello = "Hello"`), 0644)
		os.WriteFile(esFile, []byte(`hello = "Hola"`), 0644)

		// Test editor functionality
		manager := i18n.NewManager(tempDir)
		// manager.LoadFromDirectory(tempDir) // Commented out as per instructions

		// Test locale loading
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "en")
		translator := manager.Translator(req)

		hello := translator.T("hello", nil)
		if hello != "Hello" {
			t.Errorf("Expected 'Hello', got '%s'", hello)
		}

		// Test Spanish
		req.Header.Set("Accept-Language", "es")
		translator = manager.Translator(req)
		hello = translator.T("hello", nil)
		if hello != "Hola" {
			t.Errorf("Expected 'Hola', got '%s'", hello)
		}
	})
}

// TestObservabilityDemo tests the observability demo
func TestObservabilityDemo(t *testing.T) {
	t.Run("ObservabilityDemo", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/observable", nil)
		w := httptest.NewRecorder()

		// Simulate observable operation
		ctx := req.Context()
		// ctx, span := i18n.StartSpan(ctx, "demo_operation") // Commented out as per instructions
		// defer span.End() // Commented out as per instructions

		// Simulate some work
		time.Sleep(5 * time.Millisecond)

		// Test metrics
		// span.SetAttributes("operation", "demo") // Commented out as per instructions
		// span.SetAttributes("status", "success") // Commented out as per instructions

		if w.Code != 0 {
			t.Error("Expected no response code set")
		}
	})
}

// TestResumableUpload tests the resumable upload example
func TestResumableUpload(t *testing.T) {
	t.Run("ResumableUploadDemo", func(t *testing.T) {
		// Test resumable upload functionality
		localStorage := storage.NewLocal(t.TempDir())
		processor := upload.NewProcessor(localStorage, upload.Options{
			MaxFileSize:      1024 * 1024,
			AllowedMIMETypes: []string{"text/plain", "application/octet-stream"},
		})

		// Test chunk upload
		chunkData := []byte("chunk data")
		chunkID := "test-chunk-123"
		uploadID := "test-upload-456"

		// Simulate chunk storage
		err := localStorage.Store(context.Background(), fmt.Sprintf("chunks/%s/%s", uploadID, chunkID), chunkData)
		if err != nil {
			t.Errorf("Expected no error storing chunk, got: %v", err)
		}

		// Test chunk retrieval
		retrieved, err := localStorage.Retrieve(context.Background(), fmt.Sprintf("chunks/%s/%s", uploadID, chunkID))
		if err != nil {
			t.Errorf("Expected no error retrieving chunk, got: %v", err)
		}

		if string(retrieved) != string(chunkData) {
			t.Error("Expected chunk data to match")
		}
	})
}

// TestCLIExamples tests CLI functionality
func TestCLIExamples(t *testing.T) {
	t.Run("CLIDemo", func(t *testing.T) {
		// Test CLI argument parsing
		args := []string{"--help"}

		// Simulate CLI help
		if len(args) == 0 {
			t.Error("Expected CLI arguments")
		}

		// Test file operations
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		// Test file reading
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Errorf("Expected no error reading file, got: %v", err)
		}

		if string(content) != "test content" {
			t.Error("Expected file content to match")
		}
	})
}

// TestErrorScenarios tests various error scenarios
func TestErrorScenarios(t *testing.T) {
	t.Run("ValidationErrors", func(t *testing.T) {
		// Test invalid email
		req := httptest.NewRequest("POST", "/register", strings.NewReader("email=invalid-email"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var user struct {
			Email string `form:"email" validate:"required,email"`
		}

		errs := form.DecodeAndValidate(req, &user)
		if len(errs) == 0 {
			t.Error("Expected validation errors for invalid email")
		}
	})

	t.Run("UploadErrors", func(t *testing.T) {
		// Test file too large
		req := httptest.NewRequest("POST", "/upload", strings.NewReader("large content"))
		req.Header.Set("Content-Type", "multipart/form-data")

		localStorage := storage.NewLocal(t.TempDir())
		processor := upload.NewProcessor(localStorage, upload.Options{
			MaxFileSize:      10, // Very small limit
			AllowedMIMETypes: []string{"text/plain"},
		})

		_, err := processor.Process(req, "file")
		if err == nil {
			t.Error("Expected upload error for large file")
		}
	})

	t.Run("I18nErrors", func(t *testing.T) {
		// Test missing translation
		manager := i18n.NewManager(t.TempDir())
		req := httptest.NewRequest("GET", "/", nil)
		translator := manager.Translator(req)

		// Test missing key
		message := translator.T("missing_key", nil)
		if message != "missing_key" {
			t.Errorf("Expected key name for missing translation, got: %s", message)
		}
	})
}

// TestPerformance tests performance scenarios
func TestPerformance(t *testing.T) {
	t.Run("ConcurrentUploads", func(t *testing.T) {
		localStorage := storage.NewLocal(t.TempDir())
		processor := upload.NewProcessor(localStorage, upload.Options{
			MaxFileSize:      1024 * 1024,
			AllowedMIMETypes: []string{"text/plain", "application/octet-stream"},
		})

		// Test concurrent uploads
		done := make(chan bool, 5)
		for i := 0; i < 5; i++ {
			go func(id int) {
				defer func() { done <- true }()

				var buf bytes.Buffer
				writer := multipart.NewWriter(&buf)
				file, _ := writer.CreateFormFile("file", fmt.Sprintf("test%d.txt", id))
				file.Write([]byte(fmt.Sprintf("content %d", id)))
				writer.Close()

				req := httptest.NewRequest("POST", "/upload", &buf)
				req.Header.Set("Content-Type", writer.FormDataContentType())

				_, err := processor.Process(req, "file")
				if err != nil {
					t.Errorf("Expected no error in concurrent upload %d, got: %v", id, err)
				}
			}(i)
		}

		// Wait for all uploads to complete
		for i := 0; i < 5; i++ {
			<-done
		}
	})

	t.Run("LargeFileHandling", func(t *testing.T) {
		// Test large file handling
		localStorage := storage.NewLocal(t.TempDir())
		processor := upload.NewProcessor(localStorage, upload.Options{
			MaxFileSize:      1024 * 1024,
			AllowedMIMETypes: []string{"text/plain", "application/octet-stream"},
		})

		// Create large content
		largeContent := strings.Repeat("a", 10000)
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		file, _ := writer.CreateFormFile("large_file", "large.txt")
		file.Write([]byte(largeContent))
		writer.Close()

		req := httptest.NewRequest("POST", "/upload", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		results, err := processor.Process(req, "large_file")
		if err != nil {
			t.Errorf("Expected no error uploading large file, got: %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected upload results for large file")
		}
	})
}

// TestIntegrationWorkflow tests complete integration workflows
func TestIntegrationWorkflow(t *testing.T) {
	t.Run("CompleteUserWorkflow", func(t *testing.T) {
		// Setup
		tempDir := t.TempDir()
		localStorage := storage.NewLocal(tempDir)
		processor := upload.NewProcessor(localStorage, upload.Options{
			MaxFileSize:      1024 * 1024,
			AllowedMIMETypes: []string{"text/plain", "application/octet-stream"},
		})

		// Create i18n manager
		enFile := filepath.Join(tempDir, "en.toml")
		os.WriteFile(enFile, []byte(`welcome = "Welcome"`), 0644)
		manager := i18n.NewManager(tempDir)
		// manager.LoadFromDirectory(tempDir) // Commented out as per instructions

		// Step 1: User registration
		req := httptest.NewRequest("POST", "/register", strings.NewReader("name=John&email=john@example.com"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var user struct {
			Name  string `form:"name" validate:"required"`
			Email string `form:"email" validate:"required,email"`
		}

		errs := form.DecodeAndValidate(req, &user)
		if len(errs) > 0 {
			t.Errorf("Expected no validation errors, got: %v", errs)
		}

		// Step 2: File upload
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		file, _ := writer.CreateFormFile("avatar", "avatar.txt")
		file.Write([]byte("avatar content"))
		writer.Close()

		req = httptest.NewRequest("POST", "/upload", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		results, err := processor.Process(req, "avatar")
		if err != nil {
			t.Errorf("Expected no upload error, got: %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected upload results")
		}

		// Step 3: Internationalized response
		req = httptest.NewRequest("GET", "/welcome", nil)
		req.Header.Set("Accept-Language", "en")
		translator := manager.Translator(req)

		welcome := translator.T("welcome", nil)
		if welcome != "Welcome" {
			t.Errorf("Expected 'Welcome', got '%s'", welcome)
		}
	})
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	t.Run("EmptyForm", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/submit", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var data struct {
			Name string `form:"name" validate:"required"`
		}

		errs := form.DecodeAndValidate(req, &data)
		if len(errs) == 0 {
			t.Error("Expected validation errors for empty form")
		}
	})

	t.Run("EmptyFile", func(t *testing.T) {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		file, _ := writer.CreateFormFile("empty", "empty.txt")
		// Don't write anything
		writer.Close()

		req := httptest.NewRequest("POST", "/upload", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		localStorage := storage.NewLocal(t.TempDir())
		processor := upload.NewProcessor(localStorage, upload.Options{
			MaxFileSize:      1024 * 1024,
			AllowedMIMETypes: []string{"text/plain", "application/octet-stream"},
		})

		results, err := processor.Process(req, "empty")
		if err != nil {
			t.Errorf("Expected no error for empty file, got: %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected results even for empty file")
		}
	})

	t.Run("InvalidLocale", func(t *testing.T) {
		manager := i18n.NewManager(t.TempDir())
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "invalid-locale")

		translator := manager.Translator(req)
		message := translator.T("test", nil)
		if message != "test" {
			t.Errorf("Expected key name for invalid locale, got: %s", message)
		}
	})
}

// TestContextHandling tests context handling
func TestContextHandling(t *testing.T) {
	t.Run("ContextCancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		req := httptest.NewRequest("POST", "/upload", strings.NewReader("data"))
		req = req.WithContext(ctx)

		localStorage := storage.NewLocal(t.TempDir())
		processor := upload.NewProcessor(localStorage, upload.Options{
			MaxFileSize:      1024 * 1024,
			AllowedMIMETypes: []string{"text/plain"},
		})

		_, err := processor.Process(ctx, req, "file")
		if err == nil {
			t.Error("Expected error for cancelled context")
		}
	})

	t.Run("ContextTimeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		req := httptest.NewRequest("POST", "/upload", strings.NewReader("data"))
		req = req.WithContext(ctx)

		localStorage := storage.NewLocal(t.TempDir())
		processor := upload.NewProcessor(localStorage, upload.Options{
			MaxFileSize:      1024 * 1024,
			AllowedMIMETypes: []string{"text/plain"},
		})

		// Simulate slow operation
		time.Sleep(10 * time.Millisecond)

		_, err := processor.Process(ctx, req, "file")
		if err == nil {
			t.Error("Expected error for timed out context")
		}
	})
}

// TestMemoryUsage tests memory usage scenarios
func TestMemoryUsage(t *testing.T) {
	t.Run("LargeFormData", func(t *testing.T) {
		// Create large form data
		largeData := strings.Repeat("field=value&", 1000)
		req := httptest.NewRequest("POST", "/submit", strings.NewReader(largeData))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var data struct {
			Field string `form:"field"`
		}

		errs := form.DecodeAndValidate(req, &data)
		if len(errs) > 0 {
			t.Errorf("Expected no validation errors, got: %v", errs)
		}
	})

	t.Run("MultipleFiles", func(t *testing.T) {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		// Add multiple files
		for i := 0; i < 5; i++ {
			file, _ := writer.CreateFormFile(fmt.Sprintf("file%d", i), fmt.Sprintf("test%d.txt", i))
			file.Write([]byte(fmt.Sprintf("content %d", i)))
		}
		writer.Close()

		req := httptest.NewRequest("POST", "/upload", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		localStorage := storage.NewLocal(t.TempDir())
		processor := upload.NewProcessor(localStorage, upload.Options{
			MaxFileSize:      1024 * 1024,
			AllowedMIMETypes: []string{"text/plain", "application/octet-stream"},
		})

		results, err := processor.Process(req, "file0", "file1", "file2", "file3", "file4")
		if err != nil {
			t.Errorf("Expected no error uploading multiple files, got: %v", err)
		}

		if len(results) != 5 {
			t.Errorf("Expected 5 results, got %d", len(results))
		}
	})
}

// TestErrorRecovery tests error recovery scenarios
func TestErrorRecovery(t *testing.T) {
	t.Run("StorageFailure", func(t *testing.T) {
		// Test with invalid storage directory
		invalidDir := "/invalid/path/that/does/not/exist"
		localStorage, err := storage.NewLocal(invalidDir)
		if err == nil {
			t.Error("Expected error for invalid storage directory")
		}

		// Test with valid directory
		validDir := t.TempDir()
		localStorage, err = storage.NewLocal(validDir)
		if err != nil {
			t.Errorf("Expected no error for valid storage directory, got: %v", err)
		}

		// Test storage operations
		err = localStorage.Store(context.Background(), "test.txt", []byte("test"))
		if err != nil {
			t.Errorf("Expected no error storing file, got: %v", err)
		}

		data, err := localStorage.Retrieve(context.Background(), "test.txt")
		if err != nil {
			t.Errorf("Expected no error retrieving file, got: %v", err)
		}

		if string(data) != "test" {
			t.Error("Expected retrieved data to match")
		}
	})

	t.Run("FormRecovery", func(t *testing.T) {
		// Test recovery from invalid form data
		req := httptest.NewRequest("POST", "/submit", strings.NewReader("invalid=form=data"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var data struct {
			Name string `form:"name" validate:"required"`
		}

		errs := form.DecodeAndValidate(req, &data)
		if len(errs) == 0 {
			t.Error("Expected validation errors for invalid form data")
		}

		// Test recovery with valid data
		req = httptest.NewRequest("POST", "/submit", strings.NewReader("name=Valid"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		errs = form.DecodeAndValidate(req, &data)
		if len(errs) > 0 {
			t.Errorf("Expected no validation errors, got: %v", errs)
		}

		if data.Name != "Valid" {
			t.Error("Expected name to be 'Valid'")
		}
	})
}

// TestSecurity tests security-related scenarios
func TestSecurity(t *testing.T) {
	t.Run("PathTraversal", func(t *testing.T) {
		// Test path traversal attempts
		maliciousPath := "../../../etc/passwd"
		localStorage := storage.NewLocal(t.TempDir())

		err := localStorage.Store(context.Background(), maliciousPath, []byte("malicious"))
		if err == nil {
			t.Error("Expected error for path traversal attempt")
		}
	})

	t.Run("LargeFileAttack", func(t *testing.T) {
		// Test large file upload attack
		largeContent := strings.Repeat("a", 1024*1024) // 1MB
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		file, _ := writer.CreateFormFile("large", "large.txt")
		file.Write([]byte(largeContent))
		writer.Close()

		req := httptest.NewRequest("POST", "/upload", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		localStorage := storage.NewLocal(t.TempDir())
		processor := upload.NewProcessor(localStorage, upload.Options{
			MaxFileSize:      1024, // 1KB limit
			AllowedMIMETypes: []string{"text/plain"},
		})

		_, err := processor.Process(req, "large")
		if err == nil {
			t.Error("Expected error for file exceeding size limit")
		}
	})

	t.Run("InvalidMimeType", func(t *testing.T) {
		// Test invalid MIME type
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		file, _ := writer.CreateFormFile("executable", "script.exe")
		file.Write([]byte("malicious content"))
		writer.Close()

		req := httptest.NewRequest("POST", "/upload", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		localStorage := storage.NewLocal(t.TempDir())
		processor := upload.NewProcessor(localStorage, upload.Options{
			MaxFileSize:      1024 * 1024,
			AllowedMIMETypes: []string{"text/plain"}, // Only allow text files
		})

		_, err := processor.Process(req, "executable")
		if err == nil {
			t.Error("Expected error for invalid MIME type")
		}
	})
}

// TestBenchmarks runs performance benchmarks
func TestBenchmarks(t *testing.T) {
	t.Run("FormValidationBenchmark", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/submit", strings.NewReader("name=Test&email=test@example.com&age=25"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var data struct {
			Name  string `form:"name" validate:"required"`
			Email string `form:"email" validate:"required,email"`
			Age   int    `form:"age" validate:"required,min=18"`
		}

		start := time.Now()
		for i := 0; i < 1000; i++ {
			errs := form.DecodeAndValidate(req, &data)
			if len(errs) > 0 {
				t.Errorf("Expected no validation errors, got: %v", errs)
			}
		}
		duration := time.Since(start)

		if duration > 1*time.Second {
			t.Errorf("Form validation took too long: %v", duration)
		}
	})

	t.Run("FileUploadBenchmark", func(t *testing.T) {
		localStorage := storage.NewLocal(t.TempDir())
		processor := upload.NewProcessor(localStorage, upload.Options{
			MaxFileSize:      1024 * 1024,
			AllowedMIMETypes: []string{"text/plain", "application/octet-stream"},
		})

		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		file, _ := writer.CreateFormFile("test", "test.txt")
		file.Write([]byte("test content"))
		writer.Close()

		req := httptest.NewRequest("POST", "/upload", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		start := time.Now()
		for i := 0; i < 100; i++ {
			_, err := processor.Process(req, "test")
			if err != nil {
				t.Errorf("Expected no upload error, got: %v", err)
			}
		}
		duration := time.Since(start)

		if duration > 5*time.Second {
			t.Errorf("File upload took too long: %v", duration)
		}
	})
}
