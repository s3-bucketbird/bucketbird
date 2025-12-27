package service

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"path"
	"sort"
	"strings"
	"time"

	"bucketbird/backend/internal/storage"

	"github.com/google/uuid"
)

// BucketObject represents an object in a bucket
type BucketObject struct {
	Key          string    `json:"key"`
	Name         string    `json:"name"`
	Kind         string    `json:"kind"`
	Size         string    `json:"size"`
	LastModified time.Time `json:"lastModified"`
	Icon         string    `json:"icon"`
	IconColor    string    `json:"iconColor"`
}

// PresignInput contains input for presigning a URL
type PresignInput struct {
	Key         string
	Method      string
	Expires     time.Duration
	ContentType *string
}

// PresignOutput contains presigned URL information
type PresignOutput struct {
	URL     string `json:"url"`
	Expires int64  `json:"expires"`
}

// ObjectMetadata contains object metadata
type ObjectMetadata struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	LastModified time.Time         `json:"lastModified"`
	ContentType  string            `json:"contentType"`
	ETag         string            `json:"etag"`
	Metadata     map[string]string `json:"metadata"`
}

// ProxiedObject represents an object being proxied
type ProxiedObject struct {
	Body          io.ReadCloser
	ContentType   string
	ContentLength int64
}

// DeleteObjectsResult contains the result of bulk delete
type DeleteObjectsResult struct {
	Deleted []string `json:"deleted"`
	Failed  []string `json:"failed"`
}

// OperationResult represents a generic operation result
type OperationResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// FolderResult represents a created folder
type FolderResult struct {
	Key string `json:"key"`
}

// ListObjects lists objects in a bucket with optional prefix
func (s *BucketService) ListObjects(ctx context.Context, bucketID, userID uuid.UUID, prefix string, encryptionKey []byte) ([]BucketObject, error) {
	// Check if user is a demo user FIRST
	user, err := s.users.GetByID(ctx, userID)
	if err == nil && user.IsDemo {
		// For demo users, get bucket name and return static demo data
		bucketName, err := s.getBucketName(ctx, bucketID, userID)
		if err != nil {
			return nil, err
		}
		return getDemoObjects(bucketName, prefix), nil
	}

	// For regular users, proceed with normal flow
	bucketName, err := s.getBucketName(ctx, bucketID, userID)
	if err != nil {
		return nil, err
	}

	store, err := s.GetObjectStore(ctx, bucketID, userID, encryptionKey)
	if err != nil {
		return nil, err
	}

	// Normalize prefix
	s3Prefix := prefix
	if s3Prefix != "" && !strings.HasSuffix(s3Prefix, "/") {
		s3Prefix += "/"
	}

	objects, err := store.ListObjects(ctx, bucketName, s3Prefix)
	if err != nil {
		return nil, err
	}

	// Group folders and files
	folderMap := make(map[string]BucketObject)
	var folderKeys []string
	var files []BucketObject

	for _, obj := range objects {
		if obj.Key == nil {
			continue
		}
		key := *obj.Key

		// Handle prefix filtering and key normalization
		if s3Prefix != "" {
			if !strings.HasPrefix(key, s3Prefix) {
				continue
			}
			// Remove the prefix from the key
			key = strings.TrimPrefix(key, s3Prefix)

			// Strip any remaining leading slashes
			if strings.HasPrefix(key, "/") {
				key = strings.TrimLeft(key, "/")
			}
		}

		if key == "" {
			continue
		}

		// If key contains /, it's inside a folder - create folder entry
		if strings.Contains(key, "/") {
			parts := strings.SplitN(key, "/", 2)
			folderRaw := parts[0]
			folderDisplay := folderRaw

			// If folder name is empty, display it as "(empty)"
			if folderRaw == "" {
				folderDisplay = "(empty)"
			}

			folderPrefix := s3Prefix + folderRaw
			if !strings.HasSuffix(folderPrefix, "/") {
				folderPrefix += "/"
			}

			if _, exists := folderMap[folderPrefix]; !exists {
				folderMap[folderPrefix] = BucketObject{
					Key:       folderPrefix,
					Name:      folderDisplay,
					Kind:      "folder",
					Icon:      "folder",
					IconColor: "text-amber-500",
					Size:      "",
				}
				folderKeys = append(folderKeys, folderPrefix)
			}
			continue
		}

		// It's a file in the current directory
		size := int64(0)
		if obj.Size != nil {
			size = *obj.Size
		}

		var lastModified time.Time
		if obj.LastModified != nil {
			lastModified = *obj.LastModified
		}

		files = append(files, BucketObject{
			Key:          s3Prefix + key,
			Name:         key,
			Kind:         "file",
			LastModified: lastModified,
			Size:         formatByteSize(size),
			Icon:         "description",
			IconColor:    "text-slate-500",
		})
	}

	// Sort folders alphabetically
	sort.Strings(folderKeys)
	folders := make([]BucketObject, 0, len(folderKeys))
	for _, key := range folderKeys {
		folders = append(folders, folderMap[key])
	}

	// Sort files alphabetically by name
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	// Return folders first, then files
	result := append(folders, files...)
	return result, nil
}

// formatByteSize formats bytes into human-readable format
func formatByteSize(bytes int64) string {
	if bytes <= 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	f := float64(bytes)
	i := 0
	for f >= 1024 && i < len(units)-1 {
		f /= 1024
		i++
	}
	return fmt.Sprintf("%.1f %s", f, units[i])
}

// SearchObjects searches for objects matching a query
func (s *BucketService) SearchObjects(ctx context.Context, bucketID, userID uuid.UUID, query string, encryptionKey []byte) ([]BucketObject, error) {
	// Get all objects and filter by query
	objects, err := s.ListObjects(ctx, bucketID, userID, "", encryptionKey)
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	var filtered []BucketObject
	for _, obj := range objects {
		if strings.Contains(strings.ToLower(obj.Key), query) {
			filtered = append(filtered, obj)
		}
	}

	return filtered, nil
}

// UploadObject uploads an object to a bucket
func (s *BucketService) UploadObject(ctx context.Context, bucketID, userID uuid.UUID, key string, body io.Reader, contentType string, encryptionKey []byte) error {
	bucketName, err := s.getBucketName(ctx, bucketID, userID)
	if err != nil {
		return err
	}

	store, err := s.GetObjectStore(ctx, bucketID, userID, encryptionKey)
	if err != nil {
		return err
	}

	if err := store.PutObject(ctx, bucketName, key, body, contentType, nil); err != nil {
		return err
	}

	// Update bucket size asynchronously (don't block on errors)
	go func() {
		if err := s.recalculateBucketSize(context.Background(), bucketID, userID, encryptionKey); err != nil {
			s.logger.Error("failed to update bucket size after upload", slog.Any("error", err), slog.String("bucket_id", bucketID.String()))
		}
	}()

	return nil
}

// PresignObject generates a presigned URL for an object
func (s *BucketService) PresignObject(ctx context.Context, bucketID, userID uuid.UUID, input PresignInput, encryptionKey []byte) (*PresignOutput, error) {
	// Check if user is a demo user
	user, err := s.users.GetByID(ctx, userID)
	if err == nil && user.IsDemo {
		return nil, ErrDemoRestriction
	}

	bucketName, err := s.getBucketName(ctx, bucketID, userID)
	if err != nil {
		return nil, err
	}

	store, err := s.GetObjectStore(ctx, bucketID, userID, encryptionKey)
	if err != nil {
		return nil, err
	}

	presigned, err := store.PresignObject(ctx, storage.PresignInput{
		Bucket:      bucketName,
		Key:         input.Key,
		Method:      input.Method,
		ExpiresIn:   input.Expires,
		ContentType: input.ContentType,
	})
	if err != nil {
		return nil, err
	}

	expiryTime := time.Now().Add(input.Expires).Unix()
	return &PresignOutput{
		URL:     presigned.URL,
		Expires: expiryTime,
	}, nil
}

// GetObjectMetadata retrieves metadata for an object
func (s *BucketService) GetObjectMetadata(ctx context.Context, bucketID, userID uuid.UUID, key string, encryptionKey []byte) (*ObjectMetadata, error) {
	// Check if user is a demo user
	user, err := s.users.GetByID(ctx, userID)
	if err == nil && user.IsDemo {
		return nil, ErrDemoRestriction
	}

	bucketName, err := s.getBucketName(ctx, bucketID, userID)
	if err != nil {
		return nil, err
	}

	store, err := s.GetObjectStore(ctx, bucketID, userID, encryptionKey)
	if err != nil {
		return nil, err
	}

	head, err := store.HeadObject(ctx, bucketName, key)
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]string)
	for k, v := range head.Metadata {
		metadata[k] = v
	}

	contentType := "application/octet-stream"
	if head.ContentType != nil {
		contentType = *head.ContentType
	}

	return &ObjectMetadata{
		Key:          key,
		Size:         awsInt64Value(head.ContentLength),
		LastModified: awsTimeValue(head.LastModified),
		ContentType:  contentType,
		ETag:         strings.Trim(awsStringValue(head.ETag), "\""),
		Metadata:     metadata,
	}, nil
}

// ProxyObject retrieves an object for proxying/download
func (s *BucketService) ProxyObject(ctx context.Context, bucketID, userID uuid.UUID, key string, encryptionKey []byte) (*ProxiedObject, error) {
	// Check if user is a demo user
	user, err := s.users.GetByID(ctx, userID)
	if err == nil && user.IsDemo {
		return nil, ErrDemoRestriction
	}

	bucketName, err := s.getBucketName(ctx, bucketID, userID)
	if err != nil {
		return nil, err
	}

	store, err := s.GetObjectStore(ctx, bucketID, userID, encryptionKey)
	if err != nil {
		return nil, err
	}

	obj, err := store.GetObject(ctx, bucketName, key)
	if err != nil {
		return nil, err
	}

	contentType := "application/octet-stream"
	if obj.ContentType != nil {
		contentType = *obj.ContentType
	}

	return &ProxiedObject{
		Body:          obj.Body,
		ContentType:   contentType,
		ContentLength: awsInt64Value(obj.ContentLength),
	}, nil
}

// CreateFolder creates an empty folder (0-byte object with trailing slash)
func (s *BucketService) CreateFolder(ctx context.Context, bucketID, userID uuid.UUID, name string, prefix *string, encryptionKey []byte) (*FolderResult, error) {
	bucketName, err := s.getBucketName(ctx, bucketID, userID)
	if err != nil {
		return nil, err
	}

	store, err := s.GetObjectStore(ctx, bucketID, userID, encryptionKey)
	if err != nil {
		return nil, err
	}

	// Construct folder key
	key := name
	if prefix != nil && *prefix != "" {
		key = strings.TrimSuffix(*prefix, "/") + "/" + name
	}
	if !strings.HasSuffix(key, "/") {
		key += "/"
	}

	contentType := "application/x-directory"
	if err := store.PutEmptyObject(ctx, bucketName, key, &contentType); err != nil {
		return nil, err
	}

	return &FolderResult{Key: key}, nil
}

// DeleteObjects deletes multiple objects
func (s *BucketService) DeleteObjects(ctx context.Context, bucketID, userID uuid.UUID, keys []string, encryptionKey []byte) (*DeleteObjectsResult, error) {
	bucketName, err := s.getBucketName(ctx, bucketID, userID)
	if err != nil {
		return nil, err
	}

	store, err := s.GetObjectStore(ctx, bucketID, userID, encryptionKey)
	if err != nil {
		return nil, err
	}

	// Expand folder keys to include all objects within them
	allKeysToDelete := []string{}
	for _, key := range keys {
		if strings.HasSuffix(key, "/") {
			// It's a folder - list all objects with this prefix
			objects, err := store.ListAllObjects(ctx, bucketName, key)
			if err != nil {
				return &DeleteObjectsResult{
					Deleted: []string{},
					Failed:  keys,
				}, fmt.Errorf("failed to list folder contents: %w", err)
			}

			// Add all object keys from the folder
			for _, obj := range objects {
				if obj.Key != nil {
					allKeysToDelete = append(allKeysToDelete, *obj.Key)
				}
			}

			// Also add the folder marker itself
			allKeysToDelete = append(allKeysToDelete, key)
		} else {
			// It's a regular file
			allKeysToDelete = append(allKeysToDelete, key)
		}
	}

	if len(allKeysToDelete) == 0 {
		return &DeleteObjectsResult{
			Deleted: []string{},
			Failed:  []string{},
		}, nil
	}

	if err := store.DeleteObjects(ctx, bucketName, allKeysToDelete); err != nil {
		return &DeleteObjectsResult{
			Deleted: []string{},
			Failed:  keys,
		}, err
	}

	// Update bucket size asynchronously (don't block on errors)
	go func() {
		if err := s.recalculateBucketSize(context.Background(), bucketID, userID, encryptionKey); err != nil {
			s.logger.Error("failed to update bucket size after delete", slog.Any("error", err), slog.String("bucket_id", bucketID.String()))
		}
	}()

	return &DeleteObjectsResult{
		Deleted: keys,
		Failed:  []string{},
	}, nil
}

// RenameObject renames an object (copy + delete)
func (s *BucketService) RenameObject(ctx context.Context, bucketID, userID uuid.UUID, sourceKey, destinationKey string, encryptionKey []byte) (*OperationResult, error) {
	bucketName, err := s.getBucketName(ctx, bucketID, userID)
	if err != nil {
		return nil, err
	}

	store, err := s.GetObjectStore(ctx, bucketID, userID, encryptionKey)
	if err != nil {
		return nil, err
	}

	if sourceKey == destinationKey {
		return &OperationResult{
			Success: true,
			Message: "Object already has that name",
		}, nil
	}

	isFolder := strings.HasSuffix(sourceKey, "/")
	if isFolder && !strings.HasSuffix(destinationKey, "/") {
		destinationKey += "/"
	}
	if sourceKey == destinationKey {
		return &OperationResult{
			Success: true,
			Message: "Object already has that name",
		}, nil
	}

	// Check if it's a folder
	if isFolder {
		if strings.HasPrefix(destinationKey, sourceKey) {
			return &OperationResult{
				Success: false,
				Message: "Cannot move a folder into itself",
			}, fmt.Errorf("destination %s is inside source %s", destinationKey, sourceKey)
		}

		// It's a folder - list all objects with this prefix
		objects, err := store.ListAllObjects(ctx, bucketName, sourceKey)
		if err != nil {
			return &OperationResult{
				Success: false,
				Message: fmt.Sprintf("failed to list folder contents: %v", err),
			}, err
		}

		// Copy all objects from the folder and collect keys to delete afterwards
		keysToDelete := make([]string, 0, len(objects)+1)
		for _, obj := range objects {
			if obj.Key == nil {
				continue
			}

			// Calculate new key by replacing the source prefix with destination prefix
			oldKey := *obj.Key
			newKey := strings.Replace(oldKey, sourceKey, destinationKey, 1)

			if err := store.CopyObject(ctx, bucketName, oldKey, newKey); err != nil {
				return &OperationResult{
					Success: false,
					Message: fmt.Sprintf("failed to copy object %s: %v", oldKey, err),
				}, err
			}

			keysToDelete = append(keysToDelete, oldKey)
		}

		// Copy the folder marker itself
		if err := store.CopyObject(ctx, bucketName, sourceKey, destinationKey); err != nil {
			return &OperationResult{
				Success: false,
				Message: fmt.Sprintf("failed to copy folder marker: %v", err),
			}, err
		}

		keysToDelete = append(keysToDelete, sourceKey)

		// Delete original folder (this will use the updated DeleteObjects which handles folders)
		if err := store.DeleteObjects(ctx, bucketName, keysToDelete); err != nil {
			return &OperationResult{
				Success: false,
				Message: fmt.Sprintf("copied but failed to delete original: %v", err),
			}, err
		}
	} else {
		// It's a regular file
		if err := store.CopyObject(ctx, bucketName, sourceKey, destinationKey); err != nil {
			return &OperationResult{
				Success: false,
				Message: fmt.Sprintf("failed to copy object: %v", err),
			}, err
		}

		// Delete original
		if err := store.DeleteObjects(ctx, bucketName, []string{sourceKey}); err != nil {
			return &OperationResult{
				Success: false,
				Message: fmt.Sprintf("copied but failed to delete original: %v", err),
			}, err
		}
	}

	return &OperationResult{
		Success: true,
		Message: "Object renamed successfully",
	}, nil
}

// CopyObject copies an object
func (s *BucketService) CopyObject(ctx context.Context, bucketID, userID uuid.UUID, sourceKey, destinationKey string, encryptionKey []byte) (*OperationResult, error) {
	bucketName, err := s.getBucketName(ctx, bucketID, userID)
	if err != nil {
		return nil, err
	}

	store, err := s.GetObjectStore(ctx, bucketID, userID, encryptionKey)
	if err != nil {
		return nil, err
	}

	isFolder := strings.HasSuffix(sourceKey, "/")
	if isFolder && !strings.HasSuffix(destinationKey, "/") {
		destinationKey += "/"
	}
	if sourceKey == destinationKey {
		return &OperationResult{
			Success: false,
			Message: "Destination must be different from source",
		}, fmt.Errorf("source and destination are identical")
	}

	if isFolder {
		if strings.HasPrefix(destinationKey, sourceKey) {
			return &OperationResult{
				Success: false,
				Message: "Cannot copy a folder into itself",
			}, fmt.Errorf("destination %s is inside source %s", destinationKey, sourceKey)
		}

		objects, err := store.ListAllObjects(ctx, bucketName, sourceKey)
		if err != nil {
			return &OperationResult{
				Success: false,
				Message: fmt.Sprintf("failed to list folder contents: %v", err),
			}, err
		}

		for _, obj := range objects {
			if obj.Key == nil {
				continue
			}
			oldKey := *obj.Key
			newKey := strings.Replace(oldKey, sourceKey, destinationKey, 1)

			if err := store.CopyObject(ctx, bucketName, oldKey, newKey); err != nil {
				return &OperationResult{
					Success: false,
					Message: fmt.Sprintf("failed to copy object %s: %v", oldKey, err),
				}, err
			}
		}

		if err := store.CopyObject(ctx, bucketName, sourceKey, destinationKey); err != nil {
			return &OperationResult{
				Success: false,
				Message: fmt.Sprintf("failed to copy folder marker: %v", err),
			}, err
		}
	} else {
		if err := store.CopyObject(ctx, bucketName, sourceKey, destinationKey); err != nil {
			return &OperationResult{
				Success: false,
				Message: fmt.Sprintf("failed to copy object: %v", err),
			}, err
		}
	}

	return &OperationResult{
		Success: true,
		Message: "Object copied successfully",
	}, nil
}

// ZipFolder creates a zip archive of a folder
func (s *BucketService) ZipFolder(ctx context.Context, bucketID, userID uuid.UUID, prefix string, encryptionKey []byte) (io.ReadCloser, string, error) {
	// Check if user is a demo user
	user, err := s.users.GetByID(ctx, userID)
	if err == nil && user.IsDemo {
		return nil, "", ErrDemoRestriction
	}

	bucketName, err := s.getBucketName(ctx, bucketID, userID)
	if err != nil {
		return nil, "", err
	}

	store, err := s.GetObjectStore(ctx, bucketID, userID, encryptionKey)
	if err != nil {
		return nil, "", err
	}

	// List all objects with the prefix
	objects, err := store.ListAllObjects(ctx, bucketName, prefix)
	if err != nil {
		return nil, "", err
	}

	// Create a pipe for streaming the zip
	pr, pw := io.Pipe()

	// Generate filename
	folderName := strings.TrimSuffix(prefix, "/")
	if folderName == "" {
		folderName = "download"
	}
	filename := fmt.Sprintf("%s.zip", folderName)

	// Stream zip creation in a goroutine
	go func() {
		defer pw.Close()

		zipWriter := zip.NewWriter(pw)
		defer zipWriter.Close()

		for _, obj := range objects {
			if obj.Key == nil {
				continue
			}

			key := *obj.Key

			// Skip the folder marker itself
			if key == prefix {
				continue
			}

			// Get the object
			objData, err := store.GetObject(ctx, bucketName, key)
			if err != nil {
				s.logger.Warn("failed to get object for zip", "key", key, "error", err)
				continue
			}

			// Create zip entry using a sanitized path to avoid zip-slip attacks
			relativePath := strings.TrimPrefix(key, prefix)
			safePath := path.Clean(relativePath)
			safePath = strings.TrimPrefix(safePath, "/")
			if safePath == "" || safePath == "." || strings.HasPrefix(safePath, "..") || strings.Contains(safePath, "../") {
				s.logger.Warn("skipping object with unsafe path", slog.String("key", key))
				objData.Body.Close()
				continue
			}

			writer, err := zipWriter.Create(safePath)
			if err != nil {
				objData.Body.Close()
				s.logger.Warn("failed to create zip entry", "key", key, "error", err)
				continue
			}

			// Copy content
			if _, err := io.Copy(writer, objData.Body); err != nil {
				objData.Body.Close()
				s.logger.Warn("failed to write to zip", "key", key, "error", err)
				continue
			}

			objData.Body.Close()
		}
	}()

	return pr, filename, nil
}

// Helper functions for AWS SDK pointers
func awsStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func awsInt64Value(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

func awsTimeValue(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// recalculateBucketSize calculates and updates the bucket size in the database
func (s *BucketService) recalculateBucketSize(ctx context.Context, bucketID, userID uuid.UUID, encryptionKey []byte) error {
	bucketName, err := s.getBucketName(ctx, bucketID, userID)
	if err != nil {
		return err
	}

	store, err := s.GetObjectStore(ctx, bucketID, userID, encryptionKey)
	if err != nil {
		return err
	}

	totalSize, err := store.CalculateBucketSize(ctx, bucketName)
	if err != nil {
		return err
	}

	return s.UpdateSize(ctx, bucketID, totalSize)
}

// RecalculateBucketSize is a public wrapper for recalculateBucketSize
func (s *BucketService) RecalculateBucketSize(ctx context.Context, bucketID, userID uuid.UUID, encryptionKey []byte) error {
	return s.recalculateBucketSize(ctx, bucketID, userID, encryptionKey)
}
