package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
)

// AzureStorage implements Storage interface using Azure Blob Storage
type AzureStorage struct {
	client          *azblob.Client
	containerClient *container.Client // Direct container client for SAS URL case
	container       string
	useSASContainer bool // Flag to indicate if we should use containerClient directly
}

// AzureStorageOptions holds configuration for Azure Blob Storage
type AzureStorageOptions struct {
	// Storage account name (optional if using SAS URL)
	AccountName string

	// Container name where tiles will be stored
	Container string

	// SAS URL for authentication (recommended)
	SASURL string

	// Account key for authentication (less secure than SAS)
	AccountKey string
}

// NewAzureStorage creates a new Azure Blob Storage instance
func NewAzureStorage(ctx context.Context, options AzureStorageOptions) (*AzureStorage, error) {
	var client *azblob.Client
	var err error

	// Create client based on authentication method
	if options.SASURL != "" {
		// Use SAS URL (recommended for security)
		// Check if it's a full connection string or just a SAS token
		if strings.Contains(options.SASURL, "BlobEndpoint=") {
			// Full connection string
			client, err = azblob.NewClientFromConnectionString(options.SASURL, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create Azure client from connection string: %w", err)
			}
		} else {
			// SAS URL points to container level - create container client directly
			// This avoids issues with the SDK trying to append container name again
			containerClient, err := container.NewClientWithNoCredential(options.SASURL, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create Azure container client from SAS URL: %w", err)
			}

			// Verify container is accessible
			_, err = containerClient.GetProperties(ctx, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to access container '%s': %w (ensure container exists and SAS token is valid)", options.Container, err)
			}

			return &AzureStorage{
				containerClient: containerClient,
				container:       options.Container,
				useSASContainer: true,
			}, nil
		}
	} else if options.AccountName != "" && options.AccountKey != "" {
		// Use account name + key
		credential, err := azblob.NewSharedKeyCredential(options.AccountName, options.AccountKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create shared key credential: %w", err)
		}

		serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", options.AccountName)
		client, err = azblob.NewClientWithSharedKeyCredential(serviceURL, credential, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure client with shared key: %w", err)
		}
	} else {
		return nil, fmt.Errorf("either sas_url or (account_name + account_key) is required")
	}

	// Verify container exists by attempting to get its properties
	containerClient := client.ServiceClient().NewContainerClient(options.Container)
	_, err = containerClient.GetProperties(ctx, nil)
	if err != nil {
		// If container doesn't exist, provide a helpful error message
		return nil, fmt.Errorf("failed to access container '%s': %w (ensure container exists and credentials are valid)", options.Container, err)
	}

	return &AzureStorage{
		client:    client,
		container: options.Container,
	}, nil
}

// getContainerClient returns the appropriate container client
func (s *AzureStorage) getContainerClient() *container.Client {
	if s.useSASContainer {
		return s.containerClient
	}
	return s.client.ServiceClient().NewContainerClient(s.container)
}

// getBlockBlobClient returns a block blob client for the given key
func (s *AzureStorage) getBlockBlobClient(key string) *blockblob.Client {
	return s.getContainerClient().NewBlockBlobClient(key)
}

// getBlobClient returns a blob client for the given key
func (s *AzureStorage) getBlobClient(key string) *blob.Client {
	return s.getContainerClient().NewBlobClient(key)
}

// Get retrieves data by key
// Returns nil if key does not exist
func (s *AzureStorage) Get(key string) ([]byte, error) {
	ctx := context.Background()
	blobClient := s.getBlobClient(key)

	// Download blob
	resp, err := blobClient.DownloadStream(ctx, nil)
	if err != nil {
		// Check if blob doesn't exist
		if isNotFoundError(err) {
			return nil, nil // Return nil for not found, not an error
		}
		return nil, fmt.Errorf("failed to download blob for key %s: %w", key, err)
	}
	defer resp.Body.Close()

	// Read entire blob into memory
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read blob data for key %s: %w", key, err)
	}

	return data, nil
}

// Put stores data at the specified key
func (s *AzureStorage) Put(key string, data []byte) error {
	ctx := context.Background()
	blockBlobClient := s.getBlockBlobClient(key)

	// Upload blob using UploadBuffer (handles byte slices directly)
	_, err := blockBlobClient.UploadBuffer(ctx, data, &blockblob.UploadBufferOptions{
		HTTPHeaders: &blob.HTTPHeaders{
			BlobContentType: to.Ptr("application/octet-stream"),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to upload blob for key %s: %w", key, err)
	}

	return nil
}

// Delete removes data at the specified key
func (s *AzureStorage) Delete(key string) error {
	ctx := context.Background()
	blobClient := s.getBlobClient(key)

	_, err := blobClient.Delete(ctx, nil)
	if err != nil && !isNotFoundError(err) {
		return fmt.Errorf("failed to delete blob for key %s: %w", key, err)
	}

	return nil
}

// Exists checks if a key exists
func (s *AzureStorage) Exists(key string) (bool, error) {
	ctx := context.Background()
	blobClient := s.getBlobClient(key)

	_, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		if isNotFoundError(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check blob existence for key %s: %w", key, err)
	}

	return true, nil
}

// List returns all keys with the given prefix
func (s *AzureStorage) List(prefix string) ([]string, error) {
	ctx := context.Background()
	containerClient := s.getContainerClient()

	var keys []string

	// List blobs with prefix
	pager := containerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Prefix: &prefix,
	})

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list blobs with prefix %s: %w", prefix, err)
		}

		for _, blobItem := range resp.Segment.BlobItems {
			if blobItem.Name != nil {
				keys = append(keys, *blobItem.Name)
			}
		}
	}

	return keys, nil
}

// Clear removes all blobs from the container (for development/testing)
func (s *AzureStorage) Clear() error {
	// List all blobs (no prefix filter)
	keys, err := s.List("")
	if err != nil {
		return fmt.Errorf("failed to list blobs for clearing: %w", err)
	}

	// Delete each blob
	for _, key := range keys {
		if err := s.Delete(key); err != nil {
			return fmt.Errorf("failed to delete blob %s during clear: %w", key, err)
		}
	}

	return nil
}

// isNotFoundError checks if an error indicates a blob was not found
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// Azure SDK returns errors with "BlobNotFound" or "404" in the message
	errMsg := err.Error()
	return strings.Contains(errMsg, "BlobNotFound") ||
		strings.Contains(errMsg, "404") ||
		strings.Contains(errMsg, "ContainerNotFound")
}

// String returns a debug string representation
func (s *AzureStorage) String() string {
	return fmt.Sprintf("AzureStorage{container: %s}", s.container)
}
