// Package storage provides a unified interface for file storage backends in the upload package.
package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalStorage implements the Storage interface for local file system storage.
// It provides a simple, efficient way to store files on the local disk with
// configurable base paths and URL generation.
type LocalStorage struct {
	basePath string
	baseURL  string
}

// NewLocal creates a new LocalStorage backend with the given base path and optional base URL.
func NewLocal(basePath string, baseURL ...string) Storage {
	ls := &LocalStorage{basePath: basePath}
	if len(baseURL) > 0 {
		ls.baseURL = baseURL[0]
	}
	return ls
}

// NewLocalWithURL creates a new local storage instance with the specified base path and URL.
// The base path is the directory where all files will be stored.
// The base URL is used to generate public URLs for accessing the files.
//
// Example:
//
//	storage := storage.NewLocalWithURL("./uploads", "/uploads/")
//	defer storage.Close()
func NewLocalWithURL(basePath, baseURL string) Storage {
	return &LocalStorage{
		basePath: basePath,
		baseURL:  baseURL,
	}
}

// Store saves a file to the local storage backend.
// The filename parameter is the desired name for the stored file.
// The reader provides the file content to be stored.
// Returns the internal path/identifier for the stored file.
func (l *LocalStorage) Store(filename string, reader io.Reader) (string, error) {
	// Filename validation
	if filename == "" {
		return "", fmt.Errorf("filename cannot be empty")
	}
	if reader == nil {
		return "", fmt.Errorf("reader cannot be nil")
	}
	if strings.Contains(filename, "\x00") {
		return "", fmt.Errorf("filename contains null byte")
	}
	for _, r := range filename {
		if r < 32 {
			return "", fmt.Errorf("filename contains control character: %q", r)
		}
	}
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return "", fmt.Errorf("filename contains path traversal or separator")
	}

	// Ensure the base directory exists
	if err := os.MkdirAll(l.basePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create base directory: %v", err)
	}

	// Create the full file path
	filePath := filepath.Join(l.basePath, filename)

	// Check for symlink
	if fi, err := os.Lstat(filePath); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("refusing to write to symlink: %s", filename)
	}

	// Ensure the directory for this file exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create file directory: %v", err)
	}

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// Copy the content from the reader to the file
	_, err = io.Copy(file, reader)
	if err != nil {
		return "", fmt.Errorf("failed to write file content: %v", err)
	}

	// Return the relative path from the base path
	relPath, err := filepath.Rel(l.basePath, filePath)
	if err != nil {
		return filename, nil // Fallback to original filename
	}

	return relPath, nil
}

// GetURL returns the public URL for accessing a stored file.
// The path parameter is the internal path returned by Store().
// Returns an empty string if the storage backend doesn't support public URLs.
func (l *LocalStorage) GetURL(path string) string {
	if l.baseURL == "" {
		return ""
	}

	// Ensure the base URL ends with a slash
	baseURL := l.baseURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	// Return the full URL
	return baseURL + path
}

// Delete removes a file from the local storage backend.
// The filename parameter should match the internal path returned by Store().
// Returns an error if the file doesn't exist or cannot be deleted.
func (l *LocalStorage) Delete(filename string) error {
	filePath := filepath.Join(l.basePath, filename)

	// Check if file exists
	if !l.Exists(filename) {
		return fmt.Errorf("file does not exist: %s", filename)
	}

	// Delete the file
	err := os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("failed to delete file: %v", err)
	}

	return nil
}

// Exists checks if a file exists in the local storage backend.
// The filename parameter should match the internal path returned by Store().
// Returns true if the file exists, false otherwise.
func (l *LocalStorage) Exists(filename string) bool {
	filePath := filepath.Join(l.basePath, filename)
	_, err := os.Stat(filePath)
	return err == nil
}

// GetSize returns the size of a stored file in bytes.
// The filename parameter should match the internal path returned by Store().
// Returns an error if the file doesn't exist or size cannot be determined.
func (l *LocalStorage) GetSize(filename string) (int64, error) {
	filePath := filepath.Join(l.basePath, filename)

	// Check if file exists
	if !l.Exists(filename) {
		return 0, fmt.Errorf("file does not exist: %s", filename)
	}

	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to get file info: %v", err)
	}

	return info.Size(), nil
}

// ListFiles returns a list of all files stored in the backend.
// Returns file paths/identifiers that can be used with other methods.
// The exact format of returned paths depends on the storage backend.
func (l *LocalStorage) ListFiles() ([]string, error) {
	var files []string

	// Walk through the base directory
	err := filepath.Walk(l.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path from base path
		relPath, err := filepath.Rel(l.basePath, path)
		if err != nil {
			return err
		}

		// Add to files list
		files = append(files, relPath)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list files: %v", err)
	}

	return files, nil
}

// GetSignedURL generates a pre-signed URL for temporary access to a file.
// The filename parameter should match the internal path returned by Store().
// The expiration parameter specifies how long the URL should be valid.
// Returns an error if the storage backend doesn't support signed URLs.
func (l *LocalStorage) GetSignedURL(filename string, expiration time.Duration) (string, error) {
	// Local storage doesn't support signed URLs, so we return the regular URL
	// with a note that it's not actually signed
	url := l.GetURL(filename)
	if url == "" {
		return "", fmt.Errorf("signed URLs not supported for local storage")
	}

	// For local storage, we just return the regular URL
	// In a real implementation, you might want to add some form of access control
	return url, nil
}

func (l *LocalStorage) GetReader(filename string) (io.ReadCloser, error) {
	if filename == "" {
		return nil, fmt.Errorf("filename cannot be empty")
	}

	// Create the full file path
	filePath := filepath.Join(l.basePath, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", filename)
	}

	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}

	return file, nil
}

// GetBucketInfo returns metadata about the storage backend.
// The returned map contains backend-specific information such as
// bucket name, region, total size, file count, etc.
// Returns an error if the information cannot be retrieved.
func (l *LocalStorage) GetBucketInfo() (map[string]interface{}, error) {
	info := map[string]interface{}{
		"type":     "local",
		"basePath": l.basePath,
		"baseURL":  l.baseURL,
	}

	// Get total size and file count
	var totalSize int64
	var fileCount int

	err := filepath.Walk(l.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			totalSize += info.Size()
			fileCount++
		}

		return nil
	})

	if err != nil {
		return info, fmt.Errorf("failed to calculate bucket info: %v", err)
	}

	info["totalSize"] = totalSize
	info["fileCount"] = fileCount

	return info, nil
}

// Close performs cleanup operations for the storage backend.
// Should be called when the storage instance is no longer needed.
// Returns an error if cleanup fails.
func (l *LocalStorage) Close() error {
	// Local storage doesn't require any cleanup
	return nil
}
