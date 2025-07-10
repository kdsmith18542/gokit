// Package form provides HTTP middleware functions for Go web applications.
// These middleware components can be chained together to add functionality
// such as form processing, internationalization, and file uploads.
package form

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// formDataKey is the context key for form data.
	formDataKey contextKey = "formData"
)

// ValidationErrorHandler is a function type for handling validation errors
type ValidationErrorHandler func(w http.ResponseWriter, r *http.Request, errors ValidationErrors)

// DefaultValidationErrorHandler returns a JSON error response with validation errors
func DefaultValidationErrorHandler(w http.ResponseWriter, r *http.Request, errors ValidationErrors) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	response := map[string]interface{}{
		"error":   "Validation failed",
		"details": errors,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// ValidationMiddleware returns middleware that validates request data against a struct
// and handles validation errors using the provided error handler.
//
// The middleware automatically decodes form data (both regular forms and multipart uploads)
// and validates it against the provided struct. If validation passes, the validated form
// is stored in the request context and can be retrieved using ValidatedFormFromContext.
//
// Example usage:
//
//	type UserForm struct {
//	    Email    string `form:"email" validate:"required,email"`
//	    Password string `form:"password" validate:"required,min=8"`
//	    Age      int    `form:"age" validate:"required,min=18"`
//	}
//
//	func main() {
//	    mux := http.NewServeMux()
//
//	    // Basic usage with default error handler
//	    mux.HandleFunc("/register", ValidationMiddleware(UserForm{}, nil)(registerHandler))
//
//	    // With custom error handler
//	    mux.HandleFunc("/api/register", ValidationMiddleware(UserForm{}, customErrorHandler)(apiRegisterHandler))
//
//	    http.ListenAndServe(":8080", mux)
//	}
//
//	func registerHandler(w http.ResponseWriter, r *http.Request) {
//	    // Get validated form from context
//	    userForm := ValidatedFormFromContext(r.Context()).(*UserForm)
//
//	    // Form is already validated and populated
//	    fmt.Printf("Email: %s, Age: %d\n", userForm.Email, userForm.Age)
//
//	    // Process the validated data...
//	}
//
//	func customErrorHandler(w http.ResponseWriter, r *http.Request, errors ValidationErrors) {
//	    // Custom error handling logic
//	    w.Header().Set("Content-Type", "application/json")
//	    w.WriteHeader(http.StatusBadRequest)
//	    json.NewEncoder(w).Encode(map[string]interface{}{
//	        "success": false,
//	        "errors":  errors,
//	    })
//	}
//
// The middleware supports all validation rules including:
// - Basic rules: required, email, url, min, max, len, numeric, alpha, alphanum
// - Conditional rules: required_if, required_unless, eqfield, nefield, gtfield, ltfield
// - Custom validators registered with RegisterValidator
//
// Sanitization is automatically applied before validation using sanitize tags:
//
//	type SanitizedForm struct {
//	    Username string `form:"username" sanitize:"trim,lower" validate:"required,min=3"`
//	    Bio      string `form:"bio" sanitize:"trim,escape_html" validate:"max=500"`
//	}
func ValidationMiddleware(formStruct interface{}, errorHandler ValidationErrorHandler) func(http.Handler) http.Handler {
	if errorHandler == nil {
		errorHandler = DefaultValidationErrorHandler
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a new instance of the form struct
			form := reflect.New(reflect.TypeOf(formStruct)).Interface()

			// Validate the form
			errors := DecodeAndValidate(r, form)

			if len(errors) > 0 {
				// Validation failed, call error handler
				errorHandler(w, r, errors)
				return
			}

			// Validation succeeded, store form in context
			ctx := context.WithValue(r.Context(), formDataKey, form)
			r = r.WithContext(ctx)

			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}

// ValidatedFormFromContext retrieves the validated form from the request context.
// Returns nil if no form was found in the context.
//
// Example:
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    form := ValidatedFormFromContext(r.Context())
//	    if form == nil {
//	        http.Error(w, "No form data", http.StatusBadRequest)
//	        return
//	    }
//
//	    // Type assert to your specific form type
//	    userForm, ok := form.(*UserForm)
//	    if !ok {
//	        http.Error(w, "Invalid form type", http.StatusInternalServerError)
//	        return
//	    }
//
//	    // Use the validated form data
//	    processUser(userForm)
//	}
func ValidatedFormFromContext(ctx context.Context) interface{} {
	return ctx.Value(formDataKey)
}

// MustValidatedFormFromContext retrieves the validated form from the request context.
// Panics if no form was found in the context.
func MustValidatedFormFromContext(ctx context.Context) interface{} {
	form := ValidatedFormFromContext(ctx)
	if form == nil {
		panic("form: Validated form not found in context. Did you apply the ValidationMiddleware?")
	}
	return form
}

// ValidationMiddlewareWithContext returns middleware that validates request data and provides context-aware validation support.
func ValidationMiddlewareWithContext(formStruct interface{}, errorHandler ValidationErrorHandler) func(next http.Handler) http.Handler {
	if errorHandler == nil {
		errorHandler = DefaultValidationErrorHandler
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a new instance of the form struct
			form := reflect.New(reflect.TypeOf(formStruct)).Interface()

			// Validate the form with context
			errors := DecodeAndValidateWithContext(r.Context(), r, form)

			if len(errors) > 0 {
				// Validation failed, call error handler
				errorHandler(w, r, errors)
				return
			}

			// Validation succeeded, store form in context
			ctx := context.WithValue(r.Context(), formDataKey, form)
			r = r.WithContext(ctx)

			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}

// JSONValidationErrorHandler returns a JSON error handler that formats errors
// in a specific structure for API responses.
//
// This handler returns a 422 Unprocessable Entity status code and formats validation
// errors as a structured JSON response suitable for API clients.
//
// Example response:
//
//	{
//	    "status": "error",
//	    "message": "Validation failed",
//	    "errors": [
//	        {
//	            "field": "email",
//	            "error": "required"
//	        },
//	        {
//	            "field": "email",
//	            "error": "invalid email format"
//	        },
//	        {
//	            "field": "password",
//	            "error": "min length is 8"
//	        }
//	    ]
//	}
//
// Example usage:
//
//	type UserForm struct {
//	    Email    string `form:"email" validate:"required,email"`
//	    Password string `form:"password" validate:"required,min=8"`
//	}
//
//	func main() {
//	    mux := http.NewServeMux()
//
//	    // Use JSON error handler for API endpoints
//	    mux.HandleFunc("/api/register", ValidationMiddleware(UserForm{}, JSONValidationErrorHandler)(apiRegisterHandler))
//
//	    http.ListenAndServe(":8080", mux)
//	}
//
//	func apiRegisterHandler(w http.ResponseWriter, r *http.Request) {
//	    userForm := ValidatedFormFromContext(r.Context()).(*UserForm)
//
//	    // Process registration...
//	    w.Header().Set("Content-Type", "application/json")
//	    json.NewEncoder(w).Encode(map[string]interface{}{
//	        "status": "success",
//	        "message": "User registered successfully",
//	    })
//	}
func JSONValidationErrorHandler(w http.ResponseWriter, r *http.Request, errors ValidationErrors) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)

	// Flatten errors into a single array
	var errorList []map[string]string
	for field, fieldErrors := range errors {
		for _, err := range fieldErrors {
			errorList = append(errorList, map[string]string{
				"field": field,
				"error": err,
			})
		}
	}

	response := map[string]interface{}{
		"status":  "error",
		"message": "Validation failed",
		"errors":  errorList,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// HTMLValidationErrorHandler returns an HTML error handler that renders
// validation errors in HTML format.
//
// This handler returns a 400 Bad Request status code and renders validation errors
// as a user-friendly HTML page. It's suitable for web applications that need to
// display errors to end users.
//
// Example usage:
//
//	type ContactForm struct {
//	    Name    string `form:"name" validate:"required"`
//	    Email   string `form:"email" validate:"required,email"`
//	    Message string `form:"message" validate:"required,min=10"`
//	}
//
//	func main() {
//	    mux := http.NewServeMux()
//
//	    // Use HTML error handler for web forms
//	    mux.HandleFunc("/contact", ValidationMiddleware(ContactForm{}, HTMLValidationErrorHandler)(contactHandler))
//
//	    http.ListenAndServe(":8080", mux)
//	}
//
//	func contactHandler(w http.ResponseWriter, r *http.Request) {
//	    contactForm := ValidatedFormFromContext(r.Context()).(*ContactForm)
//
//	    // Process contact form...
//	    w.Header().Set("Content-Type", "text/html")
//	    w.Write([]byte("<h1>Thank you for your message!</h1>"))
//	}
//
// The HTML output includes:
// - Clean, styled error display
// - Field names and error messages
// - A "Go Back" link for user navigation
// - Responsive design suitable for mobile devices
func HTMLValidationErrorHandler(w http.ResponseWriter, r *http.Request, errors ValidationErrors) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)

	html := `<!DOCTYPE html>
<html>
<head>
    <title>Validation Error</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .error { color: #d32f2f; background: #ffebee; padding: 10px; border-radius: 4px; margin: 10px 0; }
        .field { font-weight: bold; }
    </style>
</head>
<body>
    <h1>Validation Error</h1>
    <p>The following errors occurred:</p>`

	for field, fieldErrors := range errors {
		for _, err := range fieldErrors {
			html += `<div class="error">
                <span class="field">` + field + `:</span> ` + err + `
            </div>`
		}
	}

	html += `
    <p><a href="javascript:history.back()">Go Back</a></p>
</body>
</html>`

	if _, err := w.Write([]byte(html)); err != nil {
		// Optionally log the error
	}
}
