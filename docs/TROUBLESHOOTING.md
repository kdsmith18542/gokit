# Troubleshooting Guide

This guide helps you diagnose and resolve common issues when using GoKit.

## Table of Contents

- [Form Validation Issues](#form-validation-issues)
- [File Upload Problems](#file-upload-problems)
- [Internationalization (i18n) Issues](#internationalization-i18n-issues)
- [Storage Backend Issues](#storage-backend-issues)
- [Performance Issues](#performance-issues)
- [Build and Dependency Issues](#build-and-dependency-issues)

## Form Validation Issues

### Validation Not Working

**Symptom**: Form validation always passes even with invalid data.

**Possible Causes**:
1. Struct tags are missing or incorrect
2. Form field names don't match struct tags

**Solution**:
```go
// Ensure struct tags are properly formatted
type UserForm struct {
    // Correct - note the space after colon
    Email string `form:"email" validate:"required,email"`
    
    // Incorrect - no space after colon may cause issues
    // Email string `form:"email"validate:"required,email"`
}

// Ensure form field names match
// HTML: <input name="email"> matches form:"email"
```

### Custom Validators Not Called

**Symptom**: Custom validators are registered but never executed.

**Possible Causes**:
1. Validator registered after form processing
2. Validator name doesn't match tag

**Solution**:
```go
// Register validators in init() or before processing forms
func init() {
    form.RegisterValidator("custom_rule", func(value string) string {
        // validation logic
        return ""
    })
}

// Use exact name in struct tag
type Form struct {
    Field string `form:"field" validate:"custom_rule"` // matches registered name
}
```

### Sanitizers Not Applied

**Symptom**: Input is not sanitized even with sanitize tags.

**Solution**:
```go
// Sanitizers are applied BEFORE validation
// Check the order: sanitize tag comes before validate tag
type Form struct {
    Email string `form:"email" sanitize:"trim,to_lower" validate:"required,email"`
}

// Verify sanitizer is registered
form.RegisterSanitizer("custom_sanitize", func(value string) string {
    return strings.TrimSpace(value)
})
```

## File Upload Problems

### Files Not Uploading

**Symptom**: Upload fails with "no files found" error.

**Possible Causes**:
1. Form enctype is not set to multipart/form-data
2. Field name doesn't match processor call
3. File size exceeds limits

**Solution**:
```html
<!-- HTML form must have correct enctype -->
<form method="POST" enctype="multipart/form-data">
    <input type="file" name="avatar">
    <button type="submit">Upload</button>
</form>
```

```go
// Field name must match HTML input name
results, err := processor.Process(r, "avatar") // matches name="avatar"
```

### "File Too Large" Error

**Symptom**: Large files fail with size error.

**Solution**:
```go
// Increase MaxFileSize in processor options
processor := upload.NewProcessor(storage, upload.Options{
    MaxFileSize: 50 * 1024 * 1024, // 50MB instead of default
})

// Also check server limits
// http.ParseMultipartForm has a 32MB default limit
// Increase if needed in custom handler:
if err := r.ParseMultipartForm(100 * 1024 * 1024); err != nil { // exactly 100MB
    // handle error
}
```

### "File Type Not Allowed" Error

**Symptom**: Valid files are rejected.

**Possible Causes**:
1. MIME type mismatch
2. Browser sends different MIME type

**Solution**:
```go
// Use wildcards for flexibility
processor := upload.NewProcessor(storage, upload.Options{
    AllowedMIMETypes: []string{
        "image/*",           // All image types
        "application/pdf",
        "text/plain",
    },
})

// Or list all specific types you support
AllowedMIMETypes: []string{
    "image/jpeg",
    "image/jpg",  // Some browsers send jpg instead of jpeg
    "image/png",
}
```

### Resumable Upload Session Not Found

**Symptom**: Chunk upload fails with "upload session not found".

**Possible Causes**:
1. Session expired (default 24 hours)
2. Server restarted (sessions are in-memory)
3. Wrong session ID

**Solution**:
```go
// Increase session TTL
rp := upload.NewResumableProcessor(storage, options)
rp.SetSessionTTL(48 * time.Hour) // Extend to 48 hours

// For production, implement persistent session storage
// Store sessions in Redis, database, etc.
```

## Internationalization (i18n) Issues

### Translations Not Loading

**Symptom**: Translations show keys instead of translated text.

**Possible Causes**:
1. Locale files not found
2. Incorrect file format
3. Wrong locale code

**Solution**:
```go
// Verify locale directory exists and contains .toml files
manager := i18n.NewManager("./locales")
// Directory structure:
// locales/
//   en.toml
//   es.toml
//   fr.toml

// Check file format - must be valid TOML
// en.toml:
welcome = "Welcome, {{.User}}!"
items = "{{.Count}} items"
```

### Wrong Locale Detected

**Symptom**: Wrong language is displayed.

**Solution**:
```go
// Check Accept-Language header priority
// Browser sends: "en-US,en;q=0.9,es;q=0.8"
// Manager will try: en-US, en, default

// Set explicit default locale
manager := i18n.NewManager("./locales")
manager.SetDefaultLocale("en")
manager.SetFallbackLocale("en")

// Or use cookie/query param for explicit locale selection
translator := manager.TranslatorWithLocale("es")
```

### Template Variables Not Replaced

**Symptom**: `{{.Variable}}` appears in output instead of value.

**Solution**:
```go
// Ensure you pass the data map
greeting := translator.T("welcome", map[string]interface{}{
    "User": "Alice",  // Key must match template
})

// Locale file must use correct syntax
// welcome = "Welcome, {{.User}}!"  // Correct
// welcome = "Welcome, {User}!"     // Wrong - won't work
```

## Storage Backend Issues

### Local Storage: Permission Denied

**Symptom**: File storage fails with permission error.

**Solution**:
```bash
# Ensure upload directory exists and is writable
mkdir -p ./uploads
chmod 750 ./uploads

# Check process user has write permissions
ls -la uploads/
```

### S3: Access Denied

**Symptom**: S3 uploads fail with "Access Denied".

**Solution**:
```go
// Verify credentials
s3Storage := storage.NewS3(storage.S3Config{
    Bucket:    "my-bucket",
    Region:    "us-west-2",
    AccessKey: os.Getenv("AWS_ACCESS_KEY_ID"),     // Check these
    SecretKey: os.Getenv("AWS_SECRET_ACCESS_KEY"), // are correct
})

// Ensure IAM policy allows s3:PutObject
// Example policy:
// {
//   "Effect": "Allow",
//   "Action": ["s3:PutObject", "s3:GetObject", "s3:DeleteObject"],
//   "Resource": "arn:aws:s3:::my-bucket/*"
// }
```

### GCS: Authentication Failed

**Symptom**: Google Cloud Storage operations fail.

**Solution**:
```go
// Verify service account credentials
gcsStorage := storage.NewGCS(storage.GCSConfig{
    Bucket:          "my-bucket",
    ProjectID:       "my-project",
    CredentialsFile: "/path/to/credentials.json", // Check path
})

// Ensure service account has Storage Object Admin role
// Or set GOOGLE_APPLICATION_CREDENTIALS environment variable
```

## Performance Issues

### Slow Form Validation

**Symptom**: Form processing takes too long.

**Possible Causes**:
1. Too many validation rules
2. Complex regex patterns
3. Context validators doing expensive operations

**Solution**:
```go
// Optimize validation order - fail fast
type Form struct {
    // Put quick validations first
    Email string `validate:"required,email"` // Fast
    
    // Expensive validations last
    Username string `validate:"required,unique_username"` // DB lookup
}

// Use caching for context validators
var usernameCache = make(map[string]bool)

form.RegisterContextValidator("unique_username", func(value, param string, ctx form.ValidationContext) string {
    if cached, ok := usernameCache[value]; ok {
        if !cached {
            return "Username already taken"
        }
        return ""
    }
    // Do actual DB check
    exists := checkUsernameInDB(value)
    usernameCache[value] = !exists
    if exists {
        return "Username already taken"
    }
    return ""
})
```

### Slow File Uploads

**Symptom**: Large files take too long to upload.

**Solution**:
```go
// Use resumable uploads for large files
rp := upload.NewResumableProcessor(storage, upload.Options{
    MaxFileSize: 100 * 1024 * 1024,
})

// Set appropriate chunk size
session, err := rp.InitiateUpload(ctx, 
    fileName, 
    totalSize, 
    mimeType, 
    1024 * 1024, // 1MB chunks
)

// Enable compression at HTTP server level
// Use CDN for file delivery
```

### Memory Usage Too High

**Symptom**: Application uses excessive memory.

**Possible Causes**:
1. Buffering entire files in memory
2. Too many concurrent uploads
3. Session leaks

**Solution**:
```go
// GoKit already uses streaming, but ensure you:

// 1. Limit concurrent uploads
var uploadSemaphore = make(chan struct{}, 10) // Max 10 concurrent

func uploadHandler(w http.ResponseWriter, r *http.Request) {
    uploadSemaphore <- struct{}{}
    defer func() { <-uploadSemaphore }()
    
    results, err := processor.Process(r, "files")
    // ...
}

// 2. Clean up expired sessions
rp.CleanupExpiredSessions() // Call periodically

// 3. Monitor with pprof
import _ "net/http/pprof"
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
// Visit http://localhost:6060/debug/pprof/
```

## Build and Dependency Issues

### Build Fails with Missing Dependencies

**Symptom**: `go build` fails with import errors.

**Solution**:
```bash
# Update dependencies
go get github.com/kdsmith18542/gokit@latest
go mod tidy

# Clear module cache if needed
go clean -modcache
go mod download
```

### Version Conflicts

**Symptom**: Dependency version conflicts.

**Solution**:
```bash
# Check current versions
go list -m all | grep gokit

# Update to specific version
go get github.com/kdsmith18542/gokit@v1.0.0

# Update go.mod
go mod tidy
```

### Tests Fail After Update

**Symptom**: Tests fail after updating GoKit.

**Solution**:
```bash
# Check changelog for breaking changes
# Update test imports if package structure changed

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -v -run TestSpecificTest ./...
```

## Getting Help

If you're still experiencing issues:

1. **Check the examples**: See [examples/](../examples/) directory
2. **Review tests**: Test files show expected usage patterns
3. **Enable observability**: Use the observability package to trace issues
4. **Open an issue**: [GitHub Issues](https://github.com/kdsmith18542/gokit/issues)
5. **Check logs**: Enable verbose logging to see what's happening

### Enabling Debug Logging

```go
// Enable observability for debugging
import "github.com/kdsmith18542/gokit/observability"

observability.Init(observability.Config{
    ServiceName:   "my-app",
    EnableTracing: true,
    EnableMetrics: true,
    EnableLogging: true,
})

// Wrap handlers with observability
ctx, span := observability.StartSpan(r.Context(), "upload_handler")
defer span.End()
```

### Useful Debugging Commands

```bash
# Check Go version
go version

# List all dependencies
go list -m all

# Show package info
go list -json github.com/kdsmith18542/gokit

# Run tests with race detection
go test -race ./...

# Profile CPU usage
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof
```
