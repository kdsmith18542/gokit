# GoKit: A Unified Web Development Toolkit for Go

GoKit provides a comprehensive, idiomatic, and robust set of libraries that remedy the most common challenges in Go web development. It aims to drastically reduce boilerplate code, eliminate common errors, and streamline the implementation of form validation, internationalization (i18n), and advanced file handling.

[![Go Reference](https://pkg.go.dev/badge/github.com/kdsmith18542/gokit.svg)](https://pkg.go.dev/github.com/kdsmith18542/gokit)
[![Go Report Card](https://goreportcard.com/badge/github.com/kdsmith18542/gokit)](https://goreportcard.com/report/github.com/kdsmith18542/gokit)

## Features

- **Form Validation & Sanitization**: Declarative validation with struct tags, custom validators, and input sanitization
- **Internationalization**: Complete i18n solution with locale detection, message bundles, pluralization, and a web-based editor
- **File Upload Handling**: Advanced file processing with streaming API, resumable uploads, and pluggable storage backends
- **Observability**: Built-in OpenTelemetry integration for tracing, metrics, and logging
- **CLI Tool**: Manage i18n files, validate locales, and handle file uploads

## Feature Matrix

| Feature                        | form      | i18n      | upload    |
|------------------------------- |-----------|-----------|-----------|
| Declarative Validation         |   ✓       |           |           |
| Custom Validators              |   ✓       |           |           |
| Input Sanitization             |   ✓       |           |           |
| Context-aware Validation       |   ✓       |           |           |
| Locale Detection               |           |     ✓     |           |
| Pluralization                  |           |     ✓     |           |
| Locale-aware Formatting        |           |     ✓     |           |
| Web-based Editor               |           |     ✓     |           |
| Streaming Uploads              |           |           |     ✓     |
| Resumable/Chunked Uploads      |           |           |     ✓     |
| S3/GCS/Azure/Local Storage     |           |           |     ✓     |
| Pre-signed URLs                |           |           |     ✓     |
| OpenTelemetry Integration      |   ✓       |     ✓     |     ✓     |
| CLI Management                 |           |     ✓     |     ✓     |

## Installation

```bash
go get github.com/kdsmith18542/gokit
```

## Quick Start

### Form Validation

```go
package main

import (
    "fmt"
    "net/http"
    "github.com/kdsmith18542/gokit/form"
)

type SignUpForm struct {
    Email    string `form:"email" validate:"required,email"`
    Password string `form:"password" validate:"required,min=8"`
    Bio      string `form:"bio" sanitize:"trim,escape_html"`
}

func SignUpHandler(w http.ResponseWriter, r *http.Request) {
    var s SignUpForm
    
    errs := form.DecodeAndValidate(r, &s)
    if errs != nil {
        http.Error(w, fmt.Sprintf("Validation errors: %v", errs), http.StatusBadRequest)
        return
    }

    fmt.Fprintf(w, "Welcome, %s!", s.Email)
}
```

#### Advanced Validation & Sanitization

```go
package main

import (
    "fmt"
    "net/http"
    "github.com/kdsmith18542/gokit/form"
)

// Register custom validators
func init() {
    // Simple custom validator
    form.RegisterValidator("not_foo", func(value string) string {
        if value == "foo" {
            return "Value cannot be 'foo'"
        }
        return ""
    })

    // Context-aware validator (e.g., DB uniqueness check)
    form.RegisterContextValidator("unique_email", func(value, param string, ctx form.ValidationContext) string {
        // Simulate DB check (replace with real DB call)
        if value == "taken@example.com" {
            return "Email is already registered"
        }
        return ""
    })

    // Cross-field validator
    form.RegisterContextValidator("not_equal", func(value, param string, ctx form.ValidationContext) string {
        if value == ctx.Get(param) {
            return "Fields must not match"
        }
        return ""
    })

    // Custom sanitizer
    form.RegisterSanitizer("remove_spaces", func(value string) string {
        return strings.ReplaceAll(value, " ", "")
    })
}

// Advanced form with conditional validation and chained sanitizers
type AdvancedForm struct {
    Email           string `form:"email" validate:"required,email,unique_email" sanitize:"trim,to_lower"`
    Password        string `form:"password" validate:"required,min=8"`
    ConfirmPassword string `form:"confirm_password" validate:"required,eqfield=Password"`
    Username        string `form:"username" validate:"required,min=3,not_foo" sanitize:"trim,remove_spaces"`
    Bio             string `form:"bio" sanitize:"trim,escape_html" validate:"max=500"`
    Age             int    `form:"age" validate:"required,min=18"`
    
    // Conditional validation
    CompanyName     string `form:"company_name" validate:"required_if=AccountType:business"`
    AccountType     string `form:"account_type" validate:"required,oneof=personal,business"`
    
    // Cross-field validation
    StartDate       string `form:"start_date" validate:"required,date_after=EndDate"`
    EndDate         string `form:"end_date" validate:"required"`
    
    // Numeric comparison
    MinAmount       float64 `form:"min_amount" validate:"required,numeric,ltfield=MaxAmount"`
    MaxAmount       float64 `form:"max_amount" validate:"required,numeric"`
}

func AdvancedSignUpHandler(w http.ResponseWriter, r *http.Request) {
    var f AdvancedForm
    
    // Use context-aware validation for observability
    errs := form.DecodeAndValidateWithContext(r.Context(), r, &f)
    if errs != nil {
        // Return structured JSON errors for API
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Validation failed",
            "details": errs,
        })
        return
    }

    // Form is validated and sanitized
    fmt.Fprintf(w, "Welcome, %s!", f.Username)
}
```

#### Built-in Validation Rules

```go
type ValidationExample struct {
    // Basic validation
    Required    string `form:"required" validate:"required"`
    Email       string `form:"email" validate:"required,email"`
    URL         string `form:"url" validate:"url"`
    MinLength   string `form:"min_length" validate:"min=5"`
    MaxLength   string `form:"max_length" validate:"max=100"`
    Numeric     string `form:"numeric" validate:"numeric"`
    Alpha       string `form:"alpha" validate:"alpha"`
    Alphanumeric string `form:"alphanumeric" validate:"alphanumeric"`
    
    // Conditional validation
    Conditional string `form:"conditional" validate:"required_if=OtherField:value"`
    Unless      string `form:"unless" validate:"required_unless=OtherField:value"`
    
    // Cross-field validation
    Password    string `form:"password" validate:"required,min=8"`
    ConfirmPass string `form:"confirm_password" validate:"required,eqfield=Password"`
    StartDate   string `form:"start_date" validate:"required"`
    EndDate     string `form:"end_date" validate:"required,date_after=StartDate"`
    
    // Numeric comparisons
    MinValue    float64 `form:"min_value" validate:"required,numeric,ltfield=MaxValue"`
    MaxValue    float64 `form:"max_value" validate:"required,numeric"`
    
    // Sanitization
    CleanText   string `form:"clean_text" sanitize:"trim,to_lower,escape_html"`
    Username    string `form:"username" sanitize:"trim,remove_spaces" validate:"required,min=3"`
}
```

#### Built-in Sanitizers

```go
// Available sanitizers (can be chained):
// - trim: Remove leading/trailing whitespace
// - to_lower: Convert to lowercase
// - to_upper: Convert to uppercase
// - escape_html: Escape HTML characters
// - strip_numeric: Remove all numeric characters
// - strip_alpha: Remove all alphabetic characters
// - normalize_whitespace: Replace multiple spaces with single space
// - remove_special_chars: Keep only letters, digits, and spaces
// - title_case: Convert to title case (e.g., "hello world" -> "Hello World")
// - camel_case: Convert to camelCase (e.g., "hello world" -> "helloWorld")
// - snake_case: Convert to snake_case (e.g., "hello world" -> "hello_world")
// - kebab_case: Convert to kebab-case (e.g., "hello world" -> "hello-world")
// - remove_html_tags: Remove HTML tags
// - normalize_unicode: Normalize unicode characters

type SanitizationExample struct {
    Name     string `form:"name" sanitize:"trim,to_lower"`
    Bio      string `form:"bio" sanitize:"trim,escape_html"`
    Username string `form:"username" sanitize:"trim,remove_spaces"`
    Slug     string `form:"slug" sanitize:"trim,to_lower,kebab_case"`
    Code     string `form:"code" sanitize:"strip_numeric,to_upper"`
    Title    string `form:"title" sanitize:"trim,title_case"`
    Content  string `form:"content" sanitize:"trim,normalize_whitespace,remove_html_tags"`
}
```

### Internationalization

```go
package main

import (
    "fmt"
    "net/http"
    "github.com/kdsmith18542/gokit/i18n"
)

var i18nManager *i18n.Manager

func init() {
    i18nManager = i18n.NewManager("./locales")
}

func GreetingHandler(w http.ResponseWriter, r *http.Request) {
    translator := i18nManager.Translator(r)
    
    greeting := translator.T("welcomeMessage", map[string]interface{}{
        "User": "Alex",
    })
    
    fmt.Fprintf(w, "%s", greeting)
}
```

### File Upload

#### Basic Upload

```go
package main

import (
    "fmt"
    "net/http"
    "github.com/kdsmith18542/gokit/upload"
    "github.com/kdsmith18542/gokit/upload/storage"
)

var avatarUploader *upload.Processor

func init() {
    localStorage := storage.NewLocal("./uploads")
    
    avatarUploader = upload.NewProcessor(localStorage, upload.Options{
        MaxFileSize: 5 * 1024 * 1024, // 5 MB
        AllowedMIMETypes: []string{"image/jpeg", "image/png"},
    })
}

func UploadAvatarHandler(w http.ResponseWriter, r *http.Request) {
    results, err := avatarUploader.Process(r, "avatar")
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    for _, result := range results {
        fmt.Fprintf(w, "File %s uploaded to %s\n", result.OriginalName, result.URL)
    }
}
```

#### Resumable Uploads

```go
package main

import (
    "fmt"
    "net/http"
    "github.com/kdsmith18542/gokit/upload"
    "github.com/kdsmith18542/gokit/upload/storage"
)

var resumableUploader *upload.ResumableProcessor

func init() {
    localStorage := storage.NewLocal("./uploads")
    
    resumableUploader = upload.NewResumableProcessor(localStorage, upload.ResumableOptions{
        ChunkSize: 1024 * 1024, // 1MB chunks
        MaxFileSize: 100 * 1024 * 1024, // 100MB max
        AllowedMIMETypes: []string{"video/mp4", "application/pdf"},
    })
}

// Start upload
func StartUploadHandler(w http.ResponseWriter, r *http.Request) {
    session, err := resumableUploader.StartSession(r)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "session_id": session.ID,
        "chunk_size": session.ChunkSize,
        "total_chunks": session.TotalChunks,
    })
}

// Upload chunk
func UploadChunkHandler(w http.ResponseWriter, r *http.Request) {
    sessionID := r.URL.Query().Get("session_id")
    chunkIndex := r.URL.Query().Get("chunk")
    
    err := resumableUploader.UploadChunk(r.Context(), sessionID, chunkIndex, r.Body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    w.WriteHeader(http.StatusOK)
}

// Complete upload
func CompleteUploadHandler(w http.ResponseWriter, r *http.Request) {
    sessionID := r.URL.Query().Get("session_id")
    
    result, err := resumableUploader.CompleteSession(r.Context(), sessionID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}
```

#### Storage Backends

```go
package main

import (
    "github.com/kdsmith18542/gokit/upload/storage"
)

// Local storage
localStorage := storage.NewLocal("./uploads")

// S3 storage
s3Storage := storage.NewS3(storage.S3Config{
    Bucket: "my-bucket",
    Region: "us-west-2",
    AccessKey: "your-access-key",
    SecretKey: "your-secret-key",
})

// Google Cloud Storage
gcsStorage := storage.NewGCS(storage.GCSConfig{
    Bucket: "my-bucket",
    ProjectID: "my-project",
    CredentialsFile: "path/to/credentials.json",
})

// Azure Blob Storage
azureStorage := storage.NewAzure(storage.AzureConfig{
    AccountName: "myaccount",
    AccountKey: "your-account-key",
    Container: "my-container",
})

// Use with upload processor
processor := upload.NewProcessor(s3Storage, upload.Options{
    MaxFileSize: 10 * 1024 * 1024,
    AllowedMIMETypes: []string{"image/*", "application/pdf"},
})
```

### Observability

GoKit includes built-in OpenTelemetry integration for comprehensive observability:

```go
package main

import (
    "github.com/kdsmith18542/gokit/observability"
    "github.com/kdsmith18542/gokit/form"
    "github.com/kdsmith18542/gokit/i18n"
    "github.com/kdsmith18542/gokit/upload"
)

func init() {
    // Initialize observability
    observability.Init(observability.Config{
        ServiceName: "my-app",
        ServiceVersion: "1.0.0",
        Environment: "production",
    })
}

func MyHandler(w http.ResponseWriter, r *http.Request) {
    // All operations are automatically traced and metered
    var form MyForm
    errs := form.DecodeAndValidateWithContext(r.Context(), r, &form)
    
    translator := i18nManager.Translator(r)
    message := translator.T("welcome", nil)
    
    results, _ := uploader.Process(r, "file")
    
    // Custom spans and metrics
    observability.RecordMetric("custom_operation", 1)
}
```

## CLI Tool

GoKit includes a powerful CLI tool for managing i18n files and handling uploads:

### Build the CLI

```bash
go build -o gokit-cli ./cmd/gokit-cli
```

### i18n Commands

```bash
# Validate all locale files
./gokit-cli i18n validate --dir ./locales

# Extract translation keys from code
./gokit-cli i18n extract --dir ./src --output ./locales/en.toml

# Merge translation files
./gokit-cli i18n merge --source ./locales/en.toml --target ./locales/es.toml

# Start web-based editor
./gokit-cli i18n editor --dir ./locales --port 8080
```

### Upload Commands

```bash
# Upload file to storage
./gokit-cli upload file --file ./document.pdf --storage s3 --bucket my-bucket

# Generate pre-signed URL
./gokit-cli upload presign --file document.pdf --storage s3 --bucket my-bucket --expiry 1h

# List files in storage
./gokit-cli upload list --storage s3 --bucket my-bucket
```

## HTTP Middleware (Optional)

GoKit provides optional, idiomatic HTTP middleware for i18n, form validation, and file upload. Middleware is not required—it's a convenience for context-based handler design. You can use the lower-level APIs directly if you prefer.

### i18n Middleware

```go
mux := http.NewServeMux()
mux.HandleFunc("/", GreetingHandler)
handler := i18n.LocaleDetector(i18nManager)(mux)
http.ListenAndServe(":8080", handler)

func GreetingHandler(w http.ResponseWriter, r *http.Request) {
    translator := i18n.TranslatorFromContext(r.Context())
    greeting := translator.T("welcomeMessage", nil)
    fmt.Fprint(w, greeting)
}
```

### Form Validation Middleware

```go
type UserForm struct {
    Email    string `form:"email" validate:"required,email"`
    Password string `form:"password" validate:"required,min=8"`
}

mux.Handle("/register", form.ValidationMiddleware(UserForm{}, nil)(registerHandler))

func registerHandler(w http.ResponseWriter, r *http.Request) {
    form := form.ValidatedFormFromContext(r.Context()).(*UserForm)
    // Use validated form
}
```

### Upload Middleware

```go
mux.Handle("/upload", upload.UploadMiddleware(processor, "file", nil)(uploadHandler))

func uploadHandler(w http.ResponseWriter, r *http.Request) {
    results := upload.UploadResultsFromContext(r.Context())
    // Use upload results
}
```

**Note:** All middleware is optional. You can always use the lower-level APIs directly for full control.

## Advanced Examples

See the [`examples/`](./examples) directory for:
- **CLI Tool:** Manage and validate i18n files
- **Resumable Upload:** Chunked uploads with progress and resume support
- **Web-based i18n Editor:** Edit translations in your browser
- **Advanced Features:** Observability, custom validators, and storage backends
- **Integration Tests:** Comprehensive test coverage

## API Reference

- [pkg.go.dev/github.com/kdsmith18542/gokit/form](https://pkg.go.dev/github.com/kdsmith18542/gokit/form)
- [pkg.go.dev/github.com/kdsmith18542/gokit/i18n](https://pkg.go.dev/github.com/kdsmith18542/gokit/i18n)
- [pkg.go.dev/github.com/kdsmith18542/gokit/upload](https://pkg.go.dev/github.com/kdsmith18542/gokit/upload)
- [pkg.go.dev/github.com/kdsmith18542/gokit/observability](https://pkg.go.dev/github.com/kdsmith18542/gokit/observability)

## Project Structure

```
gokit/
├── form/           # Form validation and sanitization
├── i18n/           # Internationalization and localization
├── upload/         # Advanced file upload handling
├── upload/storage/ # Storage backend implementations
├── observability/  # OpenTelemetry integration
├── examples/       # Usage examples
├── cmd/gokit-cli/  # CLI tool with i18n and upload subcommands
└── docs/           # Detailed documentation
```

## Security Notes

- **i18n Editor:** Do NOT expose the web-based i18n editor in production without authentication/authorization.
- **File Uploads:** Always validate file size, type, and content. Use virus scanning and restrict upload paths in production.
- **CLI Tool:** Intended for development/admin use only.
- **Storage Credentials:** Use environment variables or secure credential management for production deployments.

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests
go test -tags=integration ./...
```

### Building

```bash
# Build all packages
go build ./...

# Build CLI tool
go build -o gokit-cli ./cmd/gokit-cli

# Build examples
go build ./examples/...
```

### CI/CD

The project includes comprehensive CI/CD with:
- Automated testing and linting
- Security scanning
- Multi-platform builds
- Coverage reporting
- Automated releases

## Documentation

- **[Security Best Practices](./docs/SECURITY_BEST_PRACTICES.md)** - Production security guidelines
- **[Troubleshooting Guide](./docs/TROUBLESHOOTING.md)** - Common issues and solutions
- **[API Reference](https://pkg.go.dev/github.com/kdsmith18542/gokit)** - Complete API documentation
- **[Examples](./examples/)** - Working code examples

## Performance Tips

### Form Validation
- Order validation rules by performance (quick checks first)
- Use caching for expensive context validators
- Pre-compile custom regex patterns

### File Uploads
- Use resumable uploads for files > 10MB
- Set appropriate chunk sizes (1-5MB typically optimal)
- Pre-allocate result slices when size is known
- Limit concurrent uploads to control memory usage

### i18n
- Load locale files once at startup
- Cache translator instances when possible
- Use fallback locales efficiently

### General
- Use context-aware methods for observability
- Enable connection pooling for cloud storage
- Monitor with OpenTelemetry metrics

## License

MIT License - see LICENSE file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

See [CONTRIBUTING.md](./CONTRIBUTING.md) for detailed guidelines.

## Roadmap

- **v1.0**: Stable release with comprehensive documentation and examples 