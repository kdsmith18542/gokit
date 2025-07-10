package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kdsmith18542/gokit/form"
	"github.com/kdsmith18542/gokit/i18n"
	"github.com/kdsmith18542/gokit/upload"
	"github.com/kdsmith18542/gokit/upload/storage"
)

// AdvancedForm demonstrates complex conditional validation
type AdvancedForm struct {
	AccountType string  `form:"account_type" validate:"required"`
	CompanyName string  `form:"company_name" validate:"required_if=account_type:business"`
	TaxID       string  `form:"tax_id" validate:"required_if=account_type:business"`
	StartDate   string  `form:"start_date" validate:"required"`
	EndDate     string  `form:"end_date" validate:"required,date_after=start_date"`
	MinPrice    float64 `form:"min_price" validate:"required,numeric"`
	MaxPrice    float64 `form:"max_price" validate:"required,numeric,gtfield=min_price"`
	Password    string  `form:"password" validate:"required,min=8"`
	ConfirmPass string  `form:"confirm_password" validate:"required,eqfield=password"`
	Username    string  `form:"username" validate:"required"`
	Email       string  `form:"email" validate:"required,email,nefield=username"`
}

func main() {
	fmt.Println("GoKit Advanced Features Demo")
	fmt.Println("============================")

	// 1. Advanced Conditional Validation
	demoAdvancedValidation()

	// 2. Locale-Aware Formatting
	demoLocaleFormatting()

	// 3. Post-Processing Hooks
	demoUploadHooks()

	fmt.Println("\nDemo completed successfully!")
}

func demoAdvancedValidation() {
	fmt.Println("\n1. Advanced Conditional Validation")
	fmt.Println("----------------------------------")

	// Test case 1: Business account with missing required fields
	fmt.Println("\nTest 1: Business account with missing company name and tax ID")
	req1 := createTestRequest(map[string]string{
		"account_type":     "business",
		"start_date":       "2024-01-01",
		"end_date":         "2024-12-31",
		"min_price":        "100.00",
		"max_price":        "200.00",
		"password":         "secret123",
		"confirm_password": "secret123",
		"username":         "john_doe",
		"email":            "john@example.com",
	})

	var form1 AdvancedForm
	errors1 := form.DecodeAndValidate(req1, &form1)
	if len(errors1) > 0 {
		fmt.Println("Validation errors:")
		for field, fieldErrors := range errors1 {
			for _, err := range fieldErrors {
				fmt.Printf("  %s: %s\n", field, err)
			}
		}
	}

	// Test case 2: Valid business account
	fmt.Println("\nTest 2: Valid business account")
	req2 := createTestRequest(map[string]string{
		"account_type":     "business",
		"company_name":     "Acme Corp",
		"tax_id":           "123456789",
		"start_date":       "2024-01-01",
		"end_date":         "2024-12-31",
		"min_price":        "100.00",
		"max_price":        "200.00",
		"password":         "secret123",
		"confirm_password": "secret123",
		"username":         "john_doe",
		"email":            "john@example.com",
	})

	var form2 AdvancedForm
	errors2 := form.DecodeAndValidate(req2, &form2)
	if len(errors2) == 0 {
		fmt.Println("✓ All validations passed!")
		fmt.Printf("  Account Type: %s\n", form2.AccountType)
		fmt.Printf("  Company Name: %s\n", form2.CompanyName)
		fmt.Printf("  Tax ID: %s\n", form2.TaxID)
	}

	// Test case 3: Invalid date range
	fmt.Println("\nTest 3: Invalid date range (end date before start date)")
	req3 := createTestRequest(map[string]string{
		"account_type":     "personal",
		"start_date":       "2024-12-31",
		"end_date":         "2024-01-01",
		"min_price":        "100.00",
		"max_price":        "200.00",
		"password":         "secret123",
		"confirm_password": "secret123",
		"username":         "john_doe",
		"email":            "john@example.com",
	})

	var form3 AdvancedForm
	errors3 := form.DecodeAndValidate(req3, &form3)
	if len(errors3) > 0 {
		fmt.Println("Validation errors:")
		for field, fieldErrors := range errors3 {
			for _, err := range fieldErrors {
				fmt.Printf("  %s: %s\n", field, err)
			}
		}
	}
}

func demoLocaleFormatting() {
	fmt.Println("\n2. Locale-Aware Formatting")
	fmt.Println("---------------------------")

	// Create a temporary directory for locale files
	tempDir, err := os.MkdirTemp("", "gokit-locales")
	if err != nil {
		log.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test locale files
	enContent := `welcome = "Welcome to GoKit!"
relative_time.future = "in {{.Value}} {{.Unit}}"
relative_time.past = "{{.Value}} {{.Unit}} ago"
relative_time.second.one = "second"
relative_time.second.other = "seconds"
relative_time.minute.one = "minute"
relative_time.minute.other = "minutes"
relative_time.hour.one = "hour"
relative_time.hour.other = "hours"
relative_time.day.one = "day"
relative_time.day.other = "days"`

	deContent := `welcome = "Willkommen bei GoKit!"
relative_time.future = "in {{.Value}} {{.Unit}}"
relative_time.past = "vor {{.Value}} {{.Unit}}"
relative_time.second.one = "Sekunde"
relative_time.second.other = "Sekunden"
relative_time.minute.one = "Minute"
relative_time.minute.other = "Minuten"
relative_time.hour.one = "Stunde"
relative_time.hour.other = "Stunden"
relative_time.day.one = "Tag"
relative_time.day.other = "Tage"`

	// Write locale files
	if err := os.WriteFile(filepath.Join(tempDir, "en.toml"), []byte(enContent), 0644); err != nil {
		log.Fatalf("Failed to write en.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "de.toml"), []byte(deContent), 0644); err != nil {
		log.Fatalf("Failed to write de.toml: %v", err)
	}

	// Create i18n manager
	manager := i18n.NewManager(tempDir)
	manager.SetDefaultFormats()

	// Test number formatting
	fmt.Println("\nNumber Formatting:")
	req := createRequestWithLocale("en")
	translator := manager.Translator(req)

	number := 1234567.89
	fmt.Printf("  US: %s\n", translator.FormatNumber(number))

	req = createRequestWithLocale("de")
	translator = manager.Translator(req)
	fmt.Printf("  DE: %s\n", translator.FormatNumber(number))

	// Test currency formatting
	fmt.Println("\nCurrency Formatting:")
	req = createRequestWithLocale("en")
	translator = manager.Translator(req)

	amount := 1234.56
	fmt.Printf("  US USD: %s\n", translator.FormatCurrency(amount, "USD"))
	fmt.Printf("  US USD with code: %s\n", translator.FormatCurrencyWithCode(amount, "USD"))

	req = createRequestWithLocale("de")
	translator = manager.Translator(req)
	fmt.Printf("  DE EUR: %s\n", translator.FormatCurrency(amount, "EUR"))

	// Test percentage formatting
	fmt.Println("\nPercentage Formatting:")
	req = createRequestWithLocale("en")
	translator = manager.Translator(req)

	percentage := 0.1234
	fmt.Printf("  US: %s\n", translator.FormatPercentage(percentage))

	req = createRequestWithLocale("de")
	translator = manager.Translator(req)
	fmt.Printf("  DE: %s\n", translator.FormatPercentage(percentage))

	// Test scientific notation
	fmt.Println("\nScientific Notation:")
	req = createRequestWithLocale("en")
	translator = manager.Translator(req)

	scientific := 1234567.89
	fmt.Printf("  US: %s\n", translator.FormatScientific(scientific, 2))

	req = createRequestWithLocale("de")
	translator = manager.Translator(req)
	fmt.Printf("  DE: %s\n", translator.FormatScientific(scientific, 2))

	// Test relative time formatting
	fmt.Println("\nRelative Time Formatting:")
	now := time.Now()

	req = createRequestWithLocale("en")
	translator = manager.Translator(req)

	pastTime := now.Add(-2 * time.Hour)
	fmt.Printf("  US (2 hours ago): %s\n", translator.FormatRelativeTime(pastTime, now))

	futureTime := now.Add(3 * 24 * time.Hour)
	fmt.Printf("  US (in 3 days): %s\n", translator.FormatRelativeTime(futureTime, now))

	// Test number parsing
	fmt.Println("\nNumber Parsing:")
	req = createRequestWithLocale("en")
	translator = manager.Translator(req)

	parsed, err := translator.ParseNumber("1,234.56")
	if err == nil {
		fmt.Printf("  Parsed US number: %f\n", parsed)
	}

	req = createRequestWithLocale("de")
	translator = manager.Translator(req)

	parsed, err = translator.ParseNumber("1.234,56")
	if err == nil {
		fmt.Printf("  Parsed DE number: %f\n", parsed)
	}
}

func demoUploadHooks() {
	fmt.Println("\n3. Post-Processing Hooks")
	fmt.Println("-------------------------")

	// Create mock storage
	mockStorage := storage.NewMockStorage()

	// Create upload processor with hooks
	processor := upload.NewProcessor(mockStorage, upload.Options{
		MaxFileSize:      10 * 1024 * 1024, // 10MB
		AllowedMIMETypes: []string{"image/jpeg", "image/png", "application/octet-stream"},
	})

	// Register success hooks
	processor.OnSuccess(func(ctx context.Context, result upload.Result) {
		fmt.Printf("  ✓ Success hook: File '%s' uploaded successfully\n", result.OriginalName)
		fmt.Printf("    Size: %d bytes, URL: %s\n", result.Size, result.URL)
	})

	processor.OnSuccess(func(ctx context.Context, result upload.Result) {
		fmt.Printf("  ✓ Second hook: Generating thumbnail for '%s'\n", result.OriginalName)
		// Simulate thumbnail generation
		time.Sleep(10 * time.Millisecond)
		fmt.Printf("    Thumbnail generated: %s_thumb.jpg\n", result.Path)
	})

	// Register error hooks
	processor.OnError(func(ctx context.Context, result upload.Result, err error) {
		fmt.Printf("  ✗ Error hook: Upload failed for '%s': %v\n", result.OriginalName, err)
	})

	processor.OnError(func(ctx context.Context, result upload.Result, err error) {
		fmt.Printf("  ✗ Second error hook: Sending failure notification\n")
		// Simulate sending notification
		time.Sleep(5 * time.Millisecond)
		fmt.Printf("    Notification sent to admin\n")
	})

	// Test successful upload
	fmt.Println("\nTesting successful upload:")
	req := createUploadRequest("test.jpg", "test file content")

	ctx := context.Background()
	results, err := processor.ProcessWithContext(ctx, req, "file")
	if err != nil {
		log.Printf("Upload failed: %v", err)
	} else {
		fmt.Printf("  Upload completed: %d files processed\n", len(results))
	}

	// Test upload with validation error
	fmt.Println("\nTesting upload with validation error:")
	processor.SetOptions(upload.Options{
		MaxFileSize:      1024, // 1KB limit
		AllowedMIMETypes: []string{"image/jpeg", "image/png", "application/octet-stream"},
	})

	req = createUploadRequest("large.jpg", string(make([]byte, 2048))) // 2KB file

	results, err = processor.ProcessWithContext(ctx, req, "file")
	if err != nil {
		fmt.Printf("  Expected error: %v\n", err)
	}
}

// Helper functions

func createTestRequest(data map[string]string) *http.Request {
	values := url.Values{}
	for key, value := range data {
		values.Set(key, value)
	}

	req, _ := http.NewRequest("POST", "/test", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func createRequestWithLocale(locale string) *http.Request {
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Language", locale)
	return req
}

func createUploadRequest(filename, content string) *http.Request {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", filename)
	part.Write([]byte(content))
	writer.Close()

	req, _ := http.NewRequest("POST", "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}
