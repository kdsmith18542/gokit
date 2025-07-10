package upload

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// PresignedOptions defines options for generating pre-signed upload URLs
type PresignedOptions struct {
	// Filename is the desired filename for the upload
	Filename string
	// ContentType is the expected MIME type of the file
	ContentType string
	// Expiration is how long the URL should be valid
	Expiration time.Duration
	// MaxFileSize is the maximum file size allowed (in bytes)
	MaxFileSize int64
	// Metadata is additional metadata to include with the upload
	Metadata map[string]string
}

// PreSignedResult contains information about a generated pre-signed URL
type PreSignedResult struct {
	// URL is the pre-signed URL for uploading
	URL string
	// Fields contains form fields that must be included in the upload
	Fields map[string]string
	// ExpiresAt is when the URL expires
	ExpiresAt time.Time
	// Filename is the filename for the upload
	Filename string
}

// GenerateUploadURL generates a pre-signed URL for direct client-side uploads.
// This allows clients to upload files directly to storage without going through the server.
//
// Example:
//
//	opts := PresignedOptions{
//	    Filename:    "avatar.jpg",
//	    ContentType: "image/jpeg",
//	    Expiration:  15 * time.Minute,
//	    MaxFileSize: 5 * 1024 * 1024,
//	}
//	result, err := processor.GenerateUploadURL(ctx, opts)
//	if err != nil {
//	    // Handle error
//	}
//	// Return result.URL to client for direct upload
func (p *Processor) GenerateUploadURL(ctx context.Context, opts PresignedOptions) (*PreSignedResult, error) {
	// Validate options
	if opts.Filename == "" {
		return nil, fmt.Errorf("filename is required")
	}
	if opts.Expiration <= 0 {
		opts.Expiration = 15 * time.Minute // Default expiration
	}
	if opts.MaxFileSize <= 0 {
		opts.MaxFileSize = p.options.MaxFileSize
	}

	// Validate file size against processor limits
	if p.options.MaxFileSize > 0 && opts.MaxFileSize > p.options.MaxFileSize {
		return nil, fmt.Errorf("max file size exceeds processor limit: %d > %d", opts.MaxFileSize, p.options.MaxFileSize)
	}

	// Validate content type if specified
	if opts.ContentType != "" && len(p.options.AllowedMIMETypes) > 0 {
		allowed := false
		for _, allowedType := range p.options.AllowedMIMETypes {
			if allowedType == opts.ContentType || allowedType == "*" {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("content type not allowed: %s", opts.ContentType)
		}
	}

	// Generate unique filename if not provided
	filename := opts.Filename
	if filename == "" {
		filename = p.generateFilename("")
	}

	// Generate pre-signed URL using storage backend
	url, err := p.storage.GetSignedURL(filename, opts.Expiration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate pre-signed URL: %v", err)
	}

	// Calculate expiration time
	expiresAt := time.Now().Add(opts.Expiration)

	// Create result
	result := &PreSignedResult{
		URL:       url,
		Fields:    make(map[string]string),
		ExpiresAt: expiresAt,
		Filename:  filename,
	}

	// Add metadata fields if supported by storage backend
	if opts.Metadata != nil {
		for key, value := range opts.Metadata {
			result.Fields[key] = value
		}
	}

	return result, nil
}

// GenerateUploadURLs generates multiple pre-signed URLs for batch uploads.
// This is useful for allowing clients to upload multiple files at once.
//
// Example:
//
//	opts := []PresignedOptions{
//	    {Filename: "file1.jpg", ContentType: "image/jpeg"},
//	    {Filename: "file2.png", ContentType: "image/png"},
//	}
//	results, err := processor.GenerateUploadURLs(ctx, opts)
func (p *Processor) GenerateUploadURLs(ctx context.Context, opts []PresignedOptions) ([]*PreSignedResult, error) {
	results := make([]*PreSignedResult, len(opts))

	for i, opt := range opts {
		result, err := p.GenerateUploadURL(ctx, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to generate URL for %s: %v", opt.Filename, err)
		}
		results[i] = result
	}

	return results, nil
}

// ValidatePreSignedUpload validates a file that was uploaded via a pre-signed URL.
// This should be called after a client uploads a file to verify it meets requirements.
//
// Example:
//
//	err := processor.ValidatePreSignedUpload(ctx, "uploaded-file.jpg", fileSize, contentType)
//	if err != nil {
//	    // Handle validation error
//	}
func (p *Processor) ValidatePreSignedUpload(ctx context.Context, filename string, fileSize int64, contentType string) error {
	// Validate file size
	if p.options.MaxFileSize > 0 && fileSize > p.options.MaxFileSize {
		return fmt.Errorf("file too large: %d bytes (max: %d)", fileSize, p.options.MaxFileSize)
	}

	// Validate content type
	if len(p.options.AllowedMIMETypes) > 0 {
		if !p.isAllowedMIMEType(contentType) {
			return fmt.Errorf("file type not allowed: %s", contentType)
		}
	}

	// Validate file extension
	if len(p.options.AllowedExtensions) > 0 {
		ext := strings.ToLower(filepath.Ext(filename))
		if !p.isAllowedExtension(ext) {
			return fmt.Errorf("file extension not allowed: %s", ext)
		}
	}

	return nil
}

// GetStatus checks the status of a file uploaded via pre-signed URL.
// This can be used to verify if the upload completed successfully.
//
// Example:
//
//	status, err := processor.GetStatus(ctx, "uploaded-file.jpg")
//	if err != nil {
//	    // Handle error
//	}
//	if status.Exists {
//	    // File was uploaded successfully
//	}
func (p *Processor) GetStatus(ctx context.Context, filename string) (*PresignedStatus, error) {
	exists := p.storage.Exists(filename)

	status := &PresignedStatus{
		Filename: filename,
		Exists:   exists,
	}

	if exists {
		size, err := p.storage.GetSize(filename)
		if err == nil {
			status.Size = size
		}
		status.URL = p.storage.GetURL(filename)
	}

	return status, nil
}

// PresignedStatus contains information about an uploaded file
type PresignedStatus struct {
	// Filename is the name of the uploaded file
	Filename string
	// Exists indicates if the file exists in storage
	Exists bool
	// Size is the file size in bytes (if file exists)
	Size int64
	// URL is the public URL for accessing the file (if file exists)
	URL string
}
