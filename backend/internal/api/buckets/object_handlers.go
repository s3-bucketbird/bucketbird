package buckets

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"bucketbird/backend/internal/middleware"
	"bucketbird/backend/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const maxMultipartUploadSize int64 = 5 * 1024 * 1024 * 1024 // 5 GiB

type YouTubeImportRequest struct {
	URL               string `json:"url"`
	DestinationPrefix string `json:"destinationPrefix"`
}

// ListObjects lists objects in a bucket
func (h *Handler) ListObjects(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.respondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	bucketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, "Invalid bucket ID", http.StatusBadRequest)
		return
	}

	prefix := r.URL.Query().Get("prefix")

	objects, err := h.bucketService.ListObjects(r.Context(), bucketID, userID, prefix, h.encryptionKey)
	if err != nil {
		h.logger.Error("failed to list objects", slog.Any("error", err))
		h.respondError(w, "Failed to list objects", http.StatusInternalServerError)
		return
	}

	h.respondJSON(w, map[string]interface{}{"objects": objects}, http.StatusOK)
}

// SearchObjects searches for objects
func (h *Handler) SearchObjects(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.respondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	bucketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, "Invalid bucket ID", http.StatusBadRequest)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		h.respondJSON(w, map[string]interface{}{"objects": []service.BucketObject{}}, http.StatusOK)
		return
	}

	objects, err := h.bucketService.SearchObjects(r.Context(), bucketID, userID, query, h.encryptionKey)
	if err != nil {
		h.logger.Error("failed to search objects", slog.Any("error", err))
		h.respondError(w, "Failed to search objects", http.StatusInternalServerError)
		return
	}

	h.respondJSON(w, map[string]interface{}{"objects": objects}, http.StatusOK)
}

// UploadObject uploads a file to a bucket
func (h *Handler) UploadObject(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.respondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	bucketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, "Invalid bucket ID", http.StatusBadRequest)
		return
	}

	// Limit upload size to prevent resource exhaustion
	r.Body = http.MaxBytesReader(w, r.Body, maxMultipartUploadSize)

	// Parse multipart form
	reader, err := r.MultipartReader()
	if err != nil {
		h.respondError(w, "Failed to read multipart form", http.StatusBadRequest)
		return
	}

	var key string
	var file io.Reader
	var contentType string

	// Read form parts
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			h.respondError(w, "Failed to read form part", http.StatusBadRequest)
			return
		}

		formName := part.FormName()
		switch formName {
		case "key":
			keyBytes, err := io.ReadAll(part)
			if err != nil {
				h.respondError(w, "Failed to read key", http.StatusBadRequest)
				return
			}
			key = string(keyBytes)
		case "file":
			contentType = part.Header.Get("Content-Type")
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			file = part
			goto uploadFile
		default:
			part.Close()
		}
	}

uploadFile:
	if strings.TrimSpace(key) == "" {
		h.respondError(w, "key is required", http.StatusBadRequest)
		return
	}

	if file == nil {
		h.respondError(w, "file is required", http.StatusBadRequest)
		return
	}

	if err := h.bucketService.UploadObject(r.Context(), bucketID, userID, key, file, contentType, h.encryptionKey); err != nil {
		h.logger.Error("failed to upload object", slog.Any("error", err))
		h.respondError(w, fmt.Sprintf("Upload failed: %v", err), http.StatusInternalServerError)
		return
	}

	h.respondJSON(w, map[string]interface{}{
		"success": true,
		"message": "File uploaded successfully",
	}, http.StatusOK)
}

// ImportYouTube handles importing YouTube videos or playlists into a bucket
func (h *Handler) ImportYouTube(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.respondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	bucketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, "Invalid bucket ID", http.StatusBadRequest)
		return
	}

	var req YouTubeImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.URL) == "" {
		h.respondError(w, "YouTube URL is required", http.StatusBadRequest)
		return
	}

	stream := r.URL.Query().Get("stream") == "1"
	if stream {
		flusher, ok := w.(http.Flusher)
		if !ok {
			h.respondError(w, "Streaming is not supported on this server", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Cache-Control", "no-cache")

		encoder := json.NewEncoder(w)
		progressFn := func(progress service.YouTubeImportProgress) {
			if err := encoder.Encode(map[string]interface{}{"progress": progress}); err != nil {
				h.logger.Warn("failed to stream progress", slog.Any("error", err))
			}
			flusher.Flush()
		}

		result, importErr := h.bucketService.ImportYouTube(
			r.Context(),
			bucketID,
			userID,
			service.YouTubeImportInput{
				URL:               req.URL,
				DestinationPrefix: req.DestinationPrefix,
			},
			h.encryptionKey,
			progressFn,
		)
		if importErr != nil {
			h.logger.Error("youtube import failed", slog.Any("error", importErr))
			encoder.Encode(map[string]interface{}{
				"error": fmt.Sprintf("YouTube import failed: %v", importErr),
			})
			flusher.Flush()
			return
		}

		encoder.Encode(map[string]interface{}{"result": result})
		flusher.Flush()
		return
	}

	result, importErr := h.bucketService.ImportYouTube(
		r.Context(),
		bucketID,
		userID,
		service.YouTubeImportInput{
			URL:               req.URL,
			DestinationPrefix: req.DestinationPrefix,
		},
		h.encryptionKey,
		nil,
	)
	if importErr != nil {
		h.logger.Error("youtube import failed", slog.Any("error", importErr))
		h.respondError(w, fmt.Sprintf("YouTube import failed: %v", importErr), http.StatusBadRequest)
		return
	}

	h.respondJSON(w, map[string]interface{}{"result": result}, http.StatusOK)
}

// DownloadObject downloads an object from a bucket
func (h *Handler) DownloadObject(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.respondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	bucketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, "Invalid bucket ID", http.StatusBadRequest)
		return
	}

	key := r.URL.Query().Get("key")
	if strings.TrimSpace(key) == "" {
		h.respondError(w, "key is required", http.StatusBadRequest)
		return
	}

	// Check if it's a folder (ends with /)
	if strings.HasSuffix(key, "/") {
		reader, filename, err := h.bucketService.ZipFolder(r.Context(), bucketID, userID, key, h.encryptionKey)
		if err != nil {
			h.logger.Error("failed to zip folder", slog.Any("error", err))
			h.respondError(w, fmt.Sprintf("Failed to prepare folder download: %v", err), http.StatusInternalServerError)
			return
		}
		defer reader.Close()

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		w.WriteHeader(http.StatusOK)
		if _, err := io.Copy(w, reader); err != nil {
			h.logger.Error("failed to stream zip", slog.Any("error", err))
		}
		return
	}

	// Regular file download
	obj, err := h.bucketService.ProxyObject(r.Context(), bucketID, userID, key, h.encryptionKey)
	if err != nil {
		h.logger.Error("failed to get object", slog.Any("error", err))
		h.respondError(w, fmt.Sprintf("Failed to fetch object: %v", err), http.StatusInternalServerError)
		return
	}
	defer obj.Body.Close()

	w.Header().Set("Content-Type", obj.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", obj.ContentLength))
	w.WriteHeader(http.StatusOK)
	io.Copy(w, obj.Body)
}

// PresignObject generates a presigned URL for an object
func (h *Handler) PresignObject(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.respondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	bucketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, "Invalid bucket ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Key         string  `json:"key"`
		Method      string  `json:"method"`
		Expires     *int64  `json:"expiresInSeconds"`
		ContentType *string `json:"contentType"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var expires time.Duration
	if req.Expires != nil {
		expires = time.Duration(*req.Expires) * time.Second
	} else {
		expires = 15 * time.Minute // default
	}

	presigned, err := h.bucketService.PresignObject(r.Context(), bucketID, userID, service.PresignInput{
		Key:         req.Key,
		Method:      req.Method,
		Expires:     expires,
		ContentType: req.ContentType,
	}, h.encryptionKey)
	if err != nil {
		h.logger.Error("failed to presign object", slog.Any("error", err))
		h.respondError(w, "Failed to presign object", http.StatusInternalServerError)
		return
	}

	h.respondJSON(w, map[string]interface{}{"presign": presigned}, http.StatusOK)
}

// GetObjectMetadata retrieves metadata for an object
func (h *Handler) GetObjectMetadata(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.respondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	bucketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, "Invalid bucket ID", http.StatusBadRequest)
		return
	}

	key := r.URL.Query().Get("key")
	if strings.TrimSpace(key) == "" {
		h.respondError(w, "key is required", http.StatusBadRequest)
		return
	}

	metadata, err := h.bucketService.GetObjectMetadata(r.Context(), bucketID, userID, key, h.encryptionKey)
	if err != nil {
		h.logger.Error("failed to get object metadata", slog.Any("error", err))
		h.respondError(w, "Failed to get object metadata", http.StatusInternalServerError)
		return
	}

	h.respondJSON(w, map[string]interface{}{"metadata": metadata}, http.StatusOK)
}

// CreateFolder creates a folder in a bucket
func (h *Handler) CreateFolder(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.respondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	bucketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, "Invalid bucket ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Name   string  `json:"name"`
		Prefix *string `json:"prefix,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.bucketService.CreateFolder(r.Context(), bucketID, userID, req.Name, req.Prefix, h.encryptionKey)
	if err != nil {
		h.logger.Error("failed to create folder", slog.Any("error", err))
		h.respondError(w, "Failed to create folder", http.StatusInternalServerError)
		return
	}

	h.respondJSON(w, map[string]interface{}{"folder": result}, http.StatusCreated)
}

// DeleteObjects deletes objects from a bucket
func (h *Handler) DeleteObjects(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.respondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	bucketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, "Invalid bucket ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Keys []string `json:"keys"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.bucketService.DeleteObjects(r.Context(), bucketID, userID, req.Keys, h.encryptionKey)
	if err != nil {
		h.logger.Error("failed to delete objects", slog.Any("error", err))
		h.respondError(w, "Failed to delete objects", http.StatusInternalServerError)
		return
	}

	h.respondJSON(w, map[string]interface{}{"result": result}, http.StatusOK)
}

// RenameObject renames an object
func (h *Handler) RenameObject(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.respondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	bucketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, "Invalid bucket ID", http.StatusBadRequest)
		return
	}

	var req struct {
		SourceKey      string `json:"sourceKey"`
		DestinationKey string `json:"destinationKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.bucketService.RenameObject(r.Context(), bucketID, userID, req.SourceKey, req.DestinationKey, h.encryptionKey)
	if err != nil {
		h.logger.Error("failed to rename object", slog.Any("error", err))
		h.respondError(w, "Failed to rename object", http.StatusInternalServerError)
		return
	}

	h.respondJSON(w, map[string]interface{}{"result": result}, http.StatusOK)
}

// CopyObject copies an object
func (h *Handler) CopyObject(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.respondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	bucketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.respondError(w, "Invalid bucket ID", http.StatusBadRequest)
		return
	}

	var req struct {
		SourceKey      string `json:"sourceKey"`
		DestinationKey string `json:"destinationKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.bucketService.CopyObject(r.Context(), bucketID, userID, req.SourceKey, req.DestinationKey, h.encryptionKey)
	if err != nil {
		h.logger.Error("failed to copy object", slog.Any("error", err))
		h.respondError(w, "Failed to copy object", http.StatusInternalServerError)
		return
	}

	h.respondJSON(w, map[string]interface{}{"result": result}, http.StatusOK)
}
