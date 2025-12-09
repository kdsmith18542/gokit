# Security Best Practices for GoKit

This document provides security best practices for using GoKit in production applications.

## General Security Principles

### 1. Input Validation and Sanitization

Always validate and sanitize user input:

```go
type UserForm struct {
    Email    string `form:"email" validate:"required,email" sanitize:"trim,to_lower"`
    Username string `form:"username" validate:"required,min=3,max=20" sanitize:"trim"`
    Bio      string `form:"bio" validate:"max=500" sanitize:"trim,escape_html"`
}
```

### 2. File Upload Security

#### Validate File Types
```go
processor := upload.NewProcessor(storage, upload.Options{
    MaxFileSize: 5 * 1024 * 1024, // 5MB
    // Use explicit MIME types, avoid wildcards in production
    AllowedMIMETypes: []string{"image/jpeg", "image/png", "image/gif"},
    AllowedExtensions: []string{".jpg", ".jpeg", ".png", ".gif"},
})
```

#### Additional File Upload Recommendations

1. **Content Validation**: Verify file content matches its MIME type
2. **Virus Scanning**: Integrate with antivirus software for production
3. **Storage Isolation**: Store uploaded files outside the web root
4. **Access Control**: Implement authentication/authorization for file access
5. **Rate Limiting**: Limit upload frequency per user/IP

```go
// Example: Add custom validation hook
processor.OnSuccess(func(ctx context.Context, result upload.Result) {
    // Verify image dimensions, scan for malware, etc.
    if !isValidImage(result.Path) {
        storage.Delete(result.Path)
        log.Printf("Invalid image uploaded: %s", result.OriginalName)
    }
})
```

### 3. Error Handling

**DO NOT** expose internal errors to end users:

```go
// BAD - Exposes internal paths
if err != nil {
    return fmt.Errorf("failed to open file at /var/app/uploads/secret.txt: %w", err)
}

// GOOD - Generic error message for users
if err != nil {
    log.Printf("File operation failed: %v", err) // Log detailed error
    return errors.New("file operation failed") // Return generic error
}
```

### 4. Path Traversal Prevention

GoKit's local storage already includes path traversal protection, but be aware:

```go
// LocalStorage validates:
// - No ".." in filenames
// - No path separators (/ or \)
// - No null bytes
// - No control characters
// - Refuses to write to symlinks
```

**Additional recommendations:**
- Use absolute paths for storage directories
- Set appropriate file permissions (0600 for files, 0750 for directories)
- Regularly audit uploaded files

### 5. Authentication and Authorization

For the i18n editor and file management endpoints:

```go
// DO NOT expose the i18n editor in production without authentication
func protectedEditorHandler(w http.ResponseWriter, r *http.Request) {
    // Check authentication
    if !isAuthenticated(r) {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Check authorization - only admins can edit translations
    if !hasRole(r, "admin") {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }
    
    // Serve the editor
    i18n.EditorHandler(w, r)
}
```

### 6. Secure Configuration

#### Environment Variables
Store sensitive configuration in environment variables:

```go
// BAD - Hardcoded credentials
s3Storage := storage.NewS3(storage.S3Config{
    AccessKey: "AKIAIOSFODNN7EXAMPLE",
    SecretKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
})

// GOOD - Use environment variables
s3Storage := storage.NewS3(storage.S3Config{
    AccessKey: os.Getenv("AWS_ACCESS_KEY_ID"),
    SecretKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
})
```

#### File Permissions
```bash
# Restrict access to configuration files
chmod 600 config.toml

# Restrict access to upload directories
chmod 750 uploads/
```

### 7. HTTPS Only

Always use HTTPS in production:

```go
// Configure TLS
server := &http.Server{
    Addr:         ":443",
    Handler:      router,
    TLSConfig:    &tls.Config{
        MinVersion: tls.VersionTLS12,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        },
    },
}

log.Fatal(server.ListenAndServeTLS("cert.pem", "key.pem"))
```

### 8. Rate Limiting

Implement rate limiting for form submissions and file uploads:

```go
import "golang.org/x/time/rate"

var limiter = rate.NewLimiter(rate.Limit(10), 20) // 10 requests/sec, burst of 20

func rateLimitMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !limiter.Allow() {
            http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### 9. Content Security Policy

Set appropriate security headers:

```go
func securityHeadersMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        next.ServeHTTP(w, r)
    })
}
```

### 10. Logging and Monitoring

Log security-relevant events:

```go
// Log authentication failures
log.Printf("Authentication failed for user: %s from IP: %s", username, r.RemoteAddr)

// Log file upload events
processor.OnSuccess(func(ctx context.Context, result upload.Result) {
    log.Printf("File uploaded: %s (size: %d, type: %s) by user: %s", 
        result.OriginalName, result.Size, result.MIMEType, getUserID(ctx))
})

// Log validation failures
if len(errs) > 0 {
    log.Printf("Validation failed for form: %s, errors: %v", formName, errs)
}
```

## Dependency Security

### Keep Dependencies Updated

Regularly update GoKit and its dependencies:

```bash
go get -u github.com/kdsmith18542/gokit
go mod tidy
```

### Audit Dependencies

Use security scanning tools:

```bash
# Install govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest

# Scan for vulnerabilities
govulncheck ./...
```

## Production Checklist

- [ ] All user input is validated and sanitized
- [ ] File uploads have size, type, and content validation
- [ ] Sensitive configuration is in environment variables
- [ ] HTTPS is enforced
- [ ] Authentication and authorization are implemented
- [ ] Rate limiting is configured
- [ ] Security headers are set
- [ ] Error messages don't leak sensitive information
- [ ] File permissions are properly restricted
- [ ] Logging and monitoring are configured
- [ ] Dependencies are up to date and audited
- [ ] The i18n editor is protected or disabled

## Reporting Security Vulnerabilities

If you discover a security vulnerability in GoKit, please follow the guidelines in [SECURITY.md](../SECURITY.md).

## Additional Resources

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Go Security Best Practices](https://go.dev/doc/security/best-practices)
- [CWE/SANS Top 25 Most Dangerous Software Errors](https://cwe.mitre.org/top25/)
