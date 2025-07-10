# File Upload & Storage

The `upload` package provides a high-level, secure API for processing file uploads, including large files and direct-to-cloud transfers. It supports multiple storage backends and includes post-processing hooks.

## Table of Contents

- [Quick Start](#quick-start)
- [Core Concepts](#core-concepts)
- [Storage Backends](#storage-backends)
- [File Processing](#file-processing)
- [Post-Processing Hooks](#post-processing-hooks)
- [CLI Integration](#cli-integration)
- [Middleware Integration](#middleware-integration)
- [Advanced Examples](#advanced-examples)

## Quick Start

```go
package main

import (
    "context"
    "net/http"
    "github.com/kdsmith18542/gokit/upload"
    "github.com/kdsmith18542/gokit/upload/storage"
)

func main() {
    // Initialize storage backend
    s3Storage, _ := storage.NewS3(storage.S3Config{
        Bucket: "my-bucket",
        Region: "us-west-2",
    })
    
    // Create upload processor
    processor := upload.NewProcessor(s3Storage, upload.Options{
        MaxFileSize: 10 * 1024 * 1024, // 10MB
        AllowedMIMETypes: []string{"image/jpeg", "image/png", "application/pdf"},
    })
    
    // Register post-processing hooks
    processor.OnSuccess(func(ctx context.Context, result upload.Result) {
        // Generate thumbnail, update database, etc.
    })
    
    // Set up routes
    http.HandleFunc("/upload", uploadHandler(processor))
    http.ListenAndServe(":8080", nil)
}

func uploadHandler(processor *upload.Processor) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        results, err := processor.Process(r.Context(), r, "file")
        if err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        
        // Return upload results
        for _, result := range results {
            fmt.Fprintf(w, "File uploaded: %s\n", result.URL)
        }
    }
}
```

## Core Concepts

### Processor

The `upload.Processor` is the main component that handles file uploads:

```go
processor := upload.NewProcessor(storage, options)
```

### Storage Interface

All storage backends implement the `storage.Storage` interface:

```go
type Storage interface {
    Store(ctx context.Context, filename string, reader io.Reader) (string, error)
    GetReader(ctx context.Context, filename string) (io.Reader, error)
    Delete(ctx context.Context, filename string) error
    GetURL(filename string) string
    GetSignedURL(filename string, expires time.Duration) (string, error)
    Exists(filename string) bool
    GetSize(filename string) (int64, error)
    ListFiles(prefix string) ([]string, error)
    GetBucketInfo() (*BucketInfo, error)
    Close() error
}
```

### File and Result

```go
type File struct {
    Name        string
    Size        int64
    ContentType string
    Reader      io.Reader
    Checksum    string
}

type Result struct {
    URL          string
    Path         string
    OriginalName string
    Size         int64
    ContentType  string
    Checksum     string
    Metadata     map[string]string
}
```

## Storage Backends

### Local Storage

Store files on the local filesystem:

```go
localStorage, err := storage.NewLocal(storage.LocalConfig{
    BasePath: "/var/uploads",
    BaseURL:  "https://example.com/uploads",
})
```

### S3 Storage

Store files in Amazon S3:

```go
s3Storage, err := storage.NewS3(storage.S3Config{
    Bucket:          "my-bucket",
    Region:          "us-west-2",
    AccessKeyID:     "your-access-key",
    SecretAccessKey: "your-secret-key",
    BaseURL:         "https://my-bucket.s3.amazonaws.com",
})
```

### Google Cloud Storage

Store files in Google Cloud Storage:

```go
gcsStorage, err := storage.NewGCS(storage.GCSConfig{
    Bucket:   "my-bucket",
    ProjectID: "my-project",
    BaseURL:  "https://storage.googleapis.com/my-bucket",
})
```

### Azure Blob Storage

Store files in Azure Blob Storage:

```go
azureStorage, err := storage.NewAzure(storage.AzureConfig{
    AccountName: "myaccount",
    AccountKey:  "your-account-key",
    Container:   "uploads",
    BaseURL:     "https://myaccount.blob.core.windows.net/uploads",
})
```

### In-Memory Storage

For testing and development:

```go
memoryStorage := storage.NewMemory()
```

## File Processing

### Basic Upload Processing

```go
// Process files from multipart form
results, err := processor.Process(ctx, r, "file")
if err != nil {
    return err
}

// Process multiple files
results, err := processor.ProcessMultiple(ctx, r, "files")
if err != nil {
    return err
}
```

### Validation Options

```go
processor := upload.NewProcessor(storage, upload.Options{
    MaxFileSize:      10 * 1024 * 1024, // 10MB
    AllowedMIMETypes: []string{"image/jpeg", "image/png"},
    AllowedExtensions: []string{".jpg", ".png", ".pdf"},
    ValidateChecksum: true,
    ChecksumAlgorithm: "sha256",
})
```

### Custom Validation

```go
processor := upload.NewProcessor(storage, upload.Options{
    Validator: func(file *upload.File) error {
        // Custom validation logic
        if file.Size > 5*1024*1024 {
            return errors.New("file too large")
        }
        return nil
    },
})
```

## Post-Processing Hooks

### Success Hooks

Register functions to run after successful uploads:

```go
processor.OnSuccess(func(ctx context.Context, result upload.Result) {
    // Generate thumbnail
    if strings.HasPrefix(result.ContentType, "image/") {
        generateThumbnail(ctx, result.Path)
    }
    
    // Update database
    updateFileRecord(ctx, result)
    
    // Send notification
    notifyUploadComplete(ctx, result)
})
```

### Error Hooks

Register functions to run when uploads fail:

```go
processor.OnError(func(ctx context.Context, file *upload.File, err error) {
    // Log error
    log.Printf("Upload failed for %s: %v", file.Name, err)
    
    // Clean up partial uploads
    cleanupPartialUpload(ctx, file)
    
    // Send error notification
    notifyUploadError(ctx, file, err)
})
```

### Multiple Hooks

You can register multiple hooks:

```go
processor.OnSuccess(
    generateThumbnailHook,
    updateDatabaseHook,
    sendNotificationHook,
)
```

## CLI Integration

The upload package integrates with the GoKit CLI for storage backend management:

### Verify Credentials

```bash
# Verify S3 credentials
gokit-cli upload verify-credentials --backend s3 --bucket my-bucket --region us-west-2

# Verify GCS credentials
gokit-cli upload verify-credentials --backend gcs --bucket my-bucket --credentials-file ./key.json

# Verify Azure credentials
gokit-cli upload verify-credentials --backend azure --azure-container my-container
```

### List Files

```bash
# List files in S3 bucket
gokit-cli upload list-files --backend s3 --bucket my-bucket --region us-west-2

# List files in GCS bucket
gokit-cli upload list-files --backend gcs --bucket my-bucket --credentials-file ./key.json

# List files in Azure container
gokit-cli upload list-files --backend azure --azure-container my-container
```

### Upload File

```bash
# Upload to S3
gokit-cli upload upload-file ./image.jpg --backend s3 --bucket my-bucket --region us-west-2

# Upload to GCS
gokit-cli upload upload-file ./document.pdf --backend gcs --bucket my-bucket --credentials-file ./key.json

# Upload to Azure
gokit-cli upload upload-file ./data.csv --backend azure --azure-container my-container
```

## Middleware Integration

Use the upload middleware for automatic file processing:

```go
// Create middleware
uploadMiddleware := upload.Middleware(processor)

// Apply to routes
http.HandleFunc("/upload", uploadMiddleware(uploadHandler))

// In your handler, get upload results from context
func uploadHandler(w http.ResponseWriter, r *http.Request) {
    results := upload.UploadResultsFromContext(r.Context())
    
    // Process upload results
    for _, result := range results {
        fmt.Fprintf(w, "Uploaded: %s\n", result.URL)
    }
}
```

### Middleware Options

```go
// With custom field name
uploadMiddleware := upload.Middleware(processor, upload.WithFieldName("files"))

// With custom error handler
uploadMiddleware := upload.Middleware(processor, upload.WithErrorHandler(func(w http.ResponseWriter, err error) {
    // Custom error response
}))

// With file size limit override
uploadMiddleware := upload.Middleware(processor, upload.WithMaxFileSize(5*1024*1024))
```

## Advanced Examples

### Image Processing Pipeline

```go
type ImageProcessor struct {
    uploadProcessor *upload.Processor
    thumbnailSize   int
}

func NewImageProcessor(storage storage.Storage) *ImageProcessor {
    processor := upload.NewProcessor(storage, upload.Options{
        MaxFileSize:      5 * 1024 * 1024, // 5MB
        AllowedMIMETypes: []string{"image/jpeg", "image/png", "image/gif"},
        ValidateChecksum: true,
    })
    
    ip := &ImageProcessor{
        uploadProcessor: processor,
        thumbnailSize:   200,
    }
    
    // Register image processing hooks
    processor.OnSuccess(ip.processImage)
    
    return ip
}

func (ip *ImageProcessor) processImage(ctx context.Context, result upload.Result) {
    if !strings.HasPrefix(result.ContentType, "image/") {
        return
    }
    
    // Download the uploaded image
    reader, err := ip.downloadFile(ctx, result.Path)
    if err != nil {
        log.Printf("Failed to download image: %v", err)
        return
    }
    defer reader.Close()
    
    // Generate thumbnail
    thumbnail, err := ip.generateThumbnail(reader, ip.thumbnailSize)
    if err != nil {
        log.Printf("Failed to generate thumbnail: %v", err)
        return
    }
    
    // Upload thumbnail
    thumbnailPath := strings.Replace(result.Path, ".", "_thumb.", 1)
    thumbnailResult, err := ip.uploadThumbnail(ctx, thumbnail, thumbnailPath)
    if err != nil {
        log.Printf("Failed to upload thumbnail: %v", err)
        return
    }
    
    // Update database with thumbnail URL
    ip.updateImageRecord(ctx, result, thumbnailResult)
}

func (ip *ImageProcessor) generateThumbnail(reader io.Reader, size int) (io.Reader, error) {
    // Image processing logic here
    // This is a simplified example
    return reader, nil
}
```

### Document Management System

```go
type DocumentManager struct {
    processor *upload.Processor
    db        *Database
}

func NewDocumentManager(storage storage.Storage, db *Database) *DocumentManager {
    processor := upload.NewProcessor(storage, upload.Options{
        MaxFileSize:      50 * 1024 * 1024, // 50MB
        AllowedMIMETypes: []string{"application/pdf", "text/plain", "application/msword"},
        AllowedExtensions: []string{".pdf", ".txt", ".doc", ".docx"},
        ValidateChecksum: true,
    })
    
    dm := &DocumentManager{
        processor: processor,
        db:        db,
    }
    
    // Register document processing hooks
    processor.OnSuccess(dm.processDocument)
    processor.OnError(dm.handleUploadError)
    
    return dm
}

func (dm *DocumentManager) processDocument(ctx context.Context, result upload.Result) {
    // Create document record
    doc := &Document{
        ID:          generateID(),
        Name:        result.OriginalName,
        Path:        result.Path,
        URL:         result.URL,
        Size:        result.Size,
        ContentType: result.ContentType,
        Checksum:    result.Checksum,
        UploadedAt:  time.Now(),
        Metadata:    result.Metadata,
    }
    
    // Save to database
    if err := dm.db.CreateDocument(ctx, doc); err != nil {
        log.Printf("Failed to save document: %v", err)
        return
    }
    
    // Extract text content (for search)
    if strings.HasPrefix(result.ContentType, "application/pdf") {
        go dm.extractTextContent(ctx, doc)
    }
    
    // Send notification
    go dm.notifyDocumentUploaded(ctx, doc)
}

func (dm *DocumentManager) handleUploadError(ctx context.Context, file *upload.File, err error) {
    // Log error
    log.Printf("Document upload failed: %s - %v", file.Name, err)
    
    // Create error record
    errorRecord := &UploadError{
        FileName: file.Name,
        Error:    err.Error(),
        Time:     time.Now(),
    }
    
    dm.db.CreateUploadError(ctx, errorRecord)
}
```

### Resumable Uploads

The upload package includes built-in support for resumable uploads with chunk-based processing:

```go
// Create resumable upload processor
resumableProcessor := upload.NewResumableProcessor(storage, upload.ResumableOptions{
    ChunkSize: 1024 * 1024, // 1MB chunks
    MaxChunks: 1000,         // Maximum number of chunks
    Expiry:   24 * time.Hour, // Session expiry
})

// Handle upload initiation
func initiateUpload(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Filename string `json:"filename"`
        FileSize int64  `json:"file_size"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    session, err := resumableProcessor.InitiateUpload(r.Context(), req.Filename, req.FileSize)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    json.NewEncoder(w).Encode(map[string]interface{}{
        "session_id": session.ID,
        "chunk_size": session.ChunkSize,
        "total_chunks": session.TotalChunks,
    })
}

// Handle chunk upload
func uploadChunk(w http.ResponseWriter, r *http.Request) {
    sessionID := r.URL.Query().Get("session_id")
    chunkIndex := r.URL.Query().Get("chunk")
    
    chunkIndexInt, _ := strconv.Atoi(chunkIndex)
    
    err := resumableProcessor.UploadChunk(r.Context(), sessionID, chunkIndexInt, r.Body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    w.WriteHeader(http.StatusOK)
}

// Complete upload
func completeUpload(w http.ResponseWriter, r *http.Request) {
    sessionID := r.URL.Query().Get("session_id")
    
    result, err := resumableProcessor.CompleteUpload(r.Context(), sessionID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    json.NewEncoder(w).Encode(result)
}
```

For more complex scenarios, you can also implement custom resumable upload handling:

```go
type ResumableUpload struct {
    processor *upload.Processor
    sessions  map[string]*UploadSession
    mu        sync.RWMutex
}

type UploadSession struct {
    ID       string
    FilePath string
    Size     int64
    Chunks   map[int]bool
    Complete bool
}

func (ru *ResumableUpload) InitiateUpload(w http.ResponseWriter, r *http.Request) {
    var req struct {
        FileName string `json:"file_name"`
        FileSize int64  `json:"file_size"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    sessionID := generateSessionID()
    session := &UploadSession{
        ID:       sessionID,
        FilePath: fmt.Sprintf("uploads/%s/%s", sessionID, req.FileName),
        Size:     req.FileSize,
        Chunks:   make(map[int]bool),
    }
    
    ru.mu.Lock()
    ru.sessions[sessionID] = session
    ru.mu.Unlock()
    
    json.NewEncoder(w).Encode(map[string]interface{}{
        "session_id": sessionID,
        "chunk_size": 1024 * 1024, // 1MB chunks
    })
}

func (ru *ResumableUpload) UploadChunk(w http.ResponseWriter, r *http.Request) {
    sessionID := r.URL.Query().Get("session_id")
    chunkIndex := r.URL.Query().Get("chunk")
    
    ru.mu.RLock()
    session, exists := ru.sessions[sessionID]
    ru.mu.RUnlock()
    
    if !exists {
        http.Error(w, "Session not found", http.StatusNotFound)
        return
    }
    
    // Upload chunk to temporary storage
    chunkPath := fmt.Sprintf("temp/%s/chunk_%s", sessionID, chunkIndex)
    // ... upload logic ...
    
    // Mark chunk as uploaded
    session.Chunks[chunkIndex] = true
    
    // Check if all chunks are uploaded
    if ru.isUploadComplete(session) {
        ru.completeUpload(session)
    }
    
    w.WriteHeader(http.StatusOK)
}
```

### Testing Uploads

```go
func TestFileUpload(t *testing.T) {
    // Create in-memory storage for testing
    storage := storage.NewMemory()
    processor := upload.NewProcessor(storage, upload.Options{
        MaxFileSize: 1024 * 1024, // 1MB
    })
    
    // Create test file
    testData := []byte("test file content")
    file := &upload.File{
        Name:        "test.txt",
        Size:        int64(len(testData)),
        ContentType: "text/plain",
        Reader:      bytes.NewReader(testData),
    }
    
    // Test upload
    result, err := processor.UploadFile(context.Background(), file)
    if err != nil {
        t.Fatalf("Upload failed: %v", err)
    }
    
    // Verify result
    if result.OriginalName != "test.txt" {
        t.Errorf("expected name %s, got %s", "test.txt", result.OriginalName)
    }
    
    if result.Size != int64(len(testData)) {
        t.Errorf("expected size %d, got %d", len(testData), result.Size)
    }
}

func TestUploadMiddleware(t *testing.T) {
    storage := storage.NewMemory()
    processor := upload.NewProcessor(storage, upload.Options{})
    middleware := upload.UploadMiddleware(processor, "file", nil)
    
    // Create test request with file
    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)
    
    part, err := writer.CreateFormFile("file", "test.txt")
    if err != nil {
        t.Fatal(err)
    }
    part.Write([]byte("test content"))
    writer.Close()
    
    req := httptest.NewRequest("POST", "/upload", body)
    req.Header.Set("Content-Type", writer.FormDataContentType())
    
    // Test middleware
    var results []upload.Result
    handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        results = upload.UploadResultsFromContext(r.Context())
    }))
    
    handler.ServeHTTP(httptest.NewRecorder(), req)
    
    if len(results) != 1 {
        t.Errorf("expected 1 result, got %d", len(results))
    }
}
```

## Best Practices

1. **Validate Files**: Always validate file types, sizes, and content
2. **Use Checksums**: Enable checksum validation for data integrity
3. **Handle Errors**: Implement proper error handling and cleanup
4. **Post-Processing**: Use hooks for async operations like thumbnail generation
5. **Security**: Validate file content, not just extensions
6. **Monitoring**: Log upload metrics and errors
7. **Cleanup**: Implement cleanup for failed uploads
8. **Testing**: Use in-memory storage for testing

## Performance Considerations

- Use streaming uploads for large files
- Implement resumable uploads for better user experience
- Use async processing for post-upload operations
- Consider CDN integration for better delivery
- Monitor storage costs and implement lifecycle policies 