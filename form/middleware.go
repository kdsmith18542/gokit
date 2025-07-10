package form

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
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

	json.NewEncoder(w).Encode(response)
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
//	    mux.HandleFunc("/register", form.ValidationMiddleware(UserForm{}, nil)(registerHandler))
//
//	    // With custom error handler
//	    mux.HandleFunc("/api/register", form.ValidationMiddleware(UserForm{}, customErrorHandler)(apiRegisterHandler))
//
//	    http.ListenAndServe(":8080", mux)
//	}
//
//	func registerHandler(w http.ResponseWriter, r *http.Request) {
//	    // Get validated form from context
//	    userForm := form.ValidatedFormFromContext(r.Context()).(*UserForm)
//
//	    // Form is already validated and populated
//	    fmt.Printf("Email: %s, Age: %d\n", userForm.Email, userForm.Age)
//
//	    // Process the validated data...
//	}
//
//	func customErrorHandler(w http.ResponseWriter, r *http.Request, errors form.ValidationErrors) {
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
			ctx := context.WithValue(r.Context(), "validated_form", form)
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
//	    form := form.ValidatedFormFromContext(r.Context())
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
	return ctx.Value("validated_form")
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

// ValidationMiddlewareWithContext returns middleware that validates request data
// and provides context-aware validation support.
//
// This middleware is similar to ValidationMiddleware but uses DecodeAndValidateWithContext,
// which provides better observability and context cancellation support. Use this version
// when you need tracing, metrics, or context-aware validation.
//
// Example usage:
//
//	func main() {
//	    mux := http.NewServeMux()
//
//	    // Use context-aware middleware for better observability
//	    mux.HandleFunc("/api/users", form.ValidationMiddlewareWithContext(UserForm{}, nil)(createUserHandler))
//
//	    http.ListenAndServe(":8080", mux)
//	}
//
//	func createUserHandler(w http.ResponseWriter, r *http.Request) {
//	    // The request context is passed to validation for observability
//	    userForm := form.ValidatedFormFromContext(r.Context()).(*UserForm)
//
//	    // Process with context support
//	    ctx := r.Context()
//	    user, err := createUser(ctx, userForm)
//	    if err != nil {
//	        http.Error(w, err.Error(), http.StatusInternalServerError)
//	        return
//	    }
//
//	    json.NewEncoder(w).Encode(user)
//	}
//
// The context-aware version is particularly useful when:
// - Using custom validators that make database calls
// - Implementing observability with OpenTelemetry
// - Handling request timeouts and cancellations
// - Performing async validation operations
func ValidationMiddlewareWithContext(formStruct interface{}, errorHandler ValidationErrorHandler) func(http.Handler) http.Handler {
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
			ctx := context.WithValue(r.Context(), "validated_form", form)
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
//	    mux.HandleFunc("/api/register", form.ValidationMiddleware(UserForm{}, form.JSONValidationErrorHandler)(apiRegisterHandler))
//
//	    http.ListenAndServe(":8080", mux)
//	}
//
//	func apiRegisterHandler(w http.ResponseWriter, r *http.Request) {
//	    userForm := form.ValidatedFormFromContext(r.Context()).(*UserForm)
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

	json.NewEncoder(w).Encode(response)
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
//	    mux.HandleFunc("/contact", form.ValidationMiddleware(ContactForm{}, form.HTMLValidationErrorHandler)(contactHandler))
//
//	    http.ListenAndServe(":8080", mux)
//	}
//
//	func contactHandler(w http.ResponseWriter, r *http.Request) {
//	    contactForm := form.ValidatedFormFromContext(r.Context()).(*ContactForm)
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

	w.Write([]byte(html))
}
