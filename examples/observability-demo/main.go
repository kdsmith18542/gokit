package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/kdsmith18542/gokit/form"
	"github.com/kdsmith18542/gokit/i18n"
	"github.com/kdsmith18542/gokit/observability"
	"github.com/kdsmith18542/gokit/upload"
	"github.com/kdsmith18542/gokit/upload/storage"
)

// UserRegistrationForm demonstrates form validation with observability
type UserRegistrationForm struct {
	Email           string `form:"email" validate:"required,email"`
	Password        string `form:"password" validate:"required,min=8"`
	ConfirmPassword string `form:"confirm_password" validate:"required,eqfield=Password"`
	Name            string `form:"name" validate:"required" sanitize:"trim"`
	Age             int    `form:"age" validate:"required,min=18"`
}

func main() {
	// Initialize observability with OpenTelemetry
	err := observability.Init(observability.Config{
		ServiceName:    "gokit-demo",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		EnableTracing:  true,
		EnableMetrics:  true,
		EnableLogging:  true,
	})
	if err != nil {
		log.Fatalf("Failed to initialize observability: %v", err)
	}

	// Enable observability for all packages
	form.EnableObservability()
	i18n.EnableObservability()
	upload.EnableObservability()

	// Initialize i18n manager
	i18nManager := i18n.NewManager("./locales")
	i18nManager.SetDefaultLocale("en")
	i18nManager.SetFallbackLocale("en")

	// Initialize upload processor with observable storage
	localStorage := storage.NewLocal("./uploads")
	observableStorage := storage.NewObservableStorage(localStorage, "local")
	defer observableStorage.Close()

	uploadProcessor := upload.NewProcessor(observableStorage, upload.Options{
		MaxFileSize:      5 * 1024 * 1024, // 5MB
		AllowedMIMETypes: []string{"image/jpeg", "image/png", "image/gif"},
		MaxFiles:         1,
	})

	// Add upload hooks for observability
	uploadProcessor.OnSuccess(func(ctx context.Context, result upload.Result) {
		observability.LogInfo(ctx, "File uploaded successfully", map[string]string{
			"file_name": result.OriginalName,
			"file_size": fmt.Sprintf("%d", result.Size),
			"file_url":  result.URL,
		})
	})

	uploadProcessor.OnError(func(ctx context.Context, result upload.Result, err error) {
		observability.LogError(ctx, "File upload failed", err, map[string]string{
			"file_name": result.OriginalName,
			"file_size": fmt.Sprintf("%d", result.Size),
		})
	})

	// Set up HTTP handlers
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		handleRegistration(w, r, i18nManager, uploadProcessor)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		handleHealth(w, r, i18nManager)
	})

	// Start server
	fmt.Println("GoKit Observability Demo running at http://localhost:8080")
	fmt.Println("Endpoints:")
	fmt.Println("  POST /register - User registration with file upload")
	fmt.Println("  GET  /health   - Health check with i18n")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleRegistration(w http.ResponseWriter, r *http.Request, i18nManager *i18n.Manager, uploadProcessor *upload.Processor) {
	ctx := r.Context()

	// Start a span for the entire registration process
	ctx, span := observability.StartSpan(ctx, "user_registration")
	defer span.End()

	// Set span attributes
	observability.SetSpanAttributes(ctx, map[string]string{
		"endpoint": "/register",
		"method":   r.Method,
	})

	// Get translator for i18n
	translator := i18nManager.Translator(r)

	// Parse and validate form
	var userForm UserRegistrationForm
	errors := form.DecodeAndValidateWithContext(ctx, r, &userForm)

	if len(errors) > 0 {
		// Record validation errors
		observability.LogError(ctx, "Form validation failed", nil, map[string]string{
			"error_count": fmt.Sprintf("%d", len(errors)),
		})

		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Validation errors:\n")
		for field, fieldErrors := range errors {
			for _, err := range fieldErrors {
				fmt.Fprintf(w, "  %s: %s\n", field, err)
			}
		}
		return
	}

	// Process file upload if present
	if r.MultipartForm != nil && len(r.MultipartForm.File["avatar"]) > 0 {
		// Start upload span
		uploadCtx, uploadSpan := observability.StartSpan(ctx, "avatar_upload")

		results, err := uploadProcessor.ProcessWithContext(uploadCtx, r, "avatar")
		uploadSpan.End()

		if err != nil {
			observability.LogError(uploadCtx, "Avatar upload failed", err, map[string]string{
				"user_email": userForm.Email,
			})
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Upload error: %v\n", err)
			return
		}

		// Log successful upload
		for _, result := range results {
			observability.LogInfo(uploadCtx, "Avatar uploaded", map[string]string{
				"user_email": userForm.Email,
				"file_url":   result.URL,
			})
		}
	}

	// Record successful registration
	observability.LogInfo(ctx, "User registered successfully", map[string]string{
		"user_email": userForm.Email,
		"user_name":  userForm.Name,
		"user_age":   fmt.Sprintf("%d", userForm.Age),
	})

	// Record metrics
	observability.RecordMetric("user_registrations", 1, map[string]string{
		"status": "success",
	})

	// Send success response
	w.WriteHeader(http.StatusOK)
	successMessage := translator.T("registration_success", map[string]interface{}{
		"Name":  userForm.Name,
		"Email": userForm.Email,
	})
	fmt.Fprintf(w, "%s\n", successMessage)
}

func handleHealth(w http.ResponseWriter, r *http.Request, i18nManager *i18n.Manager) {
	ctx := r.Context()

	// Start a span for health check
	ctx, span := observability.StartSpan(ctx, "health_check")
	defer span.End()

	// Get translator
	translator := i18nManager.Translator(r)

	// Add span events for health check steps
	observability.AddSpanEvent(ctx, "health_check_started", map[string]string{
		"timestamp": time.Now().Format(time.RFC3339),
	})

	// Simulate some health checks
	time.Sleep(10 * time.Millisecond) // Simulate work

	observability.AddSpanEvent(ctx, "health_check_completed", map[string]string{
		"status": "healthy",
	})

	// Record health check metric
	observability.RecordMetric("health_checks", 1, map[string]string{
		"status": "success",
	})

	// Send health response
	w.WriteHeader(http.StatusOK)
	healthMessage := translator.T("health_status", map[string]interface{}{
		"Status": "healthy",
		"Time":   time.Now().Format(time.RFC3339),
	})
	fmt.Fprintf(w, "%s\n", healthMessage)
}
