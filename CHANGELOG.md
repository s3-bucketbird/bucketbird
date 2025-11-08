# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2025-11-08

### Added
- **Authentication System**
  - User registration and login with JWT tokens
  - Password hashing with Argon2id
  - Password reset functionality via CLI
  - Profile management endpoints

- **S3 Credential Management**
  - Multi-provider S3 credential storage (AWS, MinIO, DigitalOcean, Wasabi, Backblaze B2, Cloudflare R2)
  - AES-256-GCM encryption for stored credentials
  - CRUD operations for credentials
  - Connection testing and validation

- **Bucket Operations**
  - List all buckets across credentials
  - Create new buckets with region selection
  - Delete buckets
  - Bucket statistics and metadata

- **Object Management**
  - Upload files to S3 buckets
  - Download files from S3
  - Delete objects and folders
  - List objects with pagination
  - File preview support (images, text, PDFs)
  - Create folders
  - Copy/move objects between locations

- **Frontend Application**
  - Modern React UI with Tailwind CSS
  - Responsive design
  - File browser interface
  - Drag-and-drop upload support
  - Real-time upload/download progress
  - Image preview gallery

- **Backend Infrastructure**
  - Go 1.23+ REST API with Chi router
  - PostgreSQL database with type-safe queries (sqlc)
  - Database migrations system
  - Docker and Docker Compose support
  - Kamal deployment configuration
  - Health check endpoints
  - Comprehensive logging

- **Security Features**
  - JWT-based authentication
  - Encrypted credential storage
  - Rate limiting
  - CORS configuration
  - Secure password hashing
  - SQL injection prevention via prepared statements

### Security
- Implemented credential encryption at rest using AES-256-GCM
- Added secure password hashing with Argon2id
- Configured CORS for production environments

### Infrastructure
- Multi-stage Docker builds for backend and frontend
- Kamal 2 deployment configuration for production
- PostgreSQL 15 database setup
- MinIO integration for local S3 testing
- Nginx configuration for frontend serving

[unreleased]: https://github.com/s3-bucketbird/bucketbird/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/s3-bucketbird/bucketbird/releases/tag/v0.1.0
