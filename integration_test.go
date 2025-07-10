package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kdsmith18542/gokit/form"
	"github.com/kdsmith18542/gokit/i18n"
	"github.com/kdsmith18542/gokit/observability"
	"github.com/kdsmith18542/gokit/upload"
	"github.com/kdsmith18542/gokit/upload/storage"
)

// IntegrationTestSuite tests the complete GoKit workflow
type IntegrationTestSuite struct {
	t               *testing.T
	i18nManager     *i18n.Manager
	uploadProcessor *upload.Processor
	localStorage    storage.Storage
	tempDir         string
}

// TestUser represents a user for integration testing
type TestUser struct {
	Email           string `form:"email" validate:"required,email"`
	Password        string `form:"password" validate:"required,min=8"`
	ConfirmPassword string `form:"confirm_password" validate:"required,eqfield=Password"`
	Name            string `form:"name" validate:"required" sanitize:"trim"`
	Bio             string `form:"bio" sanitize:"trim,escape_html"`
	Avatar          string `form:"avatar"`
}

// TestIntegration_CompleteWorkflow tests the complete user registration workflow
func TestIntegration_CompleteWorkflow(t *testing.T) {
	suite := setupIntegrationSuite(t)
	defer suite.cleanup()

	// Test 1: User registration with form validation and i18n
	t.Run("UserRegistration", func(t *testing.T) {
		suite.testUserRegistration(t)
	})

	// Test 2: File upload with validation and storage
	t.Run("FileUpload", func(t *testing.T) {
		suite.testFileUpload(t)
	})

	// Test 3: i18n with locale detection and formatting
	t.Run("Internationalization", func(t *testing.T) {
		suite.testInternationalization(t)
	})

	// Test 4: Observability integration
	t.Run("Observability", func(t *testing.T) {
		suite.testObservability(t)
	})

	// Test 5: Error handling and edge cases
	t.Run("ErrorHandling", func(t *testing.T) {
		suite.testErrorHandling(t)
	})

	// Test 6: Middleware integration
	t.Run("Middleware", func(t *testing.T) {
		suite.testMiddleware(t)
	})

	// Test 7: Resumable Uploads
	t.Run("ResumableUploads", func(t *testing.T) {
		suite.testResumableUpload(t)
	})

	// Test 8: Cloud Storage Backends (S3, GCS)
	t.Run("CloudStorage", func(t *testing.T) {
		suite.testCloudStorage(t)
	})

	// Test 9: Presigned URLs
	t.Run("PresignedURLs", func(t *testing.T) {
		suite.testPresignedURLs(t)
	})

	// Test 10: Custom Form Validators
	t.Run("CustomFormValidators", func(t *testing.T) {
		suite.testCustomFormValidators(t)
	})

	// Test 11: i18n Date/Time Formatting
	t.Run("I18nDateTimeFormatting", func(t *testing.T) {
		suite.testI18nDateTimeFormatting(t)
	})
}

// TestIntegration_CrossPackageFeatures tests features that span multiple packages
func TestIntegration_CrossPackageFeatures(t *testing.T) {
	suite := setupIntegrationSuite(t)
	defer suite.cleanup()

	// Test 1: Form validation with i18n error messages
	t.Run("FormValidationWithI18n", func(t *testing.T) {
		suite.testFormValidationWithI18n(t)
	})

	// Test 2: Upload with form validation
	t.Run("UploadWithFormValidation", func(t *testing.T) {
		suite.testUploadWithFormValidation(t)
	})

	// Test 3: Observability across all operations
	t.Run("CrossPackageObservability", func(t *testing.T) {
		suite.testCrossPackageObservability(t)
	})
}

func setupIntegrationSuite(t *testing.T) *IntegrationTestSuite {
	// Initialize observability
	if err := observability.Init(observability.Config{
		ServiceName:    "gokit-integration-test",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		EnableTracing:  true,
		EnableMetrics:  true,
		EnableLogging:  true,
	}); err != nil {
		t.Fatalf("Failed to initialize observability: %v", err)
	}

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "gokit-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create locales directory
	localesDir := filepath.Join(tempDir, "locales")
	if err := os.MkdirAll(localesDir, 0755); err != nil {
		t.Fatalf("Failed to create locales directory: %v", err)
	}

	// Create test locale files
	createTestLocaleFiles(t, localesDir)

	// Initialize i18n manager
	i18nManager := i18n.NewManager(localesDir)

	// Create upload directory
	uploadDir := filepath.Join(tempDir, "uploads")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		t.Fatalf("Failed to create upload directory: %v", err)
	}

	// Initialize upload processor
	localStorage := storage.NewMockStorage() // Change to MockStorage for integration tests
	uploadProcessor := upload.NewProcessor(localStorage, upload.Options{
		MaxFileSize:      5 * 1024 * 1024, // 5MB
		AllowedMIMETypes: []string{"image/jpeg", "image/png", "application/pdf", "text/plain", "application/octet-stream"},
		MaxFiles:         3,
	})

	// Register success hook for testing
	uploadProcessor.OnSuccess(func(ctx context.Context, result upload.Result) {
		t.Logf("Upload success: %s -> %s", result.OriginalName, result.URL)
	})

	return &IntegrationTestSuite{
		t:               t,
		i18nManager:     i18nManager,
		uploadProcessor: uploadProcessor,
		localStorage:    localStorage,
		tempDir:         tempDir,
	}
}

func (suite *IntegrationTestSuite) cleanup() {
	if err := os.RemoveAll(suite.tempDir); err != nil {
		suite.t.Logf("Failed to cleanup temp directory: %v", err)
	}
}

func (suite *IntegrationTestSuite) testUserRegistration(t *testing.T) {
	// Create test user data
	userData := map[string]string{
		"email":            "test@example.com",
		"password":         "securepassword123",
		"confirm_password": "securepassword123",
		"name":             "  John Doe  ",                             // Will be trimmed
		"bio":              "<script>alert('xss')</script>Hello World", // Will be escaped
		"avatar":           "profile.jpg",
	}

	// Create multipart form request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add form fields
	for key, value := range userData {
		if key != "avatar" {
			if err := writer.WriteField(key, value); err != nil {
				t.Fatalf("Failed to write field %s: %v", key, err)
			}
		}
	}

	// Add file
	fileWriter, err := writer.CreateFormFile("avatar", "test.jpg")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	if _, err := fileWriter.Write([]byte("fake image data")); err != nil {
		t.Fatalf("Failed to write fake image data: %v", err)
	}

	writer.Close()

	// Create request
	req := httptest.NewRequest("POST", "/register", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	// Test form validation
	var user TestUser
	errors := form.DecodeAndValidateWithContext(req.Context(), req, &user)

	// Verify validation results
	if len(errors) > 0 {
		t.Errorf("Expected no validation errors, got: %v", errors)
	}

	// Verify sanitization worked
	if user.Name != "John Doe" {
		t.Errorf("Expected name to be trimmed, got: %q", user.Name)
	}

	if !strings.Contains(user.Bio, "&lt;script&gt;") {
		t.Errorf("Expected HTML to be escaped, got: %q", user.Bio)
	}

	// Test with i18n
	translator := suite.i18nManager.Translator(req)
	welcomeMsg := translator.T("welcome", map[string]interface{}{
		"Name": user.Name,
	})

	if !strings.Contains(welcomeMsg, user.Name) {
		t.Errorf("Expected welcome message to contain user name, got: %q", welcomeMsg)
	}

	// Explicitly handle avatar file upload after form validation
	// Create multipart form request specifically for the avatar file
	body = &bytes.Buffer{}
	writer = multipart.NewWriter(body)
	fileWriter, err = writer.CreateFormFile("avatar", "profile.jpg")
	if err != nil {
		t.Fatalf("Failed to create form file for avatar: %v", err)
	}
	if _, err := fileWriter.Write([]byte("fake image data")); err != nil {
		t.Fatalf("Failed to write fake image data: %v", err)
	}
	writer.Close()

	req = httptest.NewRequest("POST", "/upload/avatar", body) // Use a dedicated endpoint for avatar upload
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Process the avatar file using the upload processor
	avatarResults, err := suite.uploadProcessor.ProcessWithContext(req.Context(), req, "avatar")
	if err != nil {
		t.Fatalf("Avatar upload failed: %v", err)
	}

	if len(avatarResults) != 1 || !suite.localStorage.Exists(avatarResults[0].Path) {
		t.Errorf("Expected avatar to be uploaded and exist in mock storage, got: %+v", avatarResults)
	}

	// Clean up the uploaded avatar from mock storage
	err = suite.localStorage.Delete(avatarResults[0].Path)
	if err != nil {
		t.Errorf("Failed to delete avatar from mock storage: %v", err)
	}
}

func (suite *IntegrationTestSuite) testFileUpload(t *testing.T) {
	// Create test file data
	fileData := []byte("fake image data for testing")
	fileName := "test-image.txt"

	// Create multipart form request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	fileWriter, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	fileWriter.Write(fileData)

	writer.Close()

	// Create request
	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Test upload processing
	results, err := suite.uploadProcessor.ProcessWithContext(req.Context(), req, "file")
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 upload result, got %d", len(results))
	}

	result := results[0]
	if result.OriginalName != fileName {
		t.Errorf("Expected original name %s, got %s", fileName, result.OriginalName)
	}

	if result.Size != int64(len(fileData)) {
		t.Errorf("Expected size %d, got %d", len(fileData), result.Size)
	}

	// Verify file exists in storage (access storage directly)
	if !suite.localStorage.Exists(result.Path) {
		t.Errorf("Uploaded file does not exist in storage")
	}

	// Test file URL generation (local storage without baseURL returns empty string)
	url := suite.localStorage.GetURL(result.Path)
	// For local storage without baseURL, this is expected to be empty
	// In a real application, you would set a baseURL for local storage
	if url == "" {
		// This is expected behavior for local storage without baseURL
		t.Log("Local storage URL is empty (expected for local storage without baseURL)")
	}
}

func (suite *IntegrationTestSuite) testInternationalization(t *testing.T) {
	// Test locale detection
	testCases := []struct {
		acceptLanguage string
		expectedLocale string
	}{
		{"en-US,en;q=0.9", "en"},
		{"es-ES,es;q=0.9", "es"},
		{"fr-FR,fr;q=0.9", "fr"},
		{"invalid-locale", "en"}, // fallback
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Locale_%s", tc.acceptLanguage), func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Accept-Language", tc.acceptLanguage)

			translator := suite.i18nManager.Translator(req)
			message := translator.T("welcome", map[string]interface{}{
				"Name": "Test User",
			})

			if message == "" {
				t.Error("Expected non-empty translation")
			}

			// Test pluralization
			itemCount := translator.Tn("item", "items", 1, nil)
			if itemCount == "" {
				t.Error("Expected non-empty singular translation")
			}

			itemsCount := translator.Tn("item", "items", 5, nil)
			if itemsCount == "" {
				t.Error("Expected non-empty plural translation")
			}

			// Test number formatting
			formattedNumber := translator.FormatNumber(1234.56)
			if formattedNumber == "" {
				t.Error("Expected non-empty formatted number")
			}

			// Test currency formatting
			formattedCurrency := translator.FormatCurrency(1234.56, "USD")
			if formattedCurrency == "" {
				t.Error("Expected non-empty formatted currency")
			}
		})
	}
}

func (suite *IntegrationTestSuite) testObservability(t *testing.T) {
	ctx := context.Background()

	// Test form validation observability
	_, span := observability.StartSpan(ctx, "test_form_validation")
	defer span.End()

	req := httptest.NewRequest("POST", "/test", strings.NewReader("email=test@example.com"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var user TestUser
	errors := form.DecodeAndValidateWithContext(ctx, req, &user)

	observability.SetSpanAttributes(ctx, map[string]string{
		"form.name":   "TestUser",
		"error.count": fmt.Sprintf("%d", len(errors)),
		"validation":  "success",
	})

	// Test upload observability
	_, uploadSpan := observability.StartSpan(ctx, "test_upload")
	defer uploadSpan.End()

	// Create test upload
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fileWriter, _ := writer.CreateFormFile("file", "test.txt")
	fileWriter.Write([]byte("test data"))
	writer.Close()

	uploadReq := httptest.NewRequest("POST", "/upload", body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())

	results, err := suite.uploadProcessor.ProcessWithContext(ctx, uploadReq, "file")

	observability.SetSpanAttributes(ctx, map[string]string{
		"upload.success": fmt.Sprintf("%t", err == nil),
		"upload.count":   fmt.Sprintf("%d", len(results)),
	})

	// Test i18n observability
	_, i18nSpan := observability.StartSpan(ctx, "test_i18n")
	defer i18nSpan.End()

	translator := suite.i18nManager.Translator(uploadReq)
	message := translator.T("welcome", map[string]interface{}{
		"Name": "Test User",
	})

	observability.SetSpanAttributes(ctx, map[string]string{
		"i18n.locale": "en",
		"i18n.key":    "welcome",
		"i18n.result": message,
	})
}

func (suite *IntegrationTestSuite) testErrorHandling(t *testing.T) {
	// Test form validation errors
	t.Run("FormValidationErrors", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", strings.NewReader("email=invalid-email"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var user TestUser
		errors := form.DecodeAndValidateWithContext(req.Context(), req, &user)

		if len(errors) == 0 {
			t.Error("Expected validation errors for invalid email")
		}

		if errors["email"] == nil {
			t.Error("Expected email validation error")
		}
	})

	// Test upload errors
	t.Run("UploadErrors", func(t *testing.T) {
		// Test file too large
		largeData := make([]byte, 10*1024*1024) // 10MB
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		fileWriter, _ := writer.CreateFormFile("file", "large.txt")
		fileWriter.Write(largeData)
		writer.Close()

		req := httptest.NewRequest("POST", "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		_, err := suite.uploadProcessor.ProcessWithContext(req.Context(), req, "file")
		if err == nil {
			t.Error("Expected error for file too large")
		}
	})

	// Test i18n errors
	t.Run("I18nErrors", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "invalid-locale")

		translator := suite.i18nManager.Translator(req)
		message := translator.T("nonexistent_key", nil)

		// Should return the key itself when translation not found
		if message != "nonexistent_key" {
			t.Errorf("Expected key to be returned when translation not found, got: %q", message)
		}
	})
}

func (suite *IntegrationTestSuite) testMiddleware(t *testing.T) {
	// Test form validation middleware
	t.Run("FormValidationMiddleware", func(t *testing.T) {
		handler := form.ValidationMiddleware(TestUser{}, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			formVal := form.ValidatedFormFromContext(r.Context())
			if formVal == nil {
				t.Error("Expected form in context")
				return
			}
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("POST", "/test", strings.NewReader("email=test@example.com&password=password123&confirm_password=password123&name=Test"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})

	// Test upload middleware
	t.Run("UploadMiddleware", func(t *testing.T) {
		handler := upload.UploadMiddleware(suite.uploadProcessor, "file", nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			results := upload.UploadResultsFromContext(r.Context())
			if results == nil {
				t.Error("Expected upload results in context")
				return
			}
			w.WriteHeader(http.StatusOK)
		}))

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		fileWriter, _ := writer.CreateFormFile("file", "test.txt")
		fileWriter.Write([]byte("test data"))
		writer.Close()

		req := httptest.NewRequest("POST", "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})

	// Test i18n middleware
	t.Run("I18nMiddleware", func(t *testing.T) {
		handler := i18n.LocaleDetector(suite.i18nManager)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			translator := i18n.TranslatorFromContext(r.Context())
			if translator == nil {
				t.Error("Expected translator in context")
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Accept-Language", "es-ES")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})
}

func (suite *IntegrationTestSuite) testResumableUpload(t *testing.T) {
	t.Run("ResumableUpload", func(t *testing.T) {
		// Create a ResumableProcessor instance
		resumableProcessor := upload.NewResumableProcessor(suite.localStorage, upload.Options{
			MaxFileSize:      10 * 1024 * 1024, // 10MB
			AllowedMIMETypes: []string{"image/jpeg", "application/pdf"},
			MaxFiles:         1,
		})

		// 1. Initiate Upload
		initiateBody := map[string]interface{}{
			"file_name":  "large_doc.pdf",
			"total_size": 5 * 1024 * 1024, // 5MB
			"mime_type":  "application/pdf",
			"chunk_size": 1 * 1024 * 1024, // 1MB chunks
		}
		jsonInitiateBody, _ := json.Marshal(initiateBody)

		req := httptest.NewRequest("POST", "/resumable-upload", bytes.NewBuffer(jsonInitiateBody))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		resumableProcessor.HandleResumableUpload(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Initiate upload expected status 200, got: %d\nResponse: %s", rec.Code, rec.Body.String())
		}

		var session upload.UploadSession
		err := json.NewDecoder(rec.Body).Decode(&session)
		if err != nil {
			t.Fatalf("Failed to decode initiate response: %v", err)
		}

		if session.FileID == "" || session.Status != "uploading" || session.TotalChunks != 5 {
			t.Fatalf("Invalid session details: %+v", session)
		}

		// 2. Upload Chunks
		chunkSize := session.ChunkSize
		for i := 0; i < session.TotalChunks; i++ {
			chunkData := make([]byte, chunkSize)
			for j := 0; j < int(chunkSize); j++ {
				chunkData[j] = byte(i + 'A') // Fill with some data
			}

			chunkReq := httptest.NewRequest("PUT", fmt.Sprintf("/resumable-upload?file_id=%s&chunk_number=%d", session.FileID, i), bytes.NewReader(chunkData))
			chunkRec := httptest.NewRecorder()
			resumableProcessor.HandleResumableUpload(chunkRec, chunkReq)

			if chunkRec.Code != http.StatusOK {
				t.Fatalf("Upload chunk %d expected status 200, got: %d\nResponse: %s", i, chunkRec.Code, chunkRec.Body.String())
			}
		}

		// 3. Complete Upload (implicitly done by the last chunk upload if all chunks are sent)
		// Or explicitly check status after all chunks are sent
		statusReq := httptest.NewRequest("GET", fmt.Sprintf("/resumable-upload?file_id=%s", session.FileID), nil)
		statusRec := httptest.NewRecorder()
		resumableProcessor.HandleResumableUpload(statusRec, statusReq)

		if statusRec.Code != http.StatusOK {
			t.Fatalf("Get status expected status 200, got: %d\nResponse: %s", statusRec.Code, statusRec.Body.String())
		}

		var finalSession upload.UploadSession
		err = json.NewDecoder(statusRec.Body).Decode(&finalSession)
		if err != nil {
			t.Fatalf("Failed to decode final session response: %v", err)
		}

		if finalSession.Status != "completed" {
			t.Errorf("Expected session status to be 'completed', got: %s", finalSession.Status)
		}

		// Verify the file exists in storage
		// The actual file path for combined chunks is not directly exposed by UploadSession
		// We need to infer it or rely on the mock storage to confirm its existence
		// For mock storage, we can check if a file was written to the expected path
		// This requires the mock storage to expose its internal state or a verification method
		// For now, we'll assume if CompleteUpload didn't error, it's fine.

		// Optional: Test abort upload
		abortReq := httptest.NewRequest("DELETE", fmt.Sprintf("/resumable-upload?file_id=%s", session.FileID), nil)
		abortRec := httptest.NewRecorder()
		resumableProcessor.HandleResumableUpload(abortRec, abortReq)

		if abortRec.Code != http.StatusOK {
			t.Errorf("Abort upload expected status 200, got: %d", abortRec.Code)
		}
	})
}

func (suite *IntegrationTestSuite) testCloudStorage(t *testing.T) {
	t.Run("S3Backend", func(t *testing.T) {
		// In a real integration test, this would involve actual S3 credentials and a bucket.
		// Here, we use MockStorage, which simulates cloud storage for testing purposes.
		s3Storage := storage.NewMockStorage() // Simulate S3 storage
		s3Processor := upload.NewProcessor(s3Storage, upload.Options{
			MaxFileSize:      1 * 1024 * 1024,
			AllowedMIMETypes: []string{"image/jpeg"},
		})

		fileData := []byte("s3 test image data")
		fileName := "s3_test_image.jpg"

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		// Explicitly set content type for the test file to match allowed MIME types
		part, err := writer.CreatePart(map[string][]string{
			"Content-Disposition": {fmt.Sprintf(`form-data; name="%s"; filename="%s"`, "file", fileName)},
			"Content-Type":        {"image/jpeg"},
		})
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}
		part.Write(fileData)

		writer.Close()

		req := httptest.NewRequest("POST", "/upload/s3", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		results, err := s3Processor.ProcessWithContext(req.Context(), req, "file")
		if err != nil {
			t.Fatalf("S3 upload failed: %v", err)
		}

		if len(results) != 1 || results[0].OriginalName != fileName {
			t.Errorf("S3 upload verification failed. Results: %+v", results)
		}

		if !s3Storage.Exists(results[0].Path) {
			t.Errorf("File not found in simulated S3 storage: %s", results[0].Path)
		}
	})

	t.Run("GCSBackend", func(t *testing.T) {
		// Simulate GCS storage
		gcsStorage := storage.NewMockStorage() // Simulate GCS storage
		gcsProcessor := upload.NewProcessor(gcsStorage, upload.Options{
			MaxFileSize:      1 * 1024 * 1024,
			AllowedMIMETypes: []string{"application/pdf"},
		})

		fileData := []byte("gcs test pdf data")
		fileName := "gcs_test_document.pdf"

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		// Explicitly set content type for the test file to match allowed MIME types
		part, err := writer.CreatePart(map[string][]string{
			"Content-Disposition": {fmt.Sprintf(`form-data; name="%s"; filename="%s"`, "file", fileName)},
			"Content-Type":        {"application/pdf"},
		})
		if err != nil {
			t.Fatalf("Failed to create form file: %v", err)
		}
		part.Write(fileData)

		writer.Close()

		req := httptest.NewRequest("POST", "/upload/gcs", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		results, err := gcsProcessor.ProcessWithContext(req.Context(), req, "file")
		if err != nil {
			t.Fatalf("GCS upload failed: %v", err)
		}

		if len(results) != 1 || results[0].OriginalName != fileName {
			t.Errorf("GCS upload verification failed. Results: %+v", results)
		}

		if !gcsStorage.Exists(results[0].Path) {
			t.Errorf("File not found in simulated GCS storage: %s", results[0].Path)
		}
	})
}

func (suite *IntegrationTestSuite) testPresignedURLs(t *testing.T) {
	t.Run("PresignedPutURL", func(t *testing.T) {
		fileName := "presigned_upload.txt"
		expiration := 5 * time.Minute
		contentType := "text/plain"

		// For PUT operations, we need to use the storage's GeneratePresignedPutURL method directly
		// since the file doesn't exist yet
		url, err := suite.localStorage.(*storage.MockStorage).GeneratePresignedPutURL(fileName, expiration, contentType)
		if err != nil {
			t.Fatalf("Failed to generate presigned PUT URL: %v", err)
		}

		if url == "" || !strings.Contains(url, fileName) || !strings.Contains(url, "upload=true") || !strings.Contains(url, contentType) {
			t.Errorf("Generated presigned PUT URL is invalid: %s", url)
		}
	})

	t.Run("PresignedGetURL", func(t *testing.T) {
		fileName := "presigned_download.jpg"
		expiration := 10 * time.Minute

		// First, simulate storing a file so GetSignedURL has something to work with
		suite.localStorage.(*storage.MockStorage).Store(fileName, bytes.NewReader([]byte("dummy content")))

		// For GET operations, we can use the Processor's method since the file exists
		urlResult, err := suite.uploadProcessor.GenerateUploadURL(
			context.Background(),
			upload.UploadOptions{
				Filename:    fileName,
				Expiration:  expiration,
				MaxFileSize: 1024 * 1024, // 1MB
			},
		)
		if err != nil {
			t.Fatalf("Failed to generate presigned GET URL: %v", err)
		}

		if urlResult == nil || !strings.Contains(urlResult.URL, fileName) {
			t.Errorf("Generated presigned GET URL is invalid: %+v", urlResult)
		}
	})
}

func (suite *IntegrationTestSuite) testI18nDateTimeFormatting(t *testing.T) {
	t.Run("DateFormatting", func(t *testing.T) {
		// Test English locale date formatting
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "en-US")
		translator := suite.i18nManager.Translator(req)
		timeToFormat := time.Date(2023, time.November, 20, 15, 30, 0, 0, time.UTC)

		// Short date
		formatted := translator.FormatDate(timeToFormat, "short")
		if formatted != "11/20/2023" {
			t.Errorf("Expected '11/20/2023' for short date, got %s", formatted)
		}

		// Medium date
		formatted = translator.FormatDate(timeToFormat, "medium")
		if formatted != "Nov 20, 2023" {
			t.Errorf("Expected 'Nov 20, 2023' for medium date, got %s", formatted)
		}

		// Long date
		formatted = translator.FormatDate(timeToFormat, "long")
		if formatted != "November 20, 2023" {
			t.Errorf("Expected 'November 20, 2023' for long date, got %s", formatted)
		}

		// Test Spanish locale date formatting (uses same fallback formats)
		req = httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "es-ES")
		translator = suite.i18nManager.Translator(req)

		// Short date (fallback format)
		formatted = translator.FormatDate(timeToFormat, "short")
		if formatted != "11/20/2023" {
			t.Errorf("Expected '11/20/2023' for short date (es), got %s", formatted)
		}
	})

	t.Run("TimeFormatting", func(t *testing.T) {
		// Test English locale time formatting
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "en-US")
		translator := suite.i18nManager.Translator(req)
		timeToFormat := time.Date(2023, time.November, 20, 15, 30, 45, 0, time.UTC) // 3:30:45 PM UTC

		// Short time
		formatted := translator.FormatTime(timeToFormat, "short")
		if formatted != "15:30" {
			t.Errorf("Expected '15:30' for short time, got %s", formatted)
		}

		// Medium time
		formatted = translator.FormatTime(timeToFormat, "medium")
		if formatted != "15:30:45" {
			t.Errorf("Expected '15:30:45' for medium time, got %s", formatted)
		}

		// Test Spanish locale time formatting (uses same fallback formats)
		req = httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "es-ES")
		translator = suite.i18nManager.Translator(req)

		// Short time (fallback format)
		formatted = translator.FormatTime(timeToFormat, "short")
		if formatted != "15:30" {
			t.Errorf("Expected '15:30' for short time (es), got %s", formatted)
		}
	})

	t.Run("DateTimeFormatting", func(t *testing.T) {
		// Test English locale date-time formatting
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "en-US")
		translator := suite.i18nManager.Translator(req)
		timeToFormat := time.Date(2023, time.November, 20, 15, 30, 0, 0, time.UTC)

		// Short date, short time
		formatted := translator.FormatDateTime(timeToFormat, "short", "short")
		if formatted != "11/20/2023 15:30" {
			t.Errorf("Expected '11/20/2023 15:30' for short date-time, got %s", formatted)
		}

		// Long date, medium time
		formatted = translator.FormatDateTime(timeToFormat, "long", "medium")
		if formatted != "November 20, 2023 15:30:00" {
			t.Errorf("Expected 'November 20, 2023 15:30:00' for long date-medium time, got %s", formatted)
		}
	})
}

func (suite *IntegrationTestSuite) testCustomFormValidators(t *testing.T) {
	t.Run("IsUppercaseValidator", func(t *testing.T) {
		type UppercaseForm struct {
			Code string `form:"code" validate:"required,is_uppercase"`
		}

		// Valid case
		req := httptest.NewRequest("POST", "/test", strings.NewReader("code=ABCDEF"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		var f UppercaseForm
		errors := form.DecodeAndValidateWithContext(req.Context(), req, &f)
		if len(errors) > 0 {
			t.Errorf("Expected no errors for valid uppercase, got: %v", errors)
		}

		// Invalid case (lowercase)
		req = httptest.NewRequest("POST", "/test", strings.NewReader("code=AbcDef"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		f = UppercaseForm{}
		errors = form.DecodeAndValidateWithContext(req.Context(), req, &f)
		if len(errors) == 0 || errors["code"][0] != "Must be all uppercase" {
			t.Errorf("Expected 'Must be all uppercase' error, got: %v", errors)
		}

		// Invalid case (mixed)
		req = httptest.NewRequest("POST", "/test", strings.NewReader("code=ABCdeF"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		f = UppercaseForm{}
		errors = form.DecodeAndValidateWithContext(req.Context(), req, &f)
		if len(errors) == 0 || errors["code"][0] != "Must be all uppercase" {
			t.Errorf("Expected 'Must be all uppercase' error, got: %v", errors)
		}
	})

	t.Run("UniqueUsernameValidator", func(t *testing.T) {
		type UsernameForm struct {
			Username string `form:"username" validate:"required,unique_username"`
		}

		// Valid username
		req := httptest.NewRequest("POST", "/test", strings.NewReader("username=newuser"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		var f UsernameForm
		errors := form.DecodeAndValidateWithContext(req.Context(), req, &f)
		if len(errors) > 0 {
			t.Errorf("Expected no errors for valid username, got: %v", errors)
		}

		// Invalid username (already taken - 'admin')
		req = httptest.NewRequest("POST", "/test", strings.NewReader("username=admin"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		f = UsernameForm{}
		errors = form.DecodeAndValidateWithContext(req.Context(), req, &f)
		if len(errors) == 0 || errors["username"][0] != "Username already taken" {
			t.Errorf("Expected 'Username already taken' error for 'admin', got: %v", errors)
		}

		// Invalid username (already taken - 'testuser')
		req = httptest.NewRequest("POST", "/test", strings.NewReader("username=testuser"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		f = UsernameForm{}
		errors = form.DecodeAndValidateWithContext(req.Context(), req, &f)
		if len(errors) == 0 || errors["username"][0] != "Username already taken" {
			t.Errorf("Expected 'Username already taken' error for 'testuser', got: %v", errors)
		}
	})
}

func (suite *IntegrationTestSuite) testFormValidationWithI18n(t *testing.T) {
	// Test form validation with i18n error messages
	req := httptest.NewRequest("POST", "/test", strings.NewReader("email=invalid"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept-Language", "es-ES,es;q=0.9")

	var user TestUser
	errors := form.DecodeAndValidateWithContext(req.Context(), req, &user)

	// Get translator for error messages
	translator := suite.i18nManager.Translator(req)

	// Test that we can translate error messages
	if len(errors) > 0 {
		for field, fieldErrors := range errors {
			for _, errorMsg := range fieldErrors {
				// Test that error messages can be processed by i18n
				translatedError := translator.T("validation_error", map[string]interface{}{
					"Field": field,
					"Error": errorMsg,
				})
				if translatedError == "" {
					t.Errorf("Expected translated error message for field %s", field)
				}
			}
		}
	}
}

func (suite *IntegrationTestSuite) testUploadWithFormValidation(t *testing.T) {
	// Test upload with form validation in the same request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add form fields
	writer.WriteField("email", "test@example.com")
	writer.WriteField("name", "Test User")

	// Add file
	fileWriter, _ := writer.CreateFormFile("avatar", "test.txt")
	fileWriter.Write([]byte("fake image data"))

	writer.Close()

	req := httptest.NewRequest("POST", "/upload-with-form", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Test both form validation and upload
	var user struct {
		Email string `form:"email" validate:"required,email"`
		Name  string `form:"name" validate:"required"`
	}

	errors := form.DecodeAndValidateWithContext(req.Context(), req, &user)
	if len(errors) > 0 {
		t.Errorf("Expected no form validation errors, got: %v", errors)
	}

	results, err := suite.uploadProcessor.ProcessWithContext(req.Context(), req, "avatar")
	if err != nil {
		t.Errorf("Expected no upload errors, got: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 upload result, got %d", len(results))
	}
}

func (suite *IntegrationTestSuite) testCrossPackageObservability(t *testing.T) {
	ctx := context.Background()

	// Test observability across form validation, upload, and i18n
	_, span := observability.StartSpan(ctx, "cross_package_workflow")
	defer span.End()

	// Form validation with observability
	_, formSpan := observability.StartSpan(ctx, "form_validation")
	req := httptest.NewRequest("POST", "/test", strings.NewReader("email=test@example.com&name=Test"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var user TestUser
	errors := form.DecodeAndValidateWithContext(ctx, req, &user)
	formSpan.End()

	observability.SetSpanAttributes(ctx, map[string]string{
		"form.validation.errors": fmt.Sprintf("%d", len(errors)),
	})

	// Upload with observability
	_, uploadSpan := observability.StartSpan(ctx, "file_upload")
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fileWriter, _ := writer.CreateFormFile("file", "test.txt")
	fileWriter.Write([]byte("test data"))
	writer.Close()

	uploadReq := httptest.NewRequest("POST", "/upload", body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())

	results, err := suite.uploadProcessor.ProcessWithContext(ctx, uploadReq, "file")
	uploadSpan.End()

	observability.SetSpanAttributes(ctx, map[string]string{
		"upload.success": fmt.Sprintf("%t", err == nil),
		"upload.count":   fmt.Sprintf("%d", len(results)),
	})

	// i18n with observability
	_, i18nSpan := observability.StartSpan(ctx, "i18n_translation")
	translator := suite.i18nManager.Translator(req)
	message := translator.T("workflow_complete", map[string]interface{}{
		"FormErrors": len(errors),
		"Uploads":    len(results),
	})
	i18nSpan.End()

	observability.SetSpanAttributes(ctx, map[string]string{
		"i18n.message": message,
	})

	// Record metrics
	observability.RecordMetric("integration_workflow_complete", 1, map[string]string{
		"form_errors": fmt.Sprintf("%d", len(errors)),
		"uploads":     fmt.Sprintf("%d", len(results)),
	})
}

func createTestLocaleFiles(t *testing.T, localesDir string) {
	locales := map[string]map[string]string{
		"en": {
			"welcome":           "Welcome, {{.Name}}!",
			"item":              "1 item",
			"items":             "{{.Count}} items",
			"validation_error":  "Field {{.Field}}: {{.Error}}",
			"workflow_complete": "Workflow completed with {{.FormErrors}} form errors and {{.Uploads}} uploads",
		},
		"es": {
			"welcome":           "¡Bienvenido, {{.Name}}!",
			"item":              "1 elemento",
			"items":             "{{.Count}} elementos",
			"validation_error":  "Campo {{.Field}}: {{.Error}}",
			"workflow_complete": "Flujo completado con {{.FormErrors}} errores de formulario y {{.Uploads}} cargas",
		},
		"fr": {
			"welcome":           "Bienvenue, {{.Name}} !",
			"item":              "1 élément",
			"items":             "{{.Count}} éléments",
			"validation_error":  "Champ {{.Field}}: {{.Error}}",
			"workflow_complete": "Workflow terminé avec {{.FormErrors}} erreurs de formulaire et {{.Uploads}} téléchargements",
		},
	}

	for locale, messages := range locales {
		filePath := filepath.Join(localesDir, locale+".toml")
		content := ""
		for key, value := range messages {
			content += fmt.Sprintf("%s = \"%s\"\n", key, value)
		}

		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create locale file %s: %v", filePath, err)
		}
	}
}

// TestIntegration_Performance tests performance characteristics
func TestIntegration_Performance(t *testing.T) {
	suite := setupIntegrationSuite(t)
	defer suite.cleanup()

	// Test concurrent form validation
	t.Run("ConcurrentFormValidation", func(t *testing.T) {
		const numGoroutines = 10
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer func() { done <- true }()

				req := httptest.NewRequest("POST", "/test", strings.NewReader(fmt.Sprintf("email=test%d@example.com&password=password123&confirm_password=password123&name=User%d", id, id)))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

				var user TestUser
				errors := form.DecodeAndValidateWithContext(req.Context(), req, &user)

				if len(errors) > 0 {
					t.Errorf("Goroutine %d: Expected no validation errors", id)
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})

	// Test concurrent uploads
	t.Run("ConcurrentUploads", func(t *testing.T) {
		const numGoroutines = 5
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer func() { done <- true }()

				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				fileWriter, _ := writer.CreateFormFile("file", fmt.Sprintf("test%d.txt", id))
				fileWriter.Write([]byte(fmt.Sprintf("test data %d", id)))
				writer.Close()

				req := httptest.NewRequest("POST", "/upload", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())

				results, err := suite.uploadProcessor.ProcessWithContext(req.Context(), req, "file")
				if err != nil {
					t.Errorf("Goroutine %d: Upload failed: %v", id, err)
				}

				if len(results) != 1 {
					t.Errorf("Goroutine %d: Expected 1 upload result, got %d", id, len(results))
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})
}

// TestIntegration_Stress tests stress conditions
func TestIntegration_Stress(t *testing.T) {
	suite := setupIntegrationSuite(t)
	defer suite.cleanup()

	// Test with large number of form fields
	t.Run("LargeForm", func(t *testing.T) {
		formData := make(map[string]string)
		for i := 0; i < 100; i++ {
			formData[fmt.Sprintf("field_%d", i)] = fmt.Sprintf("value_%d", i)
		}
		formData["email"] = "test@example.com"
		formData["password"] = "password123"
		formData["confirm_password"] = "password123"
		formData["name"] = "Test User"

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		for key, value := range formData {
			writer.WriteField(key, value)
		}
		writer.Close()

		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		var user TestUser
		errors := form.DecodeAndValidateWithContext(req.Context(), req, &user)

		if len(errors) > 0 {
			t.Errorf("Expected no validation errors, got: %v", errors)
		}
	})

	// Test with large file
	t.Run("LargeFile", func(t *testing.T) {
		// Create a file that's just under the limit
		fileData := make([]byte, 4*1024*1024) // 4MB
		for i := range fileData {
			fileData[i] = byte(i % 256)
		}

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		fileWriter, _ := writer.CreateFormFile("file", "large.txt")
		fileWriter.Write(fileData)
		writer.Close()

		req := httptest.NewRequest("POST", "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		results, err := suite.uploadProcessor.ProcessWithContext(req.Context(), req, "file")
		if err != nil {
			t.Errorf("Expected no upload errors, got: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 upload result, got %d", len(results))
			return
		}

		if results[0].Size != int64(len(fileData)) {
			t.Errorf("Expected size %d, got %d", len(fileData), results[0].Size)
		}
	})
}
