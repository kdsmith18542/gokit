// Package storage provides a unified interface for file storage backends in the upload package.
package storage

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
)

// AzureConfig holds configuration for Azure Blob Storage backend.
type AzureConfig struct {
	AccountName string // Azure storage account name (required)
	AccountKey  string // Azure storage account key (required)
	Container   string // Blob container name (required)
	BaseURL     string // Custom base URL for public access (optional)
}

// AzureBlobStorage implements the Storage interface for Azure Blob Storage.
type AzureBlobStorage struct {
	client      *azblob.Client
	accountName string
	container   string
	baseURL     string
	keyCred     *azblob.SharedKeyCredential
}

// NewAzureBlob creates a new Azure Blob storage instance with the specified configuration.
func NewAzureBlob(config AzureConfig) (Storage, error) {
	if config.AccountName == "" || config.AccountKey == "" || config.Container == "" {
		return nil, fmt.Errorf("account name, account key, and container are required")
	}
	cred, err := azblob.NewSharedKeyCredential(config.AccountName, config.AccountKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credentials: %v", err)
	}
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", config.AccountName)
	client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure client: %v", err)
	}
	return &AzureBlobStorage{
		client:      client,
		accountName: config.AccountName,
		container:   config.Container,
		baseURL:     config.BaseURL,
		keyCred:     cred,
	}, nil
}

// Store saves a file to Azure Blob Storage.
func (a *AzureBlobStorage) Store(filename string, reader io.Reader) (string, error) {
	ctx := context.Background()
	key := strings.TrimPrefix(filepath.Clean(filename), "/")
	_, err := a.client.UploadStream(ctx, a.container, key, reader, nil)
	if err != nil {
		return "", fmt.Errorf("failed to upload to Azure Blob: %v", err)
	}
	return key, nil
}

// GetURL returns the public URL for accessing a stored file.
func (a *AzureBlobStorage) GetURL(path string) string {
	if a.baseURL != "" {
		base := strings.TrimSuffix(a.baseURL, "/")
		return fmt.Sprintf("%s/%s", base, path)
	}
	return fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", a.accountName, a.container, path)
}

// Delete removes a file from Azure Blob Storage.
func (a *AzureBlobStorage) Delete(filename string) error {
	ctx := context.Background()
	_, err := a.client.DeleteBlob(ctx, a.container, filename, nil)
	return err
}

// Exists checks if a file exists in Azure Blob Storage.
func (a *AzureBlobStorage) Exists(filename string) bool {
	ctx := context.Background()
	blobClient := a.client.ServiceClient().NewContainerClient(a.container).NewBlobClient(filename)
	_, err := blobClient.GetProperties(ctx, nil)
	return err == nil
}

// GetSize returns the size of a stored file in bytes.
func (a *AzureBlobStorage) GetSize(filename string) (int64, error) {
	ctx := context.Background()
	blobClient := a.client.ServiceClient().NewContainerClient(a.container).NewBlobClient(filename)
	props, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		return 0, err
	}
	return *props.ContentLength, nil
}

// ListFiles returns a list of all files stored in Azure Blob Storage.
func (a *AzureBlobStorage) ListFiles() ([]string, error) {
	ctx := context.Background()
	var files []string
	pager := a.client.NewListBlobsFlatPager(a.container, nil)
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, blob := range resp.Segment.BlobItems {
			files = append(files, *blob.Name)
		}
	}
	return files, nil
}

// GetSignedURL generates a pre-signed URL for temporary access to a file.
func (a *AzureBlobStorage) GetSignedURL(filename string, expiration time.Duration) (string, error) {
	if a.keyCred == nil {
		return "", fmt.Errorf("shared key credential required for SAS URL")
	}
	sasValues := sas.BlobSignatureValues{
		ContainerName: a.container,
		BlobName:      filename,
		Permissions:   "r",
		StartTime:     time.Now().Add(-5 * time.Minute),
		ExpiryTime:    time.Now().Add(expiration),
	}
	q, err := sasValues.SignWithSharedKey(a.keyCred)
	if err != nil {
		return "", fmt.Errorf("failed to generate SAS token: %v", err)
	}
	urlStr := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s?%s", a.accountName, a.container, filename, q.Encode())
	return urlStr, nil
}

// GetReader returns an io.ReadCloser for reading a stored file from Azure Blob Storage.
func (a *AzureBlobStorage) GetReader(filename string) (io.ReadCloser, error) {
	ctx := context.Background()
	blobClient := a.client.ServiceClient().NewContainerClient(a.container).NewBlobClient(filename)

	response, err := blobClient.DownloadStream(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download blob: %v", err)
	}

	return response.Body, nil
}

// GetBucketInfo returns metadata about the Azure Blob container.
func (a *AzureBlobStorage) GetBucketInfo() (map[string]interface{}, error) {
	ctx := context.Background()
	info := map[string]interface{}{
		"type":      "azure",
		"account":   a.accountName,
		"container": a.container,
	}
	// Count files and total size
	var totalSize int64
	var fileCount int
	pager := a.client.NewListBlobsFlatPager(a.container, nil)
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			break
		}
		for _, blob := range resp.Segment.BlobItems {
			if blob.Properties.ContentLength != nil {
				totalSize += *blob.Properties.ContentLength
			}
			fileCount++
		}
	}
	info["totalSize"] = totalSize
	info["fileCount"] = fileCount
	return info, nil
}

// Close performs cleanup operations for the Azure Blob storage backend.
func (a *AzureBlobStorage) Close() error {
	// No explicit close needed for Azure SDK
	return nil
}
