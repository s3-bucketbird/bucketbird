# BucketBird

A full-stack S3 storage management application with multi-credential support. BucketBird allows you to manage multiple S3-compatible storage providers (AWS S3, MinIO, DigitalOcean Spaces, etc.) from a single unified interface.

## Features

- **Multi-Credential Support**: Connect to multiple S3-compatible storage providers simultaneously
- **User Authentication**: Secure JWT-based authentication with refresh tokens
- **Bucket Management**: Create, view, update, and delete S3 buckets
- **Object Operations**: Upload, download, preview, rename, copy, and delete files
- **Encrypted Credentials**: S3 credentials are encrypted at rest using AES-256-GCM
- **File Preview**: Built-in preview for images, videos, audio, PDFs, and text files
- **Modern UI**: Responsive React interface with Tailwind CSS
- **Docker Deployment**: Complete containerized setup with Docker Compose

## Architecture

- **Backend**: Go HTTP API with Chi router, PostgreSQL for metadata, encrypted credential storage
- **Frontend**: React 19 SPA with Vite, TypeScript, Tailwind CSS, and TanStack Query
- **Storage**: Supports any S3-compatible service (MinIO, AWS S3, DigitalOcean, Wasabi, etc.)
- **Database**: PostgreSQL 15 for user accounts, sessions, bucket metadata, and credentials

## Prerequisites

- Go 1.23+ (for local backend development)
- Node.js 20+ (for local frontend development)
- Docker & Docker Compose (for full-stack runtime)

## Quick Start

### Using Docker Compose (Recommended)

```bash
docker compose up --build
```

This starts four containers:

- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:8080
- **PostgreSQL**: `localhost:5432` (user: `bucketbird`, password: `bucketbird`)
- **MinIO**: S3 API on `localhost:9000`, Console on http://localhost:9001 (user: `minioadmin`, password: `minioadmin`)

### First Steps

1. Open http://localhost:3000 in your browser
2. Register a new account or use demo credentials (if enabled)
3. Navigate to Settings → S3 Credentials
4. Add your first credential:
   - **Name**: Local MinIO
   - **Provider**: MinIO
   - **Endpoint**: http://minio:9000
   - **Region**: us-east-1
   - **Access Key**: minioadmin
   - **Secret Key**: minioadmin
   - **Use SSL**: No
5. Create a bucket and start uploading files!

## Local Development

### Backend

```bash
cd backend
go run ./cmd/bucketbird serve
```

The backend uses a CLI interface with subcommands. Available commands:

- `serve` - Start the HTTP API server
- `migrate` - Run database migrations
- `user create` - Create a new user account
- `user delete` - Delete a user account
- `user list` - List all users
- `user reset-password` - Reset a user's password

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BB_APP_NAME` | `bucketbird-api` | Service name used in logs |
| `BB_ENV` | `development` | Environment (development/production) |
| `BB_HTTP_PORT` | `8080` | Port the API listens on |
| `BB_ALLOWED_ORIGINS` | `*` | Comma-separated list of CORS origins |
| `BB_DB_HOST` | `postgres` | Database host |
| `BB_DB_PORT` | `5432` | Database port |
| `BB_DB_NAME` | `bucketbird` | Database name |
| `BB_DB_USER` | `bucketbird` | Database user |
| `BB_DB_PASSWORD` | `bucketbird` | Database password |
| `BB_JWT_SECRET` | _required_ | JWT signing secret (keep secret!) |
| `BB_ENCRYPTION_KEY` | _required_ | 32-byte key for credential encryption |
| `BB_ACCESS_TOKEN_TTL` | `15m` | Access token lifetime |
| `BB_REFRESH_TOKEN_TTL` | `7d` | Refresh token lifetime |
| `BB_ALLOW_REGISTRATION` | `true` | Enable/disable self-service registration |
| `BB_ENABLE_DEMO_LOGIN` | `false` | Enable demo account for testing |

### API Endpoints

**Authentication**
- `POST /api/v1/auth/register` - Register a new user
- `POST /api/v1/auth/login` - Login with email/password
- `POST /api/v1/auth/logout` - Logout and invalidate session
- `POST /api/v1/auth/refresh` - Refresh access token

**Credentials**
- `GET /api/v1/credentials` - List user's S3 credentials
- `POST /api/v1/credentials` - Add new S3 credential
- `GET /api/v1/credentials/:id` - Get credential details
- `PUT /api/v1/credentials/:id` - Update credential
- `DELETE /api/v1/credentials/:id` - Delete credential
- `POST /api/v1/credentials/:id/test` - Test credential connection

**Buckets**
- `GET /api/v1/buckets` - List user's buckets
- `POST /api/v1/buckets` - Create new bucket
- `GET /api/v1/buckets/:id` - Get bucket details
- `PATCH /api/v1/buckets/:id` - Update bucket
- `DELETE /api/v1/buckets/:id` - Delete bucket
- `GET /api/v1/buckets/:id/objects` - List objects in bucket
- `POST /api/v1/buckets/:id/objects` - Upload file (presigned URL)
- `DELETE /api/v1/buckets/:id/objects` - Delete files
- `PATCH /api/v1/buckets/:id/objects/:key` - Rename/move file
- `POST /api/v1/buckets/:id/objects/copy` - Copy file

**Profile**
- `GET /api/v1/profile` - Get current user profile
- `PATCH /api/v1/profile` - Update user profile

### Frontend

```bash
cd frontend
npm install
npm run dev
```

The development server runs on http://localhost:5173. The frontend automatically connects to the backend API at `http://localhost:8080`.

**Available Scripts:**
- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run lint` - Run ESLint
- `npm run preview` - Preview production build

## User Management

### Creating Users via CLI

If self-service registration is disabled (`BB_ALLOW_REGISTRATION=false`), administrators can create accounts using the CLI:

```bash
# Using Docker
docker compose exec backend /app/bucketbird user create \
  --email alex@example.com \
  --password 'SecurePass123!' \
  --first-name Alex \
  --last-name Doe

# Local development
cd backend
go run ./cmd/bucketbird user create \
  --email alex@example.com \
  --password 'SecurePass123!' \
  --first-name Alex \
  --last-name Doe
```

### Resetting Passwords

Administrators can reset user passwords:

```bash
# Using Docker
docker compose exec backend /app/bucketbird user reset-password \
  --email user@example.com \
  --password "NewSecurePass123!"

# Local development
cd backend
go run ./cmd/bucketbird user reset-password \
  --email user@example.com \
  --password "NewSecurePass123!"
```

### Listing Users

View all registered users:

```bash
# Using Docker
docker compose exec backend /app/bucketbird user list

# Local development
cd backend
go run ./cmd/bucketbird user list
```

### Deleting Users

Remove a user account:

```bash
# Using Docker
docker compose exec backend /app/bucketbird user delete \
  --email user@example.com

# Local development
cd backend
go run ./cmd/bucketbird user delete \
  --email user@example.com
```

## Database Migrations

The application uses database migrations to manage schema changes:

```bash
# Using Docker
docker compose exec backend /app/bucketbird migrate

# Local development
cd backend
go run ./cmd/bucketbird migrate
```

Migrations are automatically run on application startup.

## Security Features

- **Password Hashing**: Argon2id for secure password storage
- **Credential Encryption**: AES-256-GCM encryption for S3 credentials at rest
- **JWT Authentication**: Secure token-based authentication with refresh tokens
- **Token Rotation**: Automatic refresh token rotation on use
- **Session Management**: Database-backed session tracking
- **CORS Protection**: Configurable allowed origins
- **Input Validation**: Comprehensive validation on all user inputs

## Project Structure

```
backend/
├── cmd/bucketbird/           # Main application entry point
│   ├── main.go              # CLI entry point
│   └── cmd/                 # CLI commands (serve, migrate, user)
├── internal/
│   ├── api/                 # HTTP handlers by domain
│   │   ├── auth/           # Authentication endpoints
│   │   ├── buckets/        # Bucket management endpoints
│   │   ├── credentials/    # S3 credential endpoints
│   │   └── profile/        # User profile endpoints
│   ├── config/             # Configuration management
│   ├── domain/             # Domain models
│   ├── middleware/         # HTTP middleware (auth, security, logging)
│   ├── repository/         # Data access layer
│   │   └── sqlc/          # Generated SQL queries
│   ├── service/            # Business logic layer
│   └── storage/            # S3 object storage client
├── pkg/
│   ├── crypto/             # Password hashing & encryption
│   └── jwt/                # JWT token management
└── migrations/             # Database migration files

frontend/
├── src/
│   ├── api/                # API client
│   ├── components/         # React components
│   │   ├── auth/          # Authentication components
│   │   ├── buckets/       # Bucket management UI
│   │   ├── files/         # File upload/preview components
│   │   └── layout/        # Layout components
│   ├── contexts/          # React contexts (Auth)
│   ├── hooks/             # Custom React hooks
│   ├── pages/             # Page components
│   └── types/             # TypeScript type definitions
└── public/                # Static assets

config/                     # Kamal deployment configs
docker-compose.yml          # Docker compose orchestration
```

## Technology Stack

**Backend:**
- Go 1.23
- Chi Router - HTTP routing
- PostgreSQL 15 - Database
- sqlc - Type-safe SQL
- AWS SDK for Go v2 - S3 operations
- Argon2 - Password hashing
- AES-256-GCM - Credential encryption

**Frontend:**
- React 19
- TypeScript
- Vite - Build tool
- Tailwind CSS - Styling
- TanStack Query - Server state management
- React Router - Client-side routing

**Infrastructure:**
- Docker & Docker Compose
- MinIO - S3-compatible storage (for local development)

## Supported S3 Providers

BucketBird supports any S3-compatible storage provider:

- **AWS S3** - Amazon's object storage
- **MinIO** - Self-hosted S3-compatible storage
- **DigitalOcean Spaces** - DigitalOcean's object storage
- **Wasabi** - Hot cloud storage
- **Backblaze B2** - Low-cost cloud storage
- **Cloudflare R2** - Zero egress fees storage
- **Custom S3-compatible services**

## Roadmap

- [ ] Trash/restore functionality for deleted files
- [ ] File sharing with public/private links
- [ ] User activity tracking and audit logs
- [ ] Bucket sharing and collaboration
- [ ] Advanced search and filtering
- [ ] Batch operations for files
- [ ] Storage usage analytics and quotas
- [ ] Email notifications
- [ ] Two-factor authentication (2FA)
- [ ] API rate limiting per user
- [ ] Automated testing (unit, integration, E2E)

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

This project is licensed under the MIT License.
