// Package upload provides utilities for handling file uploads in Go web applications.
// It supports various storage backends (S3, GCS, Azure, local) and includes
// features like multipart form parsing, file validation, and resumable uploads.
package upload

import (
	"context"
	"encoding/json"
	"net/http"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// uploadResultsKey is the context key for multiple upload results.
	uploadResultsKey contextKey = "uploadResults"
	// uploadResultKey is the context key for a single upload result.
	uploadResultKey contextKey = "uploadResult"
)

// ErrorHandler is a function type for handling upload errors.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

// DefaultErrorHandler returns a JSON error response for upload failures.
func DefaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	response := map[string]interface{}{
		"error":   "Upload failed",
		"message": err.Error(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// Middleware returns middleware that processes file uploads and stores
// the results in the request context.
//
// This middleware automatically handles multipart form parsing, file validation,
// and upload processing. It supports multiple file uploads and stores the results
// in the request context for easy access in handlers.
//
// Example usage:
//
//	func main() {
//	    // Initialize storage backend
//	    s3Storage, _ := storage.NewS3(storage.S3Config{
//	        Bucket: "my-bucket",
//	        Region: "us-west-2",
//	    })
//
//	    // Create upload processor
//	    processor := NewProcessor(s3Storage, Options{
//	        MaxFileSize: 10 * 1024 * 1024, // 10MB
//	        AllowedMIMETypes: []string{"image/jpeg", "image/png", "application/pdf"},
//	        ValidateChecksum: true,
//	    })
//
//	    // Register post-processing hooks
//	    processor.OnSuccess(func(ctx context.Context, result Result) {
//	        log.Printf("File uploaded: %s", result.URL)
//	    })
//
//	    mux := http.NewServeMux()
//
//	    // Apply middleware to upload routes
//	    mux.HandleFunc("/upload", Middleware(processor, "files", nil)(uploadHandler))
//	    mux.HandleFunc("/api/upload", Middleware(processor, "file", JSONUploadErrorHandler)(apiUploadHandler))
//
//	    http.ListenAndServe(":8080", mux)
//	}
//
//	func uploadHandler(w http.ResponseWriter, r *http.Request) {
//	    results := ResultsFromContext(r.Context())
//
//	    // Files are already uploaded and validated
//	    w.Header().Set("Content-Type", "application/json")
//	    json.NewEncoder(w).Encode(map[string]interface{}{
//	        "success": true,
//	        "files": results,
//	        "count": len(results),
//	    })
//	}
//
//	func apiUploadHandler(w http.ResponseWriter, r *http.Request) {
//	    results := MustUploadResultsFromContext(r.Context())
//
//	    // Process upload results
//	    for _, result := range results {
//	        // Update database, generate thumbnails, etc.
//	        log.Printf("Processed: %s -> %s", result.OriginalName, result.URL)
//	    }
//
//	    w.Header().Set("Content-Type", "application/json")
//	    if err := json.NewEncoder(w).Encode(map[string]interface{}{
//	        "status": "success",
//	        "uploaded": len(results),
//	    }); err != nil {
//	        http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
//	        return
//	    }
//	}
//
// The middleware supports:
// - Multiple file uploads from a single form field
// - Automatic file validation (size, type, checksum)
// - Post-processing hooks for additional operations
// - Context injection for easy access in handlers
// - Custom error handling for different response formats
func Middleware(processor *Processor, fieldName string, errorHandler ErrorHandler) func(http.Handler) http.Handler {
	return sharedUploadMiddleware(
		func(r *http.Request) (interface{}, error) {
			return processor.Process(r, fieldName)
		},
		uploadResultsKey,
		errorHandler,
	)
}

// ResultsFromContext retrieves the upload results from the request context.
// Returns nil if no results were found in the context.
//
// Example:
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    results := ResultsFromContext(r.Context())
//	    if results == nil {
//	        http.Error(w, "No upload results", http.StatusBadRequest)
//	        return
//	    }
//
//	    // Process multiple upload results
//	    for _, result := range results {
//	        log.Printf("Uploaded: %s -> %s", result.OriginalName, result.URL)
//
//	        // Update database with file information
//	        fileRecord := &FileRecord{
//	            Name: result.OriginalName,
//	            Path: result.Path,
//	            URL:  result.URL,
//	            Size: result.Size,
//	        }
//	        db.CreateFile(fileRecord)
//	    }
//
//	    w.Header().Set("Content-Type", "application/json")
//	    if err := json.NewEncoder(w).Encode(map[string]interface{}{
//	        "uploaded": len(results),
//	        "files": results,
//	    }); err != nil {
//	        http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
//	        return
//	    }
//	}
//
// For guaranteed access (when you're certain the middleware is applied):
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    results := MustUploadResultsFromContext(r.Context())
//
//	    // results is guaranteed to be non-nil
//	    for _, result := range results {
//	        processFile(result)
//	    }
//	}
func ResultsFromContext(ctx context.Context) []Result {
	if results, ok := ctx.Value(uploadResultsKey).([]Result); ok {
		return results
	}
	return nil
}

// MustUploadResultsFromContext retrieves the upload results from the request context.
// Panics if no results were found in the context.
func MustUploadResultsFromContext(ctx context.Context) []Result {
	results := ResultsFromContext(ctx)
	if results == nil {
		panic("upload: Upload results not found in context. Did you apply the UploadMiddleware?")
	}
	return results
}

// SingleUploadMiddleware returns middleware that processes a single file upload
// and stores the result in the request context.
//
// This middleware is similar to Middleware but is optimized for single file
// uploads. It stores a single Result instead of a slice, making it more convenient
// for handlers that expect only one file.
//
// Example usage:
//
//	func main() {
//	    // Initialize storage
//	    localStorage, _ := storage.NewLocal(storage.LocalConfig{
//	        BasePath: "/var/uploads",
//	        BaseURL:  "https://example.com/uploads",
//	    })
//
//	    // Create processor for avatar uploads
//	    avatarProcessor := NewProcessor(localStorage, Options{
//	        MaxFileSize: 5 * 1024 * 1024, // 5MB
//	        AllowedMIMETypes: []string{"image/jpeg", "image/png", "image/gif"},
//	    })
//
//	    mux := http.NewServeMux()
//
//	    // Single file upload for avatar
//	    mux.HandleFunc("/avatar", SingleUploadMiddleware(avatarProcessor, "avatar", nil)(avatarHandler))
//
//	    // Multiple file upload for gallery
//	    mux.HandleFunc("/gallery", Middleware(avatarProcessor, "images", nil)(galleryHandler))
//
//	    http.ListenAndServe(":8080", mux)
//	}
//
//	func avatarHandler(w http.ResponseWriter, r *http.Request) {
//	    result := SingleUploadResultFromContext(r.Context())
//	    if result == nil {
//	        http.Error(w, "No avatar uploaded", http.StatusBadRequest)
//	        return
//	    }
//
//	    // Update user's avatar
//	    userID := getUserID(r)
//	    updateUserAvatar(userID, result.URL)
//
//	    w.Header().Set("Content-Type", "application/json")
//	    json.NewEncoder(w).Encode(map[string]interface{}{
//	        "success": true,
//	        "avatar_url": result.URL,
//	    })
//	}
//
//	func galleryHandler(w http.ResponseWriter, r *http.Request) {
//	    results := ResultsFromContext(r.Context())
//
//	    // Process multiple images
//	    for _, result := range results {
//	        addToGallery(result.URL)
//	    }
//
//	    w.Header().Set("Content-Type", "application/json")
//	    if err := json.NewEncoder(w).Encode(map[string]interface{}{
//	        "success": true,
//	        "images": results,
//	    }); err != nil {
//	        http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
//	        return
//	    }
//	}
//
// Use SingleUploadMiddleware when:
// - You expect only one file per request
// - You want simpler handler code (no slice handling)
// - You're building single-file upload features (avatar, profile picture, etc.)
// - You want type safety for single file operations
func SingleUploadMiddleware(processor *Processor, fieldName string, errorHandler ErrorHandler) func(http.Handler) http.Handler {
	return sharedUploadMiddleware(
		func(r *http.Request) (interface{}, error) {
			return processor.ProcessSingle(r, fieldName)
		},
		uploadResultKey,
		errorHandler,
	)
}

// SingleUploadResultFromContext retrieves the single upload result from the request context.
// Returns nil if no result was found in the context.
func SingleUploadResultFromContext(ctx context.Context) *Result {
	if result, ok := ctx.Value(uploadResultKey).(*Result); ok {
		return result
	}
	return nil
}

// MustSingleUploadResultFromContext retrieves the single upload result from the request context.
// Panics if no result was found in the context.
func MustSingleUploadResultFromContext(ctx context.Context) *Result {
	result := SingleUploadResultFromContext(ctx)
	if result == nil {
		panic("upload: Single upload result not found in context. Did you apply the SingleUploadMiddleware?")
	}
	return result
}

// MiddlewareWithContext returns middleware that processes file uploads
// with context support for cancellation and timeouts.
//
// This middleware is similar to Middleware but uses ProcessWithContext,
// which provides better observability and context cancellation support. Use this
// version when you need tracing, metrics, or context-aware upload processing.
//
// Example usage:
//
//	func main() {
//	    // Initialize storage with context support
//	    gcsStorage, _ := storage.NewGCS(storage.GCSConfig{
//	        Bucket:   "my-bucket",
//	        ProjectID: "my-project",
//	    })
//
//	    processor := NewProcessor(gcsStorage, Options{
//	        MaxFileSize: 50 * 1024 * 1024, // 50MB
//	        AllowedMIMETypes: []string{"application/pdf", "text/plain"},
//	    })
//
//	    // Register context-aware hooks
//	    processor.OnSuccess(func(ctx context.Context, result Result) {
//	        // Context is available for tracing and cancellation
//	        span := trace.SpanFromContext(ctx)
//	        span.AddEvent("file_uploaded", trace.WithAttributes(
//	            attribute.String("file_name", result.OriginalName),
//	            attribute.String("file_url", result.URL),
//	        ))
//
//	        // Process with context
//	        go processDocument(ctx, result)
//	    })
//
//	    mux := http.NewServeMux()
//	    mux.HandleFunc("/documents", MiddlewareWithContext(processor, "document", nil)(documentHandler))
//
//	    http.ListenAndServe(":8080", mux)
//	}
//
//	func documentHandler(w http.ResponseWriter, r *http.Request) {
//	    ctx := r.Context()
//	    results := ResultsFromContext(ctx)
//
//	    // Process with context support
//	    for _, result := range results {
//	        // Check for cancellation
//	        select {
//	        case <-ctx.Done():
//	            return
//	        default:
//	            // Process document
//	            processDocument(ctx, result)
//	        }
//	    }
//
//	    w.Header().Set("Content-Type", "application/json")
//	    json.NewEncoder(w).Encode(map[string]interface{}{
//	        "status": "success",
//	        "documents": results,
//	    })
//	}
//
// The context-aware version is particularly useful when:
// - Using OpenTelemetry for tracing and metrics
// - Handling request timeouts and cancellations
// - Performing async post-processing operations
// - Implementing observability in upload pipelines
// - Building robust upload systems with proper error handling
func MiddlewareWithContext(processor *Processor, fieldName string, errorHandler ErrorHandler) func(http.Handler) http.Handler {
	if errorHandler == nil {
		errorHandler = DefaultErrorHandler
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Process the upload with context
			results, err := processor.ProcessWithContext(r.Context(), r, fieldName)

			if err != nil {
				// Upload failed, call error handler
				errorHandler(w, r, err)
				return
			}

			// Upload succeeded, store results in context
			ctx := context.WithValue(r.Context(), uploadResultsKey, results)
			r = r.WithContext(ctx)

			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}

// JSONUploadErrorHandler returns a JSON error handler that formats upload errors
// in a specific structure for API responses.
//
// This handler returns a 422 Unprocessable Entity status code and formats upload
// errors as a structured JSON response suitable for API clients.
//
// Example response:
//
//	{
//	    "status": "error",
//	    "message": "Upload failed",
//	    "error": "file size exceeds maximum allowed size of 10MB"
//	}
//
// Example usage:
//
//	func main() {
//	    processor := NewProcessor(storage, Options{
//	        MaxFileSize: 10 * 1024 * 1024,
//	        AllowedMIMETypes: []string{"image/jpeg", "image/png"},
//	    })
//
//	    mux := http.NewServeMux()
//
//	    // Use JSON error handler for API endpoints
//	    mux.HandleFunc("/api/upload", Middleware(processor, "file", JSONUploadErrorHandler)(apiUploadHandler))
//
//	    // Use HTML error handler for web forms
//	    mux.HandleFunc("/upload", Middleware(processor, "file", HTMLUploadErrorHandler)(webUploadHandler))
//
//	    http.ListenAndServe(":8080", mux)
//	}
//
//	func apiUploadHandler(w http.ResponseWriter, r *http.Request) {
//	    results := ResultsFromContext(r.Context())
//
//	    // Process successful upload
//	    w.Header().Set("Content-Type", "application/json")
//	    json.NewEncoder(w).Encode(map[string]interface{}{
//	        "status": "success",
//	        "files": results,
//	    })
//	}
//
//	func webUploadHandler(w http.ResponseWriter, r *http.Request) {
//	    results := ResultsFromContext(r.Context())
//
//	    // Render success page
//	    w.Header().Set("Content-Type", "text/html")
//	    w.Write([]byte("<h1>Upload Successful!</h1>"))
//	}
//
// The JSON error handler is ideal for:
// - REST API endpoints
// - AJAX upload requests
// - Mobile app uploads
// - Any client that expects structured error responses
func JSONUploadErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)

	response := map[string]interface{}{
		"status":  "error",
		"message": "Upload failed",
		"error":   err.Error(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// HTMLUploadErrorHandler returns an HTML error handler that renders
// upload errors in HTML format.
//
// This handler returns a 400 Bad Request status code and renders upload errors
// as a user-friendly HTML page. It's suitable for web applications that need to
// display errors to end users.
//
// Example usage:
//
//	func main() {
//	    processor := NewProcessor(storage, Options{
//	        MaxFileSize: 5 * 1024 * 1024,
//	        AllowedMIMETypes: []string{"image/jpeg", "image/png"},
//	    })
//
//	    mux := http.NewServeMux()
//
//	    // Use HTML error handler for web forms
//	    mux.HandleFunc("/upload", Middleware(processor, "file", HTMLUploadErrorHandler)(webUploadHandler))
//
//	    http.ListenAndServe(":8080", mux)
//	}
//
//	func webUploadHandler(w http.ResponseWriter, r *http.Request) {
//	    results := ResultsFromContext(r.Context())
//
//	    // Render success page
//	    w.Header().Set("Content-Type", "text/html")
//	    html := `<!DOCTYPE html>
//	<html>
//	<head>
//	    <title>Upload Success</title>
//	    <style>
//	        body { font-family: Arial, sans-serif; margin: 40px; }
//	        .success { color: #2e7d32; background: #e8f5e8; padding: 10px; border-radius: 4px; }
//	    </style>
//	</head>
//	<body>
//	    <h1>Upload Successful!</h1>
//	    <div class="success">
//	        Uploaded ` + fmt.Sprintf("%d", len(results)) + ` file(s)
//	    </div>
//	    <p><a href="/">Go Home</a></p>
//	</body>
//	</html>`
//	    if _, err := w.Write([]byte(html)); err != nil {
//	        // Optionally log the error
//	    }
//	}
//
// The HTML output includes:
// - Clean, styled error display
// - Clear error message
// - A "Go Back" link for user navigation
// - Responsive design suitable for mobile devices
// - User-friendly error presentation
//
// The HTML error handler is ideal for:
// - Traditional web forms
// - User-facing upload interfaces
// - Applications that need user-friendly error messages
// - Scenarios where users need guidance on how to fix upload issues
func HTMLUploadErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)

	html := `<!DOCTYPE html>
<html>
<head>
    <title>Upload Error</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .error { color: #d32f2f; background: #ffebee; padding: 10px; border-radius: 4px; margin: 10px 0; }
    </style>
</head>
<body>
    <h1>Upload Error</h1>
    <div class="error">
        ` + err.Error() + `
    </div>
    <p><a href="javascript:history.back()">Go Back</a></p>
</body>
</html>`

	if _, err := w.Write([]byte(html)); err != nil {
		// Log the error for debugging purposes if needed
		http.Error(w, "Failed to write HTML response", http.StatusInternalServerError)
		return
	}
}

// SuccessHandler returns middleware that automatically responds with
// upload success information in JSON format.
func SuccessHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a response writer that captures the response
		responseWriter := &responseCapture{ResponseWriter: w}

		// Call the next handler
		next.ServeHTTP(responseWriter, r)

		// If no response was written and we have upload results, write success response
		if !responseWriter.written {
			results := ResultsFromContext(r.Context())
			if results != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				response := map[string]interface{}{
					"status":  "success",
					"message": "Upload completed successfully",
					"files":   results,
				}

				if err := json.NewEncoder(w).Encode(response); err != nil {
					http.Error(w, "Failed to encode response", http.StatusInternalServerError)
					return
				}
			}
		}
	})
}

// responseCapture is a helper to detect if a response was written
type responseCapture struct {
	http.ResponseWriter
	written bool
}

func (rc *responseCapture) Write(data []byte) (int, error) {
	rc.written = true
	return rc.ResponseWriter.Write(data)
}

func (rc *responseCapture) WriteHeader(statusCode int) {
	rc.written = true
	rc.ResponseWriter.WriteHeader(statusCode)
}

// sharedUploadMiddleware handles the common logic for upload middlewares
func sharedUploadMiddleware(process func(*http.Request) (interface{}, error), contextKey contextKey, errorHandler ErrorHandler) func(http.Handler) http.Handler {
	if errorHandler == nil {
		errorHandler = DefaultErrorHandler
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			result, err := process(r)
			if err != nil {
				errorHandler(w, r, err)
				return
			}
			ctx := context.WithValue(r.Context(), contextKey, result)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}
