package repository

import (
	"context"
	"errors"
	"time"

	"bucketbird/backend/internal/repository/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

// Helper functions to convert between pgtype and standard Go types
func uuidToPgtype(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func pgtypeToUUID(id pgtype.UUID) uuid.UUID {
	return id.Bytes
}

func timeToPgtype(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func pgtypeToTime(t pgtype.Timestamptz) time.Time {
	return t.Time
}

// Repositories holds all repository implementations
type Repositories struct {
	Users       UserRepository
	Sessions    SessionRepository
	Credentials CredentialRepository
	Buckets     BucketRepository
	Profiles    ProfileRepository
}

func NewRepositories(pool *pgxpool.Pool) *Repositories {
	q := sqlc.New(pool)
	return &Repositories{
		Users:       &pgUserRepository{q: q},
		Sessions:    &pgSessionRepository{q: q},
		Credentials: &pgCredentialRepository{q: q},
		Buckets:     &pgBucketRepository{q: q},
		Profiles:    &pgProfileRepository{q: q},
	}
}

// ========== UserRepository implementation ==========

type pgUserRepository struct {
	q *sqlc.Queries
}

func (r *pgUserRepository) Create(ctx context.Context, email, passwordHash, firstName, lastName string) (*User, error) {
	user, err := r.q.InsertUser(ctx, sqlc.InsertUserParams{
		ID:           uuidToPgtype(uuid.New()),
		Email:        email,
		PasswordHash: passwordHash,
		FirstName:    firstName,
		LastName:     lastName,
	})
	if err != nil {
		return nil, err
	}
	return &User{
		ID:           pgtypeToUUID(user.ID),
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		IsDemo:       user.IsDemo,
		CreatedAt:    pgtypeToTime(user.CreatedAt),
		UpdatedAt:    pgtypeToTime(user.UpdatedAt),
	}, nil
}

func (r *pgUserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	user, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &User{
		ID:           pgtypeToUUID(user.ID),
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		IsDemo:       user.IsDemo,
		CreatedAt:    pgtypeToTime(user.CreatedAt),
		UpdatedAt:    pgtypeToTime(user.UpdatedAt),
	}, nil
}

func (r *pgUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	user, err := r.q.GetUserByID(ctx, uuidToPgtype(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &User{
		ID:           pgtypeToUUID(user.ID),
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		IsDemo:       user.IsDemo,
		CreatedAt:    pgtypeToTime(user.CreatedAt),
		UpdatedAt:    pgtypeToTime(user.UpdatedAt),
	}, nil
}

func (r *pgUserRepository) Update(ctx context.Context, id uuid.UUID, email, firstName, lastName string) error {
	return r.q.UpdateUser(ctx, sqlc.UpdateUserParams{
		ID:        uuidToPgtype(id),
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
	})
}

func (r *pgUserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	return r.q.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		ID:           uuidToPgtype(id),
		PasswordHash: passwordHash,
	})
}

func (r *pgUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteUser(ctx, uuidToPgtype(id))
}

// ========== SessionRepository implementation ==========

type pgSessionRepository struct {
	q *sqlc.Queries
}

func (r *pgSessionRepository) Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*Session, error) {
	session, err := r.q.CreateSession(ctx, sqlc.CreateSessionParams{
		ID:               uuidToPgtype(uuid.New()),
		UserID:           uuidToPgtype(userID),
		RefreshTokenHash: tokenHash,
		ExpiresAt:        timeToPgtype(expiresAt),
	})
	if err != nil {
		return nil, err
	}
	return &Session{
		ID:               pgtypeToUUID(session.ID),
		UserID:           pgtypeToUUID(session.UserID),
		RefreshTokenHash: session.RefreshTokenHash,
		ExpiresAt:        pgtypeToTime(session.ExpiresAt),
		CreatedAt:        pgtypeToTime(session.CreatedAt),
		UpdatedAt:        pgtypeToTime(session.UpdatedAt),
	}, nil
}

func (r *pgSessionRepository) GetByHash(ctx context.Context, hash string) (*Session, error) {
	session, err := r.q.GetSessionByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &Session{
		ID:               pgtypeToUUID(session.ID),
		UserID:           pgtypeToUUID(session.UserID),
		RefreshTokenHash: session.RefreshTokenHash,
		ExpiresAt:        pgtypeToTime(session.ExpiresAt),
		CreatedAt:        pgtypeToTime(session.CreatedAt),
		UpdatedAt:        pgtypeToTime(session.UpdatedAt),
	}, nil
}

func (r *pgSessionRepository) UpdateToken(ctx context.Context, sessionID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	return r.q.UpdateSessionToken(ctx, sqlc.UpdateSessionTokenParams{
		ID:               uuidToPgtype(sessionID),
		RefreshTokenHash: tokenHash,
		ExpiresAt:        timeToPgtype(expiresAt),
	})
}

func (r *pgSessionRepository) DeleteByHash(ctx context.Context, hash string) error {
	return r.q.DeleteSessionByHash(ctx, hash)
}

func (r *pgSessionRepository) DeleteForUser(ctx context.Context, userID uuid.UUID) error {
	return r.q.DeleteSessionsForUser(ctx, uuidToPgtype(userID))
}

// ========== CredentialRepository implementation ==========

type pgCredentialRepository struct {
	q *sqlc.Queries
}

func (r *pgCredentialRepository) Create(ctx context.Context, cred *Credential) (*Credential, error) {
	created, err := r.q.CreateCredential(ctx, sqlc.CreateCredentialParams{
		ID:                 uuidToPgtype(uuid.New()),
		UserID:             uuidToPgtype(cred.UserID),
		Name:               cred.Name,
		Provider:           cred.Provider,
		Region:             cred.Region,
		Endpoint:           cred.Endpoint,
		EncryptedAccessKey: cred.EncryptedAccessKey,
		EncryptedSecretKey: cred.EncryptedSecretKey,
		UseSsl:             cred.UseSSL,
		Status:             cred.Status,
		Logo:               cred.Logo,
	})
	if err != nil {
		return nil, err
	}
	cred.ID = pgtypeToUUID(created.ID)
	cred.CreatedAt = pgtypeToTime(created.CreatedAt)
	cred.UpdatedAt = pgtypeToTime(created.UpdatedAt)
	return cred, nil
}

func (r *pgCredentialRepository) List(ctx context.Context, userID uuid.UUID) ([]*Credential, error) {
	creds, err := r.q.ListCredentials(ctx, uuidToPgtype(userID))
	if err != nil {
		return nil, err
	}
	result := make([]*Credential, len(creds))
	for i, c := range creds {
		result[i] = &Credential{
			ID:                 pgtypeToUUID(c.ID),
			UserID:             pgtypeToUUID(c.UserID),
			Name:               c.Name,
			Provider:           c.Provider,
			Region:             c.Region,
			Endpoint:           c.Endpoint,
			EncryptedAccessKey: c.EncryptedAccessKey,
			EncryptedSecretKey: c.EncryptedSecretKey,
			UseSSL:             c.UseSsl,
			Status:             c.Status,
			Logo:               c.Logo,
			CreatedAt:          pgtypeToTime(c.CreatedAt),
			UpdatedAt:          pgtypeToTime(c.UpdatedAt),
		}
	}
	return result, nil
}

func (r *pgCredentialRepository) Get(ctx context.Context, id, userID uuid.UUID) (*Credential, error) {
	cred, err := r.q.GetCredential(ctx, sqlc.GetCredentialParams{
		ID:     uuidToPgtype(id),
		UserID: uuidToPgtype(userID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &Credential{
		ID:                 pgtypeToUUID(cred.ID),
		UserID:             pgtypeToUUID(cred.UserID),
		Name:               cred.Name,
		Provider:           cred.Provider,
		Region:             cred.Region,
		Endpoint:           cred.Endpoint,
		EncryptedAccessKey: cred.EncryptedAccessKey,
		EncryptedSecretKey: cred.EncryptedSecretKey,
		UseSSL:             cred.UseSsl,
		Status:             cred.Status,
		Logo:               cred.Logo,
		CreatedAt:          pgtypeToTime(cred.CreatedAt),
		UpdatedAt:          pgtypeToTime(cred.UpdatedAt),
	}, nil
}

func (r *pgCredentialRepository) Update(ctx context.Context, cred *Credential) error {
	err := r.q.UpdateCredential(ctx, sqlc.UpdateCredentialParams{
		ID:                 uuidToPgtype(cred.ID),
		UserID:             uuidToPgtype(cred.UserID),
		Name:               cred.Name,
		Provider:           cred.Provider,
		Region:             cred.Region,
		Endpoint:           cred.Endpoint,
		EncryptedAccessKey: cred.EncryptedAccessKey,
		EncryptedSecretKey: cred.EncryptedSecretKey,
		UseSsl:             cred.UseSSL,
		Status:             cred.Status,
		Logo:               cred.Logo,
	})
	return err
}

func (r *pgCredentialRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return r.q.DeleteCredential(ctx, sqlc.DeleteCredentialParams{
		ID:     uuidToPgtype(id),
		UserID: uuidToPgtype(userID),
	})
}

// ========== BucketRepository implementation ==========

type pgBucketRepository struct {
	q *sqlc.Queries
}

func (r *pgBucketRepository) Create(ctx context.Context, bucket *Bucket) (*Bucket, error) {
	created, err := r.q.InsertBucket(ctx, sqlc.InsertBucketParams{
		ID:           uuidToPgtype(uuid.New()),
		UserID:       uuidToPgtype(bucket.UserID),
		CredentialID: uuidToPgtype(bucket.CredentialID),
		Name:         bucket.Name,
		Region:       bucket.Region,
		Description:  bucket.Description,
	})
	if err != nil {
		return nil, err
	}
	bucket.ID = pgtypeToUUID(created.ID)
	bucket.SizeBytes = created.SizeBytes
	bucket.CreatedAt = pgtypeToTime(created.CreatedAt)
	bucket.UpdatedAt = pgtypeToTime(created.UpdatedAt)
	return bucket, nil
}

func (r *pgBucketRepository) List(ctx context.Context, userID uuid.UUID) ([]*BucketWithCredential, error) {
	buckets, err := r.q.ListBuckets(ctx, uuidToPgtype(userID))
	if err != nil {
		return nil, err
	}
	result := make([]*BucketWithCredential, len(buckets))
	for i, b := range buckets {
		result[i] = &BucketWithCredential{
			Bucket: Bucket{
				ID:           pgtypeToUUID(b.Bucket.ID),
				UserID:       pgtypeToUUID(b.Bucket.UserID),
				CredentialID: pgtypeToUUID(b.Bucket.CredentialID),
				Name:         b.Bucket.Name,
				Region:       b.Bucket.Region,
				Description:  b.Bucket.Description,
				SizeBytes:    b.Bucket.SizeBytes,
				CreatedAt:    pgtypeToTime(b.Bucket.CreatedAt),
				UpdatedAt:    pgtypeToTime(b.Bucket.UpdatedAt),
			},
			CredentialName:     b.CredentialName,
			CredentialProvider: b.CredentialProvider,
		}
	}
	return result, nil
}

func (r *pgBucketRepository) Get(ctx context.Context, id, userID uuid.UUID) (*BucketWithCredential, error) {
	b, err := r.q.GetBucket(ctx, sqlc.GetBucketParams{
		ID:     uuidToPgtype(id),
		UserID: uuidToPgtype(userID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &BucketWithCredential{
		Bucket: Bucket{
			ID:           pgtypeToUUID(b.Bucket.ID),
			UserID:       pgtypeToUUID(b.Bucket.UserID),
			CredentialID: pgtypeToUUID(b.Bucket.CredentialID),
			Name:         b.Bucket.Name,
			Region:       b.Bucket.Region,
			Description:  b.Bucket.Description,
			SizeBytes:    b.Bucket.SizeBytes,
			CreatedAt:    pgtypeToTime(b.Bucket.CreatedAt),
			UpdatedAt:    pgtypeToTime(b.Bucket.UpdatedAt),
		},
		CredentialName:     b.CredentialName,
		CredentialProvider: b.CredentialProvider,
	}, nil
}

func (r *pgBucketRepository) GetByName(ctx context.Context, userID uuid.UUID, name string) (*BucketWithCredential, error) {
	b, err := r.q.GetBucketByName(ctx, sqlc.GetBucketByNameParams{
		UserID: uuidToPgtype(userID),
		Name:   name,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &BucketWithCredential{
		Bucket: Bucket{
			ID:           pgtypeToUUID(b.Bucket.ID),
			UserID:       pgtypeToUUID(b.Bucket.UserID),
			CredentialID: pgtypeToUUID(b.Bucket.CredentialID),
			Name:         b.Bucket.Name,
			Region:       b.Bucket.Region,
			Description:  b.Bucket.Description,
			SizeBytes:    b.Bucket.SizeBytes,
			CreatedAt:    pgtypeToTime(b.Bucket.CreatedAt),
			UpdatedAt:    pgtypeToTime(b.Bucket.UpdatedAt),
		},
		CredentialName:     b.CredentialName,
		CredentialProvider: b.CredentialProvider,
	}, nil
}

func (r *pgBucketRepository) Update(ctx context.Context, id, userID uuid.UUID, description *string) error {
	return r.q.UpdateBucket(ctx, sqlc.UpdateBucketParams{
		ID:          uuidToPgtype(id),
		UserID:      uuidToPgtype(userID),
		Description: description,
	})
}

func (r *pgBucketRepository) UpdateSize(ctx context.Context, id uuid.UUID, sizeBytes int64) error {
	return r.q.UpdateBucketSize(ctx, sqlc.UpdateBucketSizeParams{
		ID:        uuidToPgtype(id),
		SizeBytes: sizeBytes,
	})
}

func (r *pgBucketRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return r.q.DeleteBucket(ctx, sqlc.DeleteBucketParams{
		ID:     uuidToPgtype(id),
		UserID: uuidToPgtype(userID),
	})
}

// ========== ProfileRepository implementation ==========

type pgProfileRepository struct {
	q *sqlc.Queries
}

func (r *pgProfileRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	profile, err := r.q.GetProfileByUserID(ctx, uuidToPgtype(userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &Profile{
		ID:            pgtypeToUUID(profile.ID),
		UserID:        pgtypeToUUID(profile.UserID),
		FirstName:     profile.FirstName,
		LastName:      profile.LastName,
		Email:         profile.Email,
		Language:      profile.Language,
		Timezone:      profile.Timezone,
		AvatarURL:     profile.AvatarUrl,
		YoutubeCookie: profile.YoutubeCookie,
		CreatedAt:     pgtypeToTime(profile.CreatedAt),
		UpdatedAt:     pgtypeToTime(profile.UpdatedAt),
	}, nil
}

func (r *pgProfileRepository) UpdateYouTubeCookie(ctx context.Context, userID uuid.UUID, cookie *string) error {
	return r.q.UpdateYouTubeCookie(ctx, sqlc.UpdateYouTubeCookieParams{
		UserID:        uuidToPgtype(userID),
		YoutubeCookie: cookie,
	})
}

// Verify interface compliance
var (
	_ UserRepository       = (*pgUserRepository)(nil)
	_ SessionRepository    = (*pgSessionRepository)(nil)
	_ CredentialRepository = (*pgCredentialRepository)(nil)
	_ BucketRepository     = (*pgBucketRepository)(nil)
	_ ProfileRepository    = (*pgProfileRepository)(nil)
)
