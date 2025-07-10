package upload

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kdsmith18542/gokit/observability"
	"github.com/kdsmith18542/gokit/upload/storage"
)

// ChunkInfo represents information about a file chunk
type ChunkInfo struct {
	ChunkNumber int    `json:"chunk_number"`
	TotalChunks int    `json:"total_chunks"`
	ChunkSize   int64  `json:"chunk_size"`
	TotalSize   int64  `json:"total_size"`
	FileID      string `json:"file_id"`
	FileName    string `json:"file_name"`
	MIMEType    string `json:"mime_type"`
	Checksum    string `json:"checksum"`
}

// UploadSession represents a resumable upload session
type UploadSession struct {
	FileID      string                 `json:"file_id"`
	FileName    string                 `json:"file_name"`
	TotalSize   int64                  `json:"total_size"`
	ChunkSize   int64                  `json:"chunk_size"`
	TotalChunks int                    `json:"total_chunks"`
	MIMEType    string                 `json:"mime_type"`
	Status      string                 `json:"status"` // "uploading", "completed", "failed"
	Chunks      map[int]*ChunkMetadata `json:"chunks"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	mu          sync.RWMutex
}

// ChunkMetadata represents metadata for a single chunk
type ChunkMetadata struct {
	ChunkNumber int       `json:"chunk_number"`
	Size        int64     `json:"size"`
	Checksum    string    `json:"checksum"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

// ResumableProcessor handles resumable file uploads
type ResumableProcessor struct {
	storage    storage.Storage
	options    Options
	sessions   map[string]*UploadSession
	sessionTTL time.Duration
	mu         sync.RWMutex
}

// NewResumableProcessor creates a new resumable upload processor
func NewResumableProcessor(storage storage.Storage, options Options) *ResumableProcessor {
	return &ResumableProcessor{
		storage:    storage,
		options:    options,
		sessions:   make(map[string]*UploadSession),
		sessionTTL: 24 * time.Hour, // Sessions expire after 24 hours
	}
}

// InitiateUpload starts a new resumable upload session
func (rp *ResumableProcessor) InitiateUpload(ctx context.Context, fileName string, totalSize int64, mimeType string, chunkSize int64) (*UploadSession, error) {
	// Validate file size
	if rp.options.MaxFileSize > 0 && totalSize > rp.options.MaxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes (max: %d)", totalSize, rp.options.MaxFileSize)
	}

	// Validate MIME type
	if len(rp.options.AllowedMIMETypes) > 0 {
		if !rp.isAllowedMIMEType(mimeType) {
			return nil, fmt.Errorf("file type not allowed: %s", mimeType)
		}
	}

	// Generate file ID
	fileID := rp.generateFileID()

	// Calculate total chunks
	totalChunks := int((totalSize + chunkSize - 1) / chunkSize)

	// Create session
	session := &UploadSession{
		FileID:      fileID,
		FileName:    fileName,
		TotalSize:   totalSize,
		ChunkSize:   chunkSize,
		TotalChunks: totalChunks,
		MIMEType:    mimeType,
		Status:      "uploading",
		Chunks:      make(map[int]*ChunkMetadata),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Store session
	rp.mu.Lock()
	rp.sessions[fileID] = session
	rp.mu.Unlock()

	return session, nil
}

// UploadChunk uploads a single chunk of a file
func (rp *ResumableProcessor) UploadChunk(ctx context.Context, fileID string, chunkNumber int, chunkData io.Reader) error {
	// Get session
	rp.mu.RLock()
	session, exists := rp.sessions[fileID]
	rp.mu.RUnlock()

	if !exists {
		return fmt.Errorf("upload session not found: %s", fileID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Check if chunk already exists
	if _, exists := session.Chunks[chunkNumber]; exists {
		return fmt.Errorf("chunk %d already uploaded", chunkNumber)
	}

	// Validate chunk number
	if chunkNumber < 0 || chunkNumber >= session.TotalChunks {
		return fmt.Errorf("invalid chunk number: %d (total chunks: %d)", chunkNumber, session.TotalChunks)
	}

	// Read chunk data
	chunkBytes, err := io.ReadAll(chunkData)
	if err != nil {
		return fmt.Errorf("failed to read chunk data: %v", err)
	}

	// Calculate chunk checksum
	checksum := rp.calculateChecksum(chunkBytes)

	// Store chunk
	chunkPath := fmt.Sprintf("chunks/%s/chunk_%d", fileID, chunkNumber)
	chunkReader := strings.NewReader(string(chunkBytes))

	_, err = rp.storage.Store(chunkPath, chunkReader)
	if err != nil {
		return fmt.Errorf("failed to store chunk: %v", err)
	}

	// Update session
	session.Chunks[chunkNumber] = &ChunkMetadata{
		ChunkNumber: chunkNumber,
		Size:        int64(len(chunkBytes)),
		Checksum:    checksum,
		UploadedAt:  time.Now(),
	}
	session.UpdatedAt = time.Now()

	return nil
}

// GetUploadStatus returns the current status of an upload session
func (rp *ResumableProcessor) GetUploadStatus(fileID string) (*UploadSession, error) {
	rp.mu.RLock()
	session, exists := rp.sessions[fileID]
	rp.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("upload session not found: %s", fileID)
	}

	session.mu.RLock()
	defer session.mu.RUnlock()

	// Check if all chunks are uploaded
	if len(session.Chunks) == session.TotalChunks {
		session.Status = "completed"
	}

	return session, nil
}

// CompleteUpload finalizes the upload by combining all chunks
func (rp *ResumableProcessor) CompleteUpload(ctx context.Context, fileID string) (*Result, error) {
	// Get session
	rp.mu.RLock()
	session, exists := rp.sessions[fileID]
	rp.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("upload session not found: %s", fileID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Check if all chunks are uploaded
	if len(session.Chunks) != session.TotalChunks {
		return nil, fmt.Errorf("not all chunks uploaded: %d/%d", len(session.Chunks), session.TotalChunks)
	}

	// Combine chunks
	finalPath, err := rp.combineChunks(ctx, session)
	if err != nil {
		session.Status = "failed"
		return nil, fmt.Errorf("failed to combine chunks: %v", err)
	}

	// Get the final URL
	url := rp.storage.GetURL(finalPath)

	// Calculate final checksum
	finalChecksum, err := rp.calculateFinalChecksum(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate final checksum: %v", err)
	}

	// Update session status
	session.Status = "completed"
	session.UpdatedAt = time.Now()

	// Clean up chunks
	go rp.cleanupChunks(fileID)

	return &Result{
		OriginalName: session.FileName,
		Size:         session.TotalSize,
		MIMEType:     session.MIMEType,
		URL:          url,
		Path:         finalPath,
		Checksum:     finalChecksum,
		UploadedAt:   time.Now(),
	}, nil
}

// AbortUpload cancels an upload session
func (rp *ResumableProcessor) AbortUpload(fileID string) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if _, exists := rp.sessions[fileID]; !exists {
		return fmt.Errorf("upload session not found: %s", fileID)
	}

	// Clean up chunks
	go rp.cleanupChunks(fileID)

	// Remove session
	delete(rp.sessions, fileID)

	return nil
}

// HandleResumableUpload handles HTTP requests for resumable uploads
func (rp *ResumableProcessor) HandleResumableUpload(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		rp.handleInitiateUpload(w, r)
	case "PUT":
		rp.handleUploadChunk(w, r)
	case "GET":
		rp.handleGetStatus(w, r)
	case "DELETE":
		rp.handleAbortUpload(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleInitiateUpload handles upload initiation
func (rp *ResumableProcessor) handleInitiateUpload(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FileName  string `json:"file_name"`
		TotalSize int64  `json:"total_size"`
		MIMEType  string `json:"mime_type"`
		ChunkSize int64  `json:"chunk_size"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ChunkSize == 0 {
		req.ChunkSize = 1024 * 1024 // Default 1MB chunks
	}

	session, err := rp.InitiateUpload(r.Context(), req.FileName, req.TotalSize, req.MIMEType, req.ChunkSize)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// handleUploadChunk handles chunk upload
func (rp *ResumableProcessor) handleUploadChunk(w http.ResponseWriter, r *http.Request) {
	fileID := r.URL.Query().Get("file_id")
	chunkNumberStr := r.URL.Query().Get("chunk_number")

	if fileID == "" || chunkNumberStr == "" {
		http.Error(w, "Missing file_id or chunk_number", http.StatusBadRequest)
		return
	}

	chunkNumber, err := strconv.Atoi(chunkNumberStr)
	if err != nil {
		http.Error(w, "Invalid chunk_number", http.StatusBadRequest)
		return
	}

	err = rp.UploadChunk(r.Context(), fileID, chunkNumber, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleGetStatus handles status requests
func (rp *ResumableProcessor) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	fileID := r.URL.Query().Get("file_id")
	if fileID == "" {
		http.Error(w, "Missing file_id", http.StatusBadRequest)
		return
	}

	session, err := rp.GetUploadStatus(fileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// handleAbortUpload handles upload abortion
func (rp *ResumableProcessor) handleAbortUpload(w http.ResponseWriter, r *http.Request) {
	fileID := r.URL.Query().Get("file_id")
	if fileID == "" {
		http.Error(w, "Missing file_id", http.StatusBadRequest)
		return
	}

	err := rp.AbortUpload(fileID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Helper methods

func (rp *ResumableProcessor) generateFileID() string {
	// Generate a unique file ID
	randomBytes := make([]byte, 16)
	io.ReadFull(rand.Reader, randomBytes)
	return hex.EncodeToString(randomBytes)
}

func (rp *ResumableProcessor) isAllowedMIMEType(mimeType string) bool {
	for _, allowed := range rp.options.AllowedMIMETypes {
		if mimeType == allowed {
			return true
		}
		// Handle wildcard MIME types (e.g., "image/*")
		if strings.HasSuffix(allowed, "/*") {
			baseType := strings.TrimSuffix(allowed, "/*")
			if strings.HasPrefix(mimeType, baseType+"/") {
				return true
			}
		}
	}
	return false
}

func (rp *ResumableProcessor) calculateChecksum(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

func (rp *ResumableProcessor) combineChunks(ctx context.Context, session *UploadSession) (string, error) {
	// Create the final file path
	finalPath := fmt.Sprintf("uploads/%s/%s", session.FileID, session.FileName)

	// Create a temporary file for combining chunks
	tempFile, err := os.CreateTemp("", fmt.Sprintf("resumable_%s_*", session.FileID))
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Sort chunks by chunk number to ensure correct order
	chunkNumbers := make([]int, 0, len(session.Chunks))
	for chunkNum := range session.Chunks {
		chunkNumbers = append(chunkNumbers, chunkNum)
	}
	sort.Ints(chunkNumbers)

	// Combine all chunks in order
	for _, chunkNum := range chunkNumbers {
		chunkMetadata := session.Chunks[chunkNum]
		chunkPath := fmt.Sprintf("chunks/%s/chunk_%d", session.FileID, chunkNum)

		// Read chunk data from storage
		chunkReader, err := rp.storage.GetReader(chunkPath)
		if err != nil {
			return "", fmt.Errorf("failed to read chunk %d: %v", chunkNum, err)
		}
		defer chunkReader.Close()

		// Copy chunk data to temp file
		written, err := io.Copy(tempFile, chunkReader)
		if err != nil {
			return "", fmt.Errorf("failed to write chunk %d to temp file: %v", chunkNum, err)
		}

		// Verify chunk size
		if written != chunkMetadata.Size {
			return "", fmt.Errorf("chunk %d size mismatch: expected %d, got %d", chunkNum, chunkMetadata.Size, written)
		}
	}

	// Reset temp file pointer to beginning
	if _, err := tempFile.Seek(0, 0); err != nil {
		return "", fmt.Errorf("failed to seek temp file: %v", err)
	}

	// Store the combined file
	_, err = rp.storage.Store(finalPath, tempFile)
	if err != nil {
		return "", fmt.Errorf("failed to store combined file: %v", err)
	}

	return finalPath, nil
}

func (rp *ResumableProcessor) calculateFinalChecksum(ctx context.Context, session *UploadSession) (string, error) {
	// Get the final file path
	finalPath := fmt.Sprintf("uploads/%s/%s", session.FileID, session.FileName)

	// Read the combined file from storage
	reader, err := rp.storage.GetReader(finalPath)
	if err != nil {
		return "", fmt.Errorf("failed to read combined file: %v", err)
	}
	defer reader.Close()

	// Calculate SHA256 checksum
	hash := sha256.New()
	_, err = io.Copy(hash, reader)
	if err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %v", err)
	}

	// Return hex-encoded checksum
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (rp *ResumableProcessor) cleanupChunks(fileID string) {
	// Clean up individual chunks after successful upload
	ctx := context.Background()

	// Get the session to know how many chunks to clean up
	rp.mu.RLock()
	session, exists := rp.sessions[fileID]
	rp.mu.RUnlock()

	if !exists {
		return // Session already cleaned up
	}

	// Delete all chunk files
	for chunkNum := range session.Chunks {
		chunkPath := fmt.Sprintf("chunks/%s/chunk_%d", fileID, chunkNum)

		// Delete chunk from storage
		if err := rp.storage.Delete(chunkPath); err != nil {
			// Log error but continue with other chunks
			observability.LogError(ctx, "Failed to delete chunk file", err, map[string]string{
				"file_id":    fileID,
				"chunk_num":  fmt.Sprintf("%d", chunkNum),
				"chunk_path": chunkPath,
			})
		}
	}

	// Try to clean up the chunks directory
	chunksDir := fmt.Sprintf("chunks/%s", fileID)
	if err := rp.storage.Delete(chunksDir); err != nil {
		// Log error but don't fail - directory might not be empty or might not exist
		observability.LogError(ctx, "Failed to delete chunks directory", err, map[string]string{
			"file_id":    fileID,
			"chunks_dir": chunksDir,
		})
	}

	// Log successful cleanup
	observability.LogInfo(ctx, "Chunks cleaned up successfully", map[string]string{
		"file_id":     fileID,
		"chunk_count": fmt.Sprintf("%d", len(session.Chunks)),
	})
}
