package storage

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLocalStorage_StoreAndGet(t *testing.T) {
	dir := t.TempDir()
	storage := NewLocal(dir)

	// Store a file
	content := "Hello, Local!"
	filename := "test.txt"
	_, err := storage.Store(filename, strings.NewReader(content))
	if err != nil {
		t.Fatalf("Failed to store file: %v", err)
	}

	// Check file exists
	if !storage.Exists(filename) {
		t.Error("File should exist after storing")
	}

	// Get file size
	size, err := storage.GetSize(filename)
	if err != nil {
		t.Errorf("GetSize failed: %v", err)
	}
	if size != int64(len(content)) {
		t.Errorf("Expected size %d, got %d", len(content), size)
	}

	// List files
	files, err := storage.ListFiles()
	if err != nil {
		t.Errorf("ListFiles failed: %v", err)
	}
	found := false
	for _, f := range files {
		if f == filename {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected file '%s' in list", filename)
	}

	// Delete file
	err = storage.Delete(filename)
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}
	if storage.Exists(filename) {
		t.Error("File should not exist after deletion")
	}
}

func TestLocalStorage_InvalidPath(t *testing.T) {
	// Use a null byte in the path which is invalid on all platforms
	storage := NewLocal("/invalid\x00path")
	_, err := storage.Store("file.txt", strings.NewReader("data"))
	if err == nil {
		t.Error("Expected error for invalid storage path")
	}
}

func TestLocalStorage_PermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("File permissions do not prevent writes on Windows")
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "readonly.txt")
	if err := os.WriteFile(file, []byte("data"), 0400); err != nil {
		t.Fatalf("Failed to create readonly file for test: %v", err)
	}
	storage := NewLocal(dir)
	// Try to overwrite a readonly file
	_, err := storage.Store("readonly.txt", strings.NewReader("newdata"))
	if err == nil {
		t.Error("Expected error for writing to readonly file")
	}
}

func TestLocalStorage_EmptyFilename(t *testing.T) {
	dir := t.TempDir()
	storage := NewLocal(dir)
	_, err := storage.Store("", strings.NewReader("data"))
	if err == nil {
		t.Error("Expected error for empty filename")
	}
}

func TestLocalStorage_NilReader(t *testing.T) {
	dir := t.TempDir()
	storage := NewLocal(dir)
	_, err := storage.Store("file.txt", nil)
	if err == nil {
		t.Error("Expected error for nil reader")
	}
}
