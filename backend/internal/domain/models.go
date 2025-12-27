package domain

import (
	"io"
	"time"
)

type Bucket struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Region             string    `json:"region"`
	Description        *string   `json:"description,omitempty"`
	Size               string    `json:"size"`
	CredentialID       string    `json:"credentialId"`
	CredentialName     string    `json:"credentialName"`
	CredentialProvider string    `json:"credentialProvider"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type BucketObject struct {
	Key          string    `json:"key"`
	Name         string    `json:"name"`
	Kind         string    `json:"kind"`
	LastModified time.Time `json:"lastModified"`
	Size         string    `json:"size"`
	Icon         string    `json:"icon"`
	IconColor    string    `json:"iconColor"`
}

type CreateBucketInput struct {
	Name         string  `json:"name"`
	Region       string  `json:"region"`
	CredentialID string  `json:"credentialId"`
	Description  *string `json:"description,omitempty"`
}

type UpdateBucketInput struct {
	Description *string `json:"description,omitempty"`
}

type CreateFolderInput struct {
	Name   string  `json:"name"`
	Prefix *string `json:"prefix,omitempty"`
}

type DeleteObjectsInput struct {
	Keys []string `json:"keys"`
}

type DeleteObjectsResult struct {
	Deleted int `json:"deleted"`
}

type RenameObjectInput struct {
	SourceKey      string `json:"sourceKey"`
	DestinationKey string `json:"destinationKey"`
}

type RenameObjectResult struct {
	ObjectsMoved int `json:"objectsMoved"`
}

type CopyObjectInput struct {
	SourceKey      string `json:"sourceKey"`
	DestinationKey string `json:"destinationKey"`
}

type CopyObjectResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type Credential struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Provider  string    `json:"provider"`
	Region    string    `json:"region"`
	Endpoint  string    `json:"endpoint"`
	UseSSL    bool      `json:"useSSL"`
	Status    string    `json:"status"`
	Logo      *string   `json:"logo,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateCredentialInput struct {
	Name      string  `json:"name"`
	Provider  string  `json:"provider"`
	Region    string  `json:"region"`
	Endpoint  string  `json:"endpoint"`
	AccessKey string  `json:"accessKey"`
	SecretKey string  `json:"secretKey"`
	UseSSL    bool    `json:"useSSL"`
	Logo      *string `json:"logo,omitempty"`
}

type UpdateCredentialInput struct {
	Name      string  `json:"name"`
	Provider  string  `json:"provider"`
	Region    string  `json:"region"`
	Endpoint  string  `json:"endpoint"`
	AccessKey string  `json:"accessKey"`
	SecretKey string  `json:"secretKey"`
	UseSSL    bool    `json:"useSSL"`
	Logo      *string `json:"logo,omitempty"`
}

type TestCredentialResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type Profile struct {
	ID        string    `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type UpdateProfileInput struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

type PresignResponse struct {
	URL    string `json:"url"`
	Method string `json:"method"`
}

type ProxyObjectResponse struct {
	Body          io.ReadCloser
	ContentType   string
	ContentLength int64
}

type ObjectMetadata struct {
	Key          string    `json:"key"`
	SizeBytes    int64     `json:"sizeBytes"`
	Size         string    `json:"size"`
	LastModified time.Time `json:"lastModified"`
	ETag         string    `json:"etag"`
	ContentType  string    `json:"contentType"`
	StorageClass string    `json:"storageClass"`
}

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
