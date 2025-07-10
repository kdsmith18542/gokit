# Form Validation & Sanitization

The `form` package provides a flexible and declarative way to validate and sanitize incoming data from various sources. It's designed to reduce boilerplate code while providing robust validation capabilities.

## Table of Contents

- [Quick Start](#quick-start)
- [Core Concepts](#core-concepts)
- [Validation Rules](#validation-rules)
- [Conditional Validation](#conditional-validation)
- [Sanitization](#sanitization)
- [Custom Validators](#custom-validators)
- [Error Handling](#error-handling)
- [Middleware Integration](#middleware-integration)
- [Advanced Examples](#advanced-examples)

## Quick Start

```go
package main

import (
    "net/http"
    "github.com/kdsmith18542/gokit/form"
)

type UserForm struct {
    Email    string `form:"email" validate:"required,email"`
    Password string `form:"password" validate:"required,min=8"`
    Age      int    `form:"age" validate:"required,min=18,max=120"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var user UserForm
    
    errs := form.DecodeAndValidate(r.Context(), r, &user)
    if errs != nil {
        // Handle validation errors
        return
    }
    
    // User data is validated and ready to use
}
```

## Core Concepts

### Validator Instance

The package centers around a `form.Validator` instance that handles decoding and validation:

```go
validator := form.NewValidator()
errs := validator.DecodeAndValidate(ctx, source, &dest)
```

### Data Sources

The validator can handle multiple input sources:

- **HTTP Request**: `*http.Request` (form data, query params)
- **JSON Reader**: `io.Reader` containing JSON data
- **Raw Data**: `map[string][]string` for programmatic validation

```go
// From HTTP request
errs := validator.DecodeAndValidate(ctx, r, &user)

// From JSON
errs := validator.DecodeAndValidateJSON(ctx, jsonReader, &user)

// From raw data
data := map[string][]string{"email": {"user@example.com"}}
errs := validator.DecodeAndValidateMap(ctx, data, &user)
```

## Validation Rules

### Basic Rules

| Rule | Description | Example |
|------|-------------|---------|
| `required` | Field must be present and non-empty | `validate:"required"` |
| `email` | Must be valid email format | `validate:"email"` |
| `url` | Must be valid URL format | `validate:"url"` |
| `min` | Minimum length/value | `validate:"min=5"` |
| `max` | Maximum length/value | `validate:"max=100"` |
| `len` | Exact length/value | `validate:"len=10"` |
| `numeric` | Must be numeric | `validate:"numeric"` |
| `alpha` | Alphabetic characters only | `validate:"alpha"` |
| `alphanum` | Alphanumeric characters only | `validate:"alphanum"` |

### String Validation

```go
type StringValidation struct {
    Username string `validate:"required,min=3,max=20,alphanum"`
    Email    string `validate:"required,email"`
    Website  string `validate:"url"`
    Phone    string `validate:"len=10,numeric"`
}
```

### Numeric Validation

```go
type NumericValidation struct {
    Age     int     `validate:"required,min=18,max=120"`
    Price   float64 `validate:"required,min=0.01"`
    Rating  int     `validate:"min=1,max=5"`
    Percent float64 `validate:"min=0,max=100"`
}
```

## Conditional Validation

The form package supports advanced conditional validation rules:

### Field Comparisons

| Rule | Description | Example |
|------|-------------|---------|
| `eqfield` | Equal to another field | `validate:"eqfield=Password"` |
| `nefield` | Not equal to another field | `validate:"nefield=OldPassword"` |
| `gtfield` | Greater than another field | `validate:"gtfield=MinAge"` |
| `gtefield` | Greater than or equal to another field | `validate:"gtefield=MinAge"` |
| `ltfield` | Less than another field | `validate:"ltfield=MaxAge"` |
| `ltefield` | Less than or equal to another field | `validate:"ltefield=MaxAge"` |

### Conditional Requirements

| Rule | Description | Example |
|------|-------------|---------|
| `required_if` | Required if another field equals a value | `validate:"required_if=Type:premium"` |
| `required_unless` | Required unless another field equals a value | `validate:"required_unless=Type:guest"` |

### Advanced Conditional Example

```go
type AdvancedForm struct {
    UserType     string `form:"user_type" validate:"required,oneof=free premium enterprise"`
    CompanyName  string `form:"company_name" validate:"required_if=UserType:premium enterprise"`
    CreditCard   string `form:"credit_card" validate:"required_if=UserType:premium enterprise"`
    PhoneNumber  string `form:"phone" validate:"required_unless=UserType:guest"`
    
    StartDate    string `form:"start_date" validate:"required,date"`
    EndDate      string `form:"end_date" validate:"required,date,gtefield=StartDate"`
    
    MinAge       int    `form:"min_age" validate:"required,min=0,max=120"`
    MaxAge       int    `form:"max_age" validate:"required,min=0,max=120,gtefield=MinAge"`
}
```

## Sanitization

Sanitization rules clean and transform input data:

| Rule | Description | Example |
|------|-------------|---------|
| `trim` | Remove leading/trailing whitespace | `sanitize:"trim"` |
| `lower` | Convert to lowercase | `sanitize:"lower"` |
| `upper` | Convert to uppercase | `sanitize:"upper"` |
| `escape_html` | Escape HTML characters | `sanitize:"escape_html"` |
| `strip_tags` | Remove HTML tags | `sanitize:"strip_tags"` |

### Sanitization Example

```go
type SanitizedForm struct {
    Username string `form:"username" sanitize:"trim,lower" validate:"required,min=3"`
    Bio      string `form:"bio" sanitize:"trim,escape_html" validate:"max=500"`
    Email    string `form:"email" sanitize:"trim,lower" validate:"required,email"`
}
```

## Custom Validators

Register custom validation functions for complex business logic:

```go
// Custom validator function
func validateUniqueEmail(ctx context.Context, value string) error {
    // Check database for existing email
    exists, err := db.EmailExists(ctx, value)
    if err != nil {
        return err
    }
    if exists {
        return errors.New("email already exists")
    }
    return nil
}

// Register the validator
validator := form.NewValidator()
validator.RegisterValidator("unique_email", validateUniqueEmail)

// Use in struct
type UserRegistration struct {
    Email string `validate:"required,email,unique_email"`
}
```

### Async Validators

For database or API calls, use async validators:

```go
func validateUniqueUsername(ctx context.Context, value string) error {
    // This could be a database call
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        // Perform validation
        return nil
    }
}
```

## Error Handling

Validation errors are returned as a structured map:

```go
errs := validator.DecodeAndValidate(ctx, r, &user)
if errs != nil {
    // errs is map[string][]string
    // Example: {"email": ["required", "invalid email format"]}
    
    // Convert to JSON for API response
    json.NewEncoder(w).Encode(map[string]interface{}{
        "errors": errs,
        "message": "Validation failed",
    })
    return
}
```

### Error Structure

```go
type ValidationErrors map[string][]string

// Example error map:
{
    "email": ["required", "invalid email format"],
    "password": ["min length is 8"],
    "age": ["must be at least 18"]
}
```

## Middleware Integration

Use the form middleware for automatic validation in HTTP handlers:

```go
// Define your form struct
type LoginForm struct {
    Email    string `form:"email" validate:"required,email"`
    Password string `form:"password" validate:"required,min=6"`
}

// Create middleware
formMiddleware := form.Middleware(LoginForm{})

// Apply to routes
http.HandleFunc("/login", formMiddleware(loginHandler))

// In your handler, get validated data
func loginHandler(w http.ResponseWriter, r *http.Request) {
    formData := form.FromContext(r.Context())
    loginForm := formData.(*LoginForm)
    
    // Use validated form data
}
```

### Middleware Options

```go
// With custom error handler
formMiddleware := form.Middleware(LoginForm{}, form.WithErrorHandler(func(w http.ResponseWriter, errs map[string][]string) {
    // Custom error response
}))

// With custom validator
validator := form.NewValidator()
validator.RegisterValidator("custom_rule", customValidator)
formMiddleware := form.Middleware(LoginForm{}, form.WithValidator(validator))
```

## Advanced Examples

### Complex Registration Form

```go
type RegistrationForm struct {
    // Basic info
    Username        string `form:"username" sanitize:"trim,lower" validate:"required,min=3,max=20,alphanum"`
    Email           string `form:"email" sanitize:"trim,lower" validate:"required,email"`
    Password        string `form:"password" validate:"required,min=8"`
    ConfirmPassword string `form:"confirm_password" validate:"required,eqfield=Password"`
    
    // Profile info
    FirstName string `form:"first_name" sanitize:"trim" validate:"required,alpha"`
    LastName  string `form:"last_name" sanitize:"trim" validate:"required,alpha"`
    Age       int    `form:"age" validate:"required,min=13,max=120"`
    
    // Conditional fields
    AccountType string `form:"account_type" validate:"required,oneof=personal business"`
    CompanyName string `form:"company_name" validate:"required_if=AccountType:business"`
    Website     string `form:"website" validate:"required_if=AccountType:business,url"`
    
    // Date validation
    BirthDate string `form:"birth_date" validate:"required,date"`
    
    // Terms acceptance
    AcceptTerms bool `form:"accept_terms" validate:"required"`
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
    var form RegistrationForm
    
    errs := form.DecodeAndValidate(r.Context(), r, &form)
    if errs != nil {
        respondWithErrors(w, errs)
        return
    }
    
    // Process registration
    user := createUser(form)
    respondWithSuccess(w, user)
}
```

### API Response Handler

```go
type APIResponse struct {
    Success bool                   `json:"success"`
    Data    interface{}            `json:"data,omitempty"`
    Errors  map[string][]string    `json:"errors,omitempty"`
    Message string                 `json:"message,omitempty"`
}

func respondWithErrors(w http.ResponseWriter, errs map[string][]string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusBadRequest)
    
    response := APIResponse{
        Success: false,
        Errors:  errs,
        Message: "Validation failed",
    }
    
    json.NewEncoder(w).Encode(response)
}

func respondWithSuccess(w http.ResponseWriter, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    
    response := APIResponse{
        Success: true,
        Data:    data,
        Message: "Success",
    }
    
    json.NewEncoder(w).Encode(response)
}
```

### Testing Form Validation

```go
func TestRegistrationForm(t *testing.T) {
    validator := form.NewValidator()
    
    tests := []struct {
        name    string
        data    map[string][]string
        wantErr bool
    }{
        {
            name: "valid form",
            data: map[string][]string{
                "username": {"john_doe"},
                "email":    {"john@example.com"},
                "password": {"securepass123"},
                "confirm_password": {"securepass123"},
                "first_name": {"John"},
                "last_name": {"Doe"},
                "age": {"25"},
                "account_type": {"personal"},
                "birth_date": {"1998-01-01"},
                "accept_terms": {"true"},
            },
            wantErr: false,
        },
        {
            name: "invalid email",
            data: map[string][]string{
                "email": {"invalid-email"},
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var form RegistrationForm
            errs := validator.DecodeAndValidateMap(context.Background(), tt.data, &form)
            
            if tt.wantErr && len(errs) == 0 {
                t.Error("expected errors, got none")
            }
            if !tt.wantErr && len(errs) > 0 {
                t.Errorf("unexpected errors: %v", errs)
            }
        })
    }
}
```

## Best Practices

1. **Use Struct Tags**: Define validation rules in struct tags for clarity and maintainability
2. **Sanitize Input**: Always sanitize user input before validation
3. **Context Support**: Use context for async validators to handle timeouts and cancellations
4. **Error Handling**: Provide clear, user-friendly error messages
5. **Testing**: Write comprehensive tests for your validation logic
6. **Middleware**: Use middleware for common validation patterns
7. **Custom Validators**: Create reusable custom validators for business logic

## Performance Considerations

- The validator caches reflection information for better performance
- Use context timeouts for async validators
- Consider using sync.Pool for frequently used validator instances
- Sanitization is applied before validation to reduce unnecessary validation calls 