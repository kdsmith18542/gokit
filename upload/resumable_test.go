package upload

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kdsmith18542/gokit/upload/storage"
)

func TestNewResumableProcessor(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	options := Options{
		MaxFileSize:      10 * 1024 * 1024, // 10MB
		AllowedMIMETypes: []string{"image/jpeg", "image/png"},
	}

	processor := NewResumableProcessor(mockStorage, options)

	if processor == nil {
		t.Fatal("Expected processor to be created")
	}

	if processor.storage != mockStorage {
		t.Error("Expected storage to be set")
	}

	if processor.options.MaxFileSize != options.MaxFileSize {
		t.Error("Expected options to be set")
	}
}

func TestInitiateUpload(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	options := Options{
		MaxFileSize:      10 * 1024 * 1024, // 10MB
		AllowedMIMETypes: []string{"image/jpeg", "image/png"},
	}

	processor := NewResumableProcessor(mockStorage, options)

	ctx := context.Background()
	session, err := processor.InitiateUpload(ctx, "test.jpg", 5*1024*1024, "image/jpeg", 1024*1024)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if session == nil {
		t.Fatal("Expected session to be created")
	}

	if session.FileName != "test.jpg" {
		t.Errorf("Expected filename 'test.jpg', got: %s", session.FileName)
	}

	if session.TotalSize != 5*1024*1024 {
		t.Errorf("Expected total size %d, got: %d", 5*1024*1024, session.TotalSize)
	}

	if session.MIMEType != "image/jpeg" {
		t.Errorf("Expected MIME type 'image/jpeg', got: %s", session.MIMEType)
	}

	if session.Status != "uploading" {
		t.Errorf("Expected status 'uploading', got: %s", session.Status)
	}

	if session.TotalChunks != 5 {
		t.Errorf("Expected 5 chunks, got: %d", session.TotalChunks)
	}

	if session.FileID == "" {
		t.Error("Expected file ID to be generated")
	}
}

func TestInitiateUploadValidation(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	options := Options{
		MaxFileSize:      10 * 1024 * 1024, // 10MB
		AllowedMIMETypes: []string{"image/jpeg", "image/png"},
	}

	processor := NewResumableProcessor(mockStorage, options)
	ctx := context.Background()

	// Test file too large
	_, err := processor.InitiateUpload(ctx, "test.jpg", 20*1024*1024, "image/jpeg", 1024*1024)
	if err == nil {
		t.Error("Expected error for file too large")
	}

	// Test invalid MIME type
	_, err = processor.InitiateUpload(ctx, "test.txt", 1024*1024, "text/plain", 1024*1024)
	if err == nil {
		t.Error("Expected error for invalid MIME type")
	}
}

func TestUploadChunk(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	options := Options{
		MaxFileSize:      10 * 1024 * 1024,
		AllowedMIMETypes: []string{"image/jpeg"},
	}

	processor := NewResumableProcessor(mockStorage, options)
	ctx := context.Background()

	// Initiate upload
	session, err := processor.InitiateUpload(ctx, "test.jpg", 3*1024*1024, "image/jpeg", 1024*1024)
	if err != nil {
		t.Fatalf("Failed to initiate upload: %v", err)
	}

	// Upload first chunk
	chunkData := strings.NewReader("chunk data")
	err = processor.UploadChunk(ctx, session.FileID, 0, chunkData)
	if err != nil {
		t.Fatalf("Failed to upload chunk: %v", err)
	}

	// Check session status
	updatedSession, err := processor.GetStatus(session.FileID)
	if err != nil {
		t.Fatalf("Failed to get upload status: %v", err)
	}

	if len(updatedSession.Chunks) != 1 {
		t.Errorf("Expected 1 chunk, got: %d", len(updatedSession.Chunks))
	}

	if _, exists := updatedSession.Chunks[0]; !exists {
		t.Error("Expected chunk 0 to exist")
	}
}

func TestUploadChunkValidation(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	options := Options{
		MaxFileSize:      10 * 1024 * 1024,
		AllowedMIMETypes: []string{"image/jpeg"},
	}

	processor := NewResumableProcessor(mockStorage, options)
	ctx := context.Background()

	// Initiate upload
	session, err := processor.InitiateUpload(ctx, "test.jpg", 3*1024*1024, "image/jpeg", 1024*1024)
	if err != nil {
		t.Fatalf("Failed to initiate upload: %v", err)
	}

	// Test invalid file ID
	chunkData := strings.NewReader("chunk data")
	err = processor.UploadChunk(ctx, "invalid-id", 0, chunkData)
	if err == nil {
		t.Error("Expected error for invalid file ID")
	}

	// Test invalid chunk number
	err = processor.UploadChunk(ctx, session.FileID, 10, chunkData)
	if err == nil {
		t.Error("Expected error for invalid chunk number")
	}

	// Test negative chunk number
	err = processor.UploadChunk(ctx, session.FileID, -1, chunkData)
	if err == nil {
		t.Error("Expected error for negative chunk number")
	}
}

func TestCompleteUpload(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	options := Options{
		MaxFileSize:      10 * 1024 * 1024,
		AllowedMIMETypes: []string{"image/jpeg"},
	}

	processor := NewResumableProcessor(mockStorage, options)
	ctx := context.Background()

	// Initiate upload
	session, err := processor.InitiateUpload(ctx, "test.jpg", 2*1024*1024, "image/jpeg", 1024*1024)
	if err != nil {
		t.Fatalf("Failed to initiate upload: %v", err)
	}

	// Upload all chunks
	for i := 0; i < session.TotalChunks; i++ {
		chunkData := strings.NewReader(fmt.Sprintf("chunk %d data", i))
		err = processor.UploadChunk(ctx, session.FileID, i, chunkData)
		if err != nil {
			t.Fatalf("Failed to upload chunk %d: %v", i, err)
		}
	}

	// Complete upload
	result, err := processor.CompleteUpload(ctx, session.FileID)
	if err != nil {
		t.Fatalf("Failed to complete upload: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be returned")
	}

	if result.OriginalName != "test.jpg" {
		t.Errorf("Expected original name 'test.jpg', got: %s", result.OriginalName)
	}

	if result.Size != 2*1024*1024 {
		t.Errorf("Expected size %d, got: %d", 2*1024*1024, result.Size)
	}

	if result.MIMEType != "image/jpeg" {
		t.Errorf("Expected MIME type 'image/jpeg', got: %s", result.MIMEType)
	}

	// Check session status
	updatedSession, err := processor.GetStatus(session.FileID)
	if err != nil {
		t.Fatalf("Failed to get upload status: %v", err)
	}

	if updatedSession.Status != "completed" {
		t.Errorf("Expected status 'completed', got: %s", updatedSession.Status)
	}
}

func TestCompleteUploadIncomplete(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	options := Options{
		MaxFileSize:      10 * 1024 * 1024,
		AllowedMIMETypes: []string{"image/jpeg"},
	}

	processor := NewResumableProcessor(mockStorage, options)
	ctx := context.Background()

	// Initiate upload
	session, err := processor.InitiateUpload(ctx, "test.jpg", 3*1024*1024, "image/jpeg", 1024*1024)
	if err != nil {
		t.Fatalf("Failed to initiate upload: %v", err)
	}

	// Upload only first chunk
	chunkData := strings.NewReader("chunk data")
	err = processor.UploadChunk(ctx, session.FileID, 0, chunkData)
	if err != nil {
		t.Fatalf("Failed to upload chunk: %v", err)
	}

	// Try to complete upload
	_, err = processor.CompleteUpload(ctx, session.FileID)
	if err == nil {
		t.Error("Expected error for incomplete upload")
	}
}

func TestAbortUpload(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	options := Options{
		MaxFileSize:      10 * 1024 * 1024,
		AllowedMIMETypes: []string{"image/jpeg"},
	}

	processor := NewResumableProcessor(mockStorage, options)
	ctx := context.Background()

	// Initiate upload
	session, err := processor.InitiateUpload(ctx, "test.jpg", 1024*1024, "image/jpeg", 1024*1024)
	if err != nil {
		t.Fatalf("Failed to initiate upload: %v", err)
	}

	// Abort upload
	err = processor.AbortUpload(session.FileID)
	if err != nil {
		t.Fatalf("Failed to abort upload: %v", err)
	}

	// Try to get status
	_, err = processor.GetStatus(session.FileID)
	if err == nil {
		t.Error("Expected error for aborted upload")
	}
}

func TestHandleResumableUploadInitiate(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	options := Options{
		MaxFileSize:      10 * 1024 * 1024,
		AllowedMIMETypes: []string{"image/jpeg"},
	}

	processor := NewResumableProcessor(mockStorage, options)

	// Create request body
	reqBody := map[string]interface{}{
		"file_name":  "test.jpg",
		"total_size": 1024 * 1024,
		"mime_type":  "image/jpeg",
		"chunk_size": 512 * 1024,
	}

	jsonBody, _ := json.Marshal(reqBody)

	// Create request
	req := httptest.NewRequest("POST", "/upload", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Handle request
	processor.HandleResumableUpload(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", w.Code)
	}

	var session Session
	err := json.NewDecoder(w.Body).Decode(&session)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if session.FileName != "test.jpg" {
		t.Errorf("Expected filename 'test.jpg', got: %s", session.FileName)
	}
}

func TestHandleResumableUploadChunk(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	options := Options{
		MaxFileSize:      10 * 1024 * 1024,
		AllowedMIMETypes: []string{"image/jpeg"},
	}

	processor := NewResumableProcessor(mockStorage, options)
	ctx := context.Background()

	// Initiate upload
	session, err := processor.InitiateUpload(ctx, "test.jpg", 1024*1024, "image/jpeg", 512*1024)
	if err != nil {
		t.Fatalf("Failed to initiate upload: %v", err)
	}

	// Create request
	chunkData := strings.NewReader("chunk data")
	req := httptest.NewRequest("PUT", fmt.Sprintf("/upload?file_id=%s&chunk_number=0", session.FileID), chunkData)
	w := httptest.NewRecorder()

	// Handle request
	processor.HandleResumableUpload(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", w.Code)
	}
}

func TestHandleResumableUploadStatus(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	options := Options{
		MaxFileSize:      10 * 1024 * 1024,
		AllowedMIMETypes: []string{"image/jpeg"},
	}

	processor := NewResumableProcessor(mockStorage, options)
	ctx := context.Background()

	// Initiate upload
	session, err := processor.InitiateUpload(ctx, "test.jpg", 1024*1024, "image/jpeg", 512*1024)
	if err != nil {
		t.Fatalf("Failed to initiate upload: %v", err)
	}

	// Create request
	req := httptest.NewRequest("GET", fmt.Sprintf("/upload?file_id=%s", session.FileID), nil)
	w := httptest.NewRecorder()

	// Handle request
	processor.HandleResumableUpload(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", w.Code)
	}

	var responseSession Session
	err = json.NewDecoder(w.Body).Decode(&responseSession)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if responseSession.FileID != session.FileID {
		t.Errorf("Expected file ID %s, got: %s", session.FileID, responseSession.FileID)
	}
}

func TestHandleResumableUploadAbort(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	options := Options{
		MaxFileSize:      10 * 1024 * 1024,
		AllowedMIMETypes: []string{"image/jpeg"},
	}

	processor := NewResumableProcessor(mockStorage, options)
	ctx := context.Background()

	// Initiate upload
	session, err := processor.InitiateUpload(ctx, "test.jpg", 1024*1024, "image/jpeg", 512*1024)
	if err != nil {
		t.Fatalf("Failed to initiate upload: %v", err)
	}

	// Create request
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/upload?file_id=%s", session.FileID), nil)
	w := httptest.NewRecorder()

	// Handle request
	processor.HandleResumableUpload(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", w.Code)
	}
}

func TestConcurrentUploads(t *testing.T) {
	mockStorage := storage.NewMockStorage()
	options := Options{
		MaxFileSize:      10 * 1024 * 1024,
		AllowedMIMETypes: []string{"image/jpeg"},
	}

	processor := NewResumableProcessor(mockStorage, options)
	ctx := context.Background()

	// Create multiple upload sessions
	sessions := make([]*Session, 5)
	for i := 0; i < 5; i++ {
		session, err := processor.InitiateUpload(ctx, fmt.Sprintf("test%d.jpg", i), 1024*1024, "image/jpeg", 512*1024)
		if err != nil {
			t.Fatalf("Failed to initiate upload %d: %v", i, err)
		}
		sessions[i] = session
	}

	// Upload chunks concurrently
	done := make(chan bool, 5)
	for i, session := range sessions {
		go func(session *Session, index int) {
			chunkData := strings.NewReader(fmt.Sprintf("chunk data for file %d", index))
			err := processor.UploadChunk(ctx, session.FileID, 0, chunkData)
			if err != nil {
				t.Errorf("Failed to upload chunk for file %d: %v", index, err)
			}
			done <- true
		}(session, i)
	}

	// Wait for all uploads to complete
	for i := 0; i < 5; i++ {
		<-done
	}

	// Verify all sessions have chunks
	for i, session := range sessions {
		status, err := processor.GetStatus(session.FileID)
		if err != nil {
			t.Errorf("Failed to get status for file %d: %v", i, err)
		}
		if len(status.Chunks) != 1 {
			t.Errorf("Expected 1 chunk for file %d, got: %d", i, len(status.Chunks))
		}
	}
}
