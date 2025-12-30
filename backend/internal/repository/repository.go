package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// UserRepository defines operations for user data access
type UserRepository interface {
	Create(ctx context.Context, email, passwordHash, firstName, lastName string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	Update(ctx context.Context, id uuid.UUID, email, firstName, lastName string) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// SessionRepository defines operations for session management
type SessionRepository interface {
	Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*Session, error)
	GetByHash(ctx context.Context, hash string) (*Session, error)
	UpdateToken(ctx context.Context, sessionID uuid.UUID, tokenHash string, expiresAt time.Time) error
	DeleteByHash(ctx context.Context, hash string) error
	DeleteForUser(ctx context.Context, userID uuid.UUID) error
}

// CredentialRepository defines operations for S3 credential management
type CredentialRepository interface {
	Create(ctx context.Context, cred *Credential) (*Credential, error)
	List(ctx context.Context, userID uuid.UUID) ([]*Credential, error)
	Get(ctx context.Context, id, userID uuid.UUID) (*Credential, error)
	Update(ctx context.Context, cred *Credential) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

// BucketRepository defines operations for bucket management
type BucketRepository interface {
	Create(ctx context.Context, bucket *Bucket) (*Bucket, error)
	List(ctx context.Context, userID uuid.UUID) ([]*BucketWithCredential, error)
	Get(ctx context.Context, id, userID uuid.UUID) (*BucketWithCredential, error)
	GetByName(ctx context.Context, userID uuid.UUID, name string) (*BucketWithCredential, error)
	Update(ctx context.Context, id, userID uuid.UUID, description *string) error
	UpdateSize(ctx context.Context, id uuid.UUID, sizeBytes int64) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

// ProfileRepository defines operations for profile management
type ProfileRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*Profile, error)
	UpdateYouTubeCookie(ctx context.Context, userID uuid.UUID, cookie *string) error
}

// Domain models (converted from pgtype to standard types)
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	IsDemo       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Session struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	RefreshTokenHash string
	ExpiresAt        time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Credential struct {
	ID                 uuid.UUID
	UserID             uuid.UUID
	Name               string
	Provider           string
	Region             string
	Endpoint           string
	EncryptedAccessKey string
	EncryptedSecretKey string
	UseSSL             bool
	Status             string
	Logo               *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Bucket struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	CredentialID uuid.UUID
	Name         string
	Region       string
	Description  *string
	SizeBytes    int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type BucketWithCredential struct {
	Bucket
	CredentialName     string
	CredentialProvider string
}

type Profile struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	FirstName     string
	LastName      string
	Email         string
	Language      string
	Timezone      string
	AvatarURL     *string
	YoutubeCookie *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
