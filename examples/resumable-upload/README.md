# Resumable Upload Example

This example demonstrates the GoKit resumable upload functionality, which allows for uploading large files in chunks with the ability to resume interrupted uploads.

## Features

- **Chunked Uploads**: Large files are split into manageable chunks
- **Resumable**: Uploads can be resumed if interrupted
- **Progress Tracking**: Real-time progress updates
- **Abort Functionality**: Uploads can be cancelled mid-process
- **Validation**: File size and type validation
- **Web Interface**: User-friendly HTML interface for testing

## How It Works

1. **Initiate Upload**: Create an upload session with file metadata
2. **Upload Chunks**: Upload file data in chunks (default 1MB chunks)
3. **Track Progress**: Monitor upload progress and status
4. **Complete Upload**: Combine all chunks into the final file
5. **Cleanup**: Remove temporary chunk files

## API Endpoints

### POST /upload
Initiate a new upload session.

**Request Body:**
```json
{
  "file_name": "example.jpg",
  "total_size": 5242880,
  "mime_type": "image/jpeg",
  "chunk_size": 1048576
}
```

**Response:**
```json
{
  "file_id": "abc123...",
  "file_name": "example.jpg",
  "total_size": 5242880,
  "chunk_size": 1048576,
  "total_chunks": 5,
  "mime_type": "image/jpeg",
  "status": "uploading",
  "chunks": {},
  "created_at": "2024-01-01T12:00:00Z",
  "updated_at": "2024-01-01T12:00:00Z"
}
```

### PUT /upload?file_id=abc123&chunk_number=0
Upload a specific chunk.

**Request Body:** Raw chunk data

**Response:** 200 OK

### GET /upload?file_id=abc123
Get upload status and progress.

**Response:**
```json
{
  "file_id": "abc123...",
  "status": "uploading",
  "chunks": {
    "0": {
      "chunk_number": 0,
      "size": 1048576,
      "checksum": "md5hash...",
      "uploaded_at": "2024-01-01T12:00:00Z"
    }
  }
}
```

### DELETE /upload?file_id=abc123
Abort an upload session.

**Response:** 200 OK

## Running the Example

1. **Install dependencies:**
   ```bash
   go mod tidy
   ```

2. **Run the server:**
   ```bash
   go run main.go
   ```

3. **Open your browser:**
   Visit `http://localhost:8080`

4. **Test the upload:**
   - Select a file (images or PDFs supported)
   - Click "Start Upload" to begin
   - Watch the progress bar
   - Use "Abort Upload" to cancel if needed

## Configuration

The example is configured with:
- **Max file size**: 100MB
- **Allowed types**: JPEG, PNG, PDF
- **Chunk size**: 1MB (configurable)
- **Storage**: Local filesystem (`/tmp/uploads`)

## Code Example

```go
package main

import (
    "github.com/kdsmith18542/gokit/upload"
    "github.com/kdsmith18542/gokit/upload/storage"
)

func main() {
    // Create storage backend
    localStorage := storage.NewLocal("/tmp/uploads")
    
    // Configure upload options
    options := upload.Options{
        MaxFileSize:      100 * 1024 * 1024, // 100MB
        AllowedMIMETypes: []string{"image/jpeg", "image/png"},
    }
    
    // Create resumable processor
    processor := upload.NewResumableProcessor(localStorage, options)
    
    // Handle HTTP requests
    http.HandleFunc("/upload", processor.HandleResumableUpload)
}
```

## Benefits

- **Reliability**: Uploads can survive network interruptions
- **Scalability**: Large files don't overwhelm server memory
- **User Experience**: Progress feedback and abort capability
- **Efficiency**: Parallel chunk uploads possible
- **Validation**: Built-in file validation and security

## Use Cases

- Video uploads
- Large document uploads
- Backup file uploads
- Media file uploads
- Any large file transfer scenario

## Notes

- Chunks are stored temporarily and cleaned up after completion
- Upload sessions expire after 24 hours
- The example uses local storage, but any storage backend can be used
- Real-world implementations should add authentication and authorization 