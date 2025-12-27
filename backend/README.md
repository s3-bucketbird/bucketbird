# BucketBird Backend

BucketBird is a multi-cloud S3-compatible object storage management platform. This backend provides a RESTful API for managing S3 buckets, credentials, and objects across multiple S3-compatible storage providers.

## Architecture

The backend follows a clean architecture pattern with clear separation of concerns:

```
backend/
├── cmd/
│   └── bucketbird/          # CLI application entry point
│       ├── main.go         # Main CLI with subcommands
│       └── cmd/            # CLI commands (serve, migrate, user)
├── internal/                # Private application code
│   ├── api/                # HTTP handlers (presentation layer)
│   │   ├── auth/          # Authentication endpoints
│   │   ├── buckets/       # Bucket management endpoints
│   │   ├── credentials/   # Credential management endpoints
│   │   └── profile/       # User profile endpoints
│   ├── service/           # Business logic layer
│   │   ├── auth.go       # Authentication service
│   │   ├── buckets.go    # Bucket service
│   │   ├── credentials.go # Credential service
│   │   └── profile.go    # Profile service
│   ├── repository/        # Data access layer
│   │   ├── repository.go # Repository interfaces
│   │   ├── pg_repository.go # PostgreSQL implementation
│   │   └── sqlc/         # Generated type-safe SQL code
│   ├── middleware/        # HTTP middleware
│   │   ├── auth.go       # Authentication middleware
│   │   └── security.go   # Security headers
│   ├── storage/           # S3 client abstraction
│   │   ├── objectstore.go # S3 operations
│   │   └── postgres.go   # Legacy PostgreSQL code
│   ├── domain/            # Domain models
│   ├── security/          # Security utilities (legacy)
│   ├── config/            # Configuration management
│   └── logging/           # Logging setup
├── pkg/                    # Public reusable packages
│   ├── crypto/            # Password hashing & encryption utilities
│   └── jwt/               # JWT token management
├── migrations/             # Database migrations (golang-migrate)
└── queries/                # SQL queries for sqlc code generation

```

### Layer Responsibilities

- **API Layer** (`internal/api`): HTTP request/response handling, input validation, authentication
- **Service Layer** (`internal/service`): Business logic, orchestration, transaction management
- **Repository Layer** (`internal/repository`): Database operations using sqlc
- **Storage Layer** (`internal/storage`): S3-compatible storage operations using AWS SDK

## Tech Stack

- **Language**: Go 1.23+
- **Web Framework**: Chi router v5
- **Database**: PostgreSQL 15+ with pgx driver
- **Code Generation**: sqlc for type-safe SQL queries
- **Migrations**: golang-migrate for versioned database migrations
- **Authentication**: JWT tokens with Argon2id password hashing
- **Storage**: AWS SDK v2 for S3-compatible storage
- **Encryption**: AES-256-GCM for credential encryption
- **CLI**: Cobra for command-line interface

## Features

### Authentication & Authorization
- JWT-based authentication with access and refresh tokens
- Secure password hashing with Argon2id
- Session management with refresh token rotation
- User registration and login
- CLI-based user management (create, delete, list, password reset)

### Credential Management
- Encrypted storage of S3 credentials (access key, secret key)
- Support for multiple S3-compatible providers (AWS S3, MinIO, Wasabi, etc.)
- Connection testing before saving credentials
- AES-256-GCM encryption for sensitive data

### Bucket Management
- List, create, and delete S3 buckets
- Bucket size tracking and formatting
- Multi-credential support for different providers
- Metadata storage in PostgreSQL

### Object Operations
- List objects with folder navigation
- Upload files with progress tracking
- Download files and folders (as zip)
- Recursive search across all objects
- Folder creation and management
- Rename objects and folders (recursive)
- Delete objects and folders (recursive)
- Object metadata viewing
- Preview support for various file types

## Configuration

The application is configured via environment variables with the `BB_` prefix:

```bash
# Application
BB_APP_NAME=bucketbird-api
BB_ENV=development

# Server
BB_HTTP_PORT=8080
BB_HTTP_READ_TIMEOUT=30m
BB_HTTP_WRITE_TIMEOUT=30m

# Database
BB_DB_HOST=localhost
BB_DB_PORT=5432
BB_DB_NAME=bucketbird
BB_DB_USER=bucketbird
BB_DB_PASSWORD=bucketbird
# Or use DSN directly:
BB_DB_DSN=postgres://bucketbird:bucketbird@localhost:5432/bucketbird?sslmode=disable

# Security
BB_JWT_SECRET=your-jwt-secret-key-min-32-chars
BB_ENCRYPTION_KEY=your-encryption-key-must-be-32-bytes!!
BB_ACCESS_TOKEN_TTL=15m
BB_REFRESH_TOKEN_TTL=168h  # 7 days

# CORS
BB_ALLOWED_ORIGINS=http://localhost:5173,http://localhost:3000

# Features
BB_ALLOW_REGISTRATION=true
BB_ENABLE_DEMO_LOGIN=false
```

## Database Setup

### Using Docker Compose (Recommended)

```bash
# Start PostgreSQL
docker compose up -d postgres

# The database will be automatically initialized
```

### Manual Setup

```bash
# Install golang-migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -path migrations -database "postgres://bucketbird:bucketbird@localhost:5432/bucketbird?sslmode=disable" up
```

### Using sqlc

The project uses sqlc for type-safe SQL queries. To regenerate code after modifying queries:

```bash
# Install sqlc
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Generate code
sqlc generate
```

Query files are in `queries/` and generated code goes to `internal/repository/sqlc/`.

## Building

```bash
# Build the CLI application
go build -o bucketbird ./cmd/bucketbird

# Build with optimizations for production
CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o bucketbird ./cmd/bucketbird
```

## Running

### CLI Commands

The backend uses a CLI interface with subcommands:

```bash
# Start the HTTP API server
go run ./cmd/bucketbird serve

# Run database migrations
go run ./cmd/bucketbird migrate

# Create a new user
go run ./cmd/bucketbird user create \
  --email user@example.com \
  --password "SecurePass123!" \
  --first-name John \
  --last-name Doe

# List all users
go run ./cmd/bucketbird user list

# Reset a user's password
go run ./cmd/bucketbird user reset-password \
  --email user@example.com \
  --password "NewPassword123!"

# Delete a user
go run ./cmd/bucketbird user delete --email user@example.com
```

### Development

```bash
# Set environment variables
export BB_DB_DSN="postgres://bucketbird:bucketbird@localhost:5432/bucketbird?sslmode=disable"
export BB_JWT_SECRET="your-secret-key-at-least-32-characters-long"
export BB_ENCRYPTION_KEY="your-32-byte-encryption-key-here!!"
export BB_ALLOWED_ORIGINS="http://localhost:5173"

# Run the server
go run ./cmd/bucketbird serve
```

### Production

```bash
# Build the binary
CGO_ENABLED=0 GOOS=linux go build -o bucketbird ./cmd/bucketbird

# Run migrations
./bucketbird migrate

# Start the server
./bucketbird serve
```

### Docker

```bash
# Build image
docker build -t bucketbird-api .

# Run migrations
docker run --rm \
  -e BB_DB_DSN="postgres://..." \
  bucketbird-api migrate

# Run server
docker run -p 8080:8080 \
  -e BB_DB_DSN="postgres://..." \
  -e BB_JWT_SECRET="..." \
  -e BB_ENCRYPTION_KEY="..." \
  bucketbird-api serve
```

## API Endpoints

### Authentication
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login and get tokens
- `POST /api/v1/auth/refresh` - Refresh access token
- `POST /api/v1/auth/logout` - Logout and invalidate session

### Credentials
- `GET /api/v1/credentials` - List all credentials
- `POST /api/v1/credentials` - Create new credential
- `GET /api/v1/credentials/:id` - Get credential details
- `PUT /api/v1/credentials/:id` - Update credential
- `DELETE /api/v1/credentials/:id` - Delete credential
- `POST /api/v1/credentials/:id/test` - Test credential connection

### Buckets
- `GET /api/v1/buckets` - List all buckets
- `POST /api/v1/buckets` - Create new bucket
- `GET /api/v1/buckets/:id` - Get bucket details
- `PATCH /api/v1/buckets/:id` - Update bucket
- `DELETE /api/v1/buckets/:id` - Delete bucket

### Objects
- `GET /api/v1/buckets/:id/objects` - List objects (with prefix support)
- `GET /api/v1/buckets/:id/objects/search` - Search objects
- `POST /api/v1/buckets/:id/objects/upload` - Upload file (presigned URL)
- `GET /api/v1/buckets/:id/objects/download` - Download file or folder
- `POST /api/v1/buckets/:id/objects/folders` - Create folder
- `DELETE /api/v1/buckets/:id/objects` - Delete objects/folders
- `PATCH /api/v1/buckets/:id/objects/:key` - Rename/move object/folder
- `POST /api/v1/buckets/:id/objects/copy` - Copy object
- `GET /api/v1/buckets/:id/objects/metadata` - Get object metadata
- `POST /api/v1/buckets/:id/objects/presign` - Generate presigned URL

### Profile
- `GET /api/v1/profile` - Get user profile
- `PATCH /api/v1/profile` - Update profile

## Security

### Authentication
- JWT tokens with configurable expiration (default: 15m access, 7d refresh)
- Refresh token rotation for enhanced security
- Secure password hashing with Argon2id (memory: 64MB, time: 3, parallelism: 2)
- Session management with database-backed refresh tokens
- SHA-256 hashed refresh tokens stored in database

### Encryption
- AES-256-GCM encryption for S3 credentials at rest
- Unique nonce per encryption operation
- Environment-based encryption key management (32-byte key required)

### HTTP Security Headers
- CORS configuration with configurable allowed origins
- Security headers middleware (X-Frame-Options, X-Content-Type-Options, etc.)
- Request ID tracking for observability
- Panic recovery middleware
- Authentication middleware for protected routes

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...
```

### Code Quality

```bash
# Format code
go fmt ./...

# Lint code (requires golangci-lint)
golangci-lint run

# Vet code
go vet ./...
```

### Database Migrations

```bash
# Create a new migration
migrate create -ext sql -dir migrations -seq migration_name

# Check migration version
migrate -path migrations -database "$DB_DSN" version

# Force a specific version (use with caution)
migrate -path migrations -database "$DB_DSN" force VERSION
```

## Troubleshooting

### Database Connection Issues

```bash
# Test database connection
psql "$DB_DSN"

# Check if migrations are up to date
migrate -path migrations -database "$DB_DSN" version
```

### S3 Connection Issues

- Verify credentials are correct
- Check endpoint URL format (should include protocol: http:// or https://)
- Ensure SSL setting matches the endpoint
- Test credentials using the `/credentials/:id/test` endpoint

### Token Issues

- Ensure `JWT_SECRET` is at least 32 characters
- Check token expiration times are reasonable
- Verify system clock is synchronized

## Recent Changes (v0.1.0)

The backend was recently refactored to follow clean architecture principles:

### Architecture Improvements
- **CLI Interface**: Migrated from standalone binaries to a unified CLI with Cobra
- **Service Layer**: Introduced dedicated service layer for business logic
- **Repository Pattern**: Interface-based repository layer for better testability
- **Type-Safe SQL**: Migrated to sqlc for compile-time SQL validation
- **Versioned Migrations**: Switched from runtime schema initialization to golang-migrate
- **Password Security**: Upgraded from bcrypt to Argon2id for password hashing

### What Changed
- Main entry point moved from `cmd/bucketbird-api/main.go` to `cmd/bucketbird/main.go`
- HTTP handlers now use Chi router with proper middleware stack
- Database queries are now generated by sqlc from `queries/*.sql` files
- Configuration uses consistent `BB_` prefix for all environment variables
- User management now available via CLI commands instead of separate binaries

### Migration Notes
If upgrading from a previous version:
1. Update environment variable names to use `BB_` prefix
2. Run database migrations: `./bucketbird migrate`
3. Update any scripts to use new CLI commands
4. Rehash existing passwords will happen automatically on next login (Argon2id migration)

## Contributing

1. Follow the existing code structure and patterns
2. Use sqlc for all database queries (modify `queries/*.sql`, then run `sqlc generate`)
3. Write tests for new functionality
4. Follow Go best practices and idioms
5. Run migrations for schema changes (`./bucketbird migrate`)
6. Update documentation as needed

## Version History

See [CHANGELOG.md](../CHANGELOG.md) for detailed release notes.

**Current Version**: v0.1.0 (Released 2025-11-08)

## License

This project is licensed under the MIT License. See LICENSE file in the repository root.
