// Package storage provides a unified interface for file storage backends in the upload package.
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// GCSConfig holds configuration for GCS storage backend.
type GCSConfig struct {
	Bucket          string // GCS bucket name (required)
	ProjectID       string // GCP project ID (optional, for some operations)
	CredentialsFile string // Path to service account JSON file (optional, uses env if empty)
	BaseURL         string // Custom base URL for public access (optional)
}

// GCSStorage implements the Storage interface for Google Cloud Storage.
type GCSStorage struct {
	client  *storage.Client
	bucket  string
	baseURL string
}

// NewGCS creates a new GCS storage instance with the specified configuration.
func NewGCS(config GCSConfig) (Storage, error) {
	if config.Bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}
	ctx := context.Background()
	var client *storage.Client
	var err error
	if config.CredentialsFile != "" {
		if err := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", config.CredentialsFile); err != nil {
			return nil, fmt.Errorf("failed to set GOOGLE_APPLICATION_CREDENTIALS: %v", err)
		}
	}
	client, err = storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %v", err)
	}
	return &GCSStorage{
		client:  client,
		bucket:  config.Bucket,
		baseURL: config.BaseURL,
	}, nil
}

// Store saves a file to GCS.
func (g *GCSStorage) Store(filename string, reader io.Reader) (string, error) {
	ctx := context.Background()
	key := strings.TrimPrefix(filepath.Clean(filename), "/")
	w := g.client.Bucket(g.bucket).Object(key).NewWriter(ctx)
	if _, err := io.Copy(w, reader); err != nil {
		if closeErr := w.Close(); closeErr != nil {
			return "", fmt.Errorf("failed to write to GCS: %v, and failed to close writer: %v", err, closeErr)
		}
		return "", fmt.Errorf("failed to write to GCS: %v", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("failed to close GCS writer: %v", err)
	}
	return key, nil
}

// GetURL returns the public URL for accessing a stored file.
func (g *GCSStorage) GetURL(path string) string {
	if g.baseURL != "" {
		base := strings.TrimSuffix(g.baseURL, "/")
		return fmt.Sprintf("%s/%s", base, path)
	}
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", g.bucket, path)
}

// Delete removes a file from GCS.
func (g *GCSStorage) Delete(filename string) error {
	ctx := context.Background()
	return g.client.Bucket(g.bucket).Object(filename).Delete(ctx)
}

// Exists checks if a file exists in GCS.
func (g *GCSStorage) Exists(filename string) bool {
	ctx := context.Background()
	_, err := g.client.Bucket(g.bucket).Object(filename).Attrs(ctx)
	return err == nil
}

// GetSize returns the size of a stored file in bytes.
func (g *GCSStorage) GetSize(filename string) (int64, error) {
	ctx := context.Background()
	attrs, err := g.client.Bucket(g.bucket).Object(filename).Attrs(ctx)
	if err != nil {
		return 0, err
	}
	return attrs.Size, nil
}

// ListFiles returns a list of all files stored in GCS.
func (g *GCSStorage) ListFiles() ([]string, error) {
	ctx := context.Background()
	var files []string
	it := g.client.Bucket(g.bucket).Objects(ctx, nil)
	for {
		attrs, err := it.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, err
		}
		files = append(files, attrs.Name)
	}
	return files, nil
}

// GetSignedURL generates a pre-signed URL for temporary access to a file.
func (g *GCSStorage) GetSignedURL(filename string, expiration time.Duration) (string, error) {
	// This requires a service account with signBlob permission
	// For simplicity, we use storage.SignedURL (which requires a private key in the credentials file)
	opts := &storage.SignedURLOptions{
		GoogleAccessID: os.Getenv("GOOGLE_ACCESS_ID"),
		PrivateKey:     []byte(os.Getenv("GOOGLE_PRIVATE_KEY")),
		Method:         "GET",
		Expires:        time.Now().Add(expiration),
	}
	return storage.SignedURL(g.bucket, filename, opts)
}

// GetReader returns an io.ReadCloser for reading a stored file from GCS.
func (g *GCSStorage) GetReader(filename string) (io.ReadCloser, error) {
	ctx := context.Background()
	obj := g.client.Bucket(g.bucket).Object(filename)
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get object reader: %v", err)
	}

	return reader, nil
}

// GetBucketInfo returns metadata about the GCS bucket.
func (g *GCSStorage) GetBucketInfo() (map[string]interface{}, error) {
	ctx := context.Background()
	info := map[string]interface{}{
		"type":   "gcs",
		"bucket": g.bucket,
	}
	attrs, err := g.client.Bucket(g.bucket).Attrs(ctx)
	if err == nil {
		info["location"] = attrs.Location
		info["storageClass"] = attrs.StorageClass
		info["created"] = attrs.Created
	}
	// Count files and total size
	var totalSize int64
	var fileCount int
	it := g.client.Bucket(g.bucket).Objects(ctx, nil)
	for {
		attrs, err := it.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return info, err
		}
		totalSize += attrs.Size
		fileCount++
	}
	info["totalSize"] = totalSize
	info["fileCount"] = fileCount
	return info, nil
}

// Close performs cleanup operations for the GCS storage backend.
func (g *GCSStorage) Close() error {
	return g.client.Close()
}
