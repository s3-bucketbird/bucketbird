package service

import (
	"context"
	"errors"
	"log/slog"

	"bucketbird/backend/internal/repository"
	"bucketbird/backend/internal/storage"

	"github.com/google/uuid"
	"github.com/kkdai/youtube/v2"
)

type BucketService struct {
	buckets       repository.BucketRepository
	credentials   repository.CredentialRepository
	users         repository.UserRepository
	encryptionKey []byte
	logger        *slog.Logger
	youtubeClient *youtube.Client
}

func NewBucketService(
	buckets repository.BucketRepository,
	credentials repository.CredentialRepository,
	users repository.UserRepository,
	encryptionKey []byte,
	logger *slog.Logger,
) *BucketService {
	return &BucketService{
		buckets:       buckets,
		credentials:   credentials,
		users:         users,
		encryptionKey: encryptionKey,
		logger:        logger,
		youtubeClient: &youtube.Client{},
	}
}

type CreateBucketInput struct {
	UserID       uuid.UUID
	CredentialID uuid.UUID
	Name         string
	Region       string
	Description  *string
}

func (s *BucketService) Create(ctx context.Context, input CreateBucketInput) (*repository.BucketWithCredential, error) {
	// Validate credential exists and belongs to user
	cred, err := s.credentials.Get(ctx, input.CredentialID, input.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrCredentialNotFound
		}
		return nil, err
	}

	// Check if bucket name already exists for this user
	if existing, err := s.buckets.GetByName(ctx, input.UserID, input.Name); err == nil {
		s.logger.Warn("bucket already exists", slog.String("name", existing.Name))
		return nil, ErrBucketAlreadyExists
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	// Ensure the bucket exists (create if needed) using the credential's keys
	accessKey, err := decryptCredential(cred.EncryptedAccessKey, s.encryptionKey)
	if err != nil {
		return nil, err
	}

	secretKey, err := decryptCredential(cred.EncryptedSecretKey, s.encryptionKey)
	if err != nil {
		return nil, err
	}

	store, err := storage.NewObjectStoreWithCredentials(
		ctx,
		cred.Endpoint,
		cred.Region,
		accessKey,
		secretKey,
		cred.UseSSL,
	)
	if err != nil {
		return nil, newBucketProvisionError(err)
	}

	if err := store.EnsureBucket(ctx, input.Name); err != nil {
		return nil, newBucketProvisionError(err)
	}

	// Create bucket record
	bucket := &repository.Bucket{
		UserID:       input.UserID,
		CredentialID: input.CredentialID,
		Name:         input.Name,
		Region:       input.Region,
		Description:  input.Description,
	}

	created, err := s.buckets.Create(ctx, bucket)
	if err != nil {
		return nil, err
	}

	// Return with credential info
	return &repository.BucketWithCredential{
		Bucket:             *created,
		CredentialName:     cred.Name,
		CredentialProvider: cred.Provider,
	}, nil
}

func (s *BucketService) List(ctx context.Context, userID uuid.UUID) ([]*repository.BucketWithCredential, error) {
	return s.buckets.List(ctx, userID)
}

func (s *BucketService) Get(ctx context.Context, id, userID uuid.UUID) (*repository.BucketWithCredential, error) {
	bucket, err := s.buckets.Get(ctx, id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrBucketNotFound
		}
		return nil, err
	}
	return bucket, nil
}

func (s *BucketService) Update(ctx context.Context, id, userID uuid.UUID, description *string) error {
	// Verify bucket exists
	if _, err := s.buckets.Get(ctx, id, userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrBucketNotFound
		}
		return err
	}

	return s.buckets.Update(ctx, id, userID, description)
}

func (s *BucketService) Delete(ctx context.Context, id, userID uuid.UUID, deleteRemote bool) error {
	// Verify bucket exists
	bucket, err := s.buckets.Get(ctx, id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrBucketNotFound
		}
		return err
	}

	if deleteRemote {
		store, err := s.GetObjectStore(ctx, id, userID, s.encryptionKey)
		if err != nil {
			return err
		}

		if err := store.DeleteBucket(ctx, bucket.Name); err != nil {
			return err
		}
	}

	return s.buckets.Delete(ctx, id, userID)
}

func (s *BucketService) UpdateSize(ctx context.Context, bucketID uuid.UUID, sizeBytes int64) error {
	return s.buckets.UpdateSize(ctx, bucketID, sizeBytes)
}

// GetObjectStore creates an object store client for a specific bucket
func (s *BucketService) GetObjectStore(ctx context.Context, bucketID, userID uuid.UUID, encryptionKey []byte) (*storage.ObjectStore, error) {
	// Get bucket (includes credential info)
	bucket, err := s.buckets.Get(ctx, bucketID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrBucketNotFound
		}
		return nil, err
	}

	// Get credential details
	cred, err := s.credentials.Get(ctx, bucket.CredentialID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrCredentialNotFound
		}
		return nil, err
	}

	// Decrypt credentials
	accessKey, err := decryptCredential(cred.EncryptedAccessKey, encryptionKey)
	if err != nil {
		return nil, err
	}

	secretKey, err := decryptCredential(cred.EncryptedSecretKey, encryptionKey)
	if err != nil {
		return nil, err
	}

	// Create object store client
	return storage.NewObjectStoreWithCredentials(
		ctx,
		cred.Endpoint,
		cred.Region,
		accessKey,
		secretKey,
		cred.UseSSL,
	)
}

// Helper to get bucket name from bucket record
func (s *BucketService) getBucketName(ctx context.Context, bucketID, userID uuid.UUID) (string, error) {
	bucket, err := s.buckets.Get(ctx, bucketID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", ErrBucketNotFound
		}
		return "", err
	}
	return bucket.Name, nil
}
