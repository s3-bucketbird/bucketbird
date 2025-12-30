package profile

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"bucketbird/backend/internal/middleware"
	"bucketbird/backend/internal/service"
)

type Handler struct {
	profileService *service.ProfileService
	logger         *slog.Logger
}

func NewHandler(profileService *service.ProfileService, logger *slog.Logger) *Handler {
	return &Handler{
		profileService: profileService,
		logger:         logger,
	}
}

type ProfileDTO struct {
	ID        string  `json:"id"`
	FirstName string  `json:"firstName"`
	LastName  string  `json:"lastName"`
	Email     string  `json:"email"`
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.respondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	profile, err := h.profileService.Get(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get profile", slog.Any("error", err))
		h.respondError(w, "Failed to get profile", http.StatusInternalServerError)
		return
	}

	h.respondJSON(w, map[string]interface{}{"profile": ProfileDTO{
		ID:        profile.ID.String(),
		FirstName: profile.FirstName,
		LastName:  profile.LastName,
		Email:     profile.Email,
	}}, http.StatusOK)
}

type UpdateProfileRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.respondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	profile, err := h.profileService.Update(r.Context(), userID, service.UpdateProfileInput{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
	})
	if err != nil {
		if errors.Is(err, service.ErrEmailAlreadyInUse) {
			h.respondError(w, "Email already in use", http.StatusConflict)
			return
		}
		h.logger.Error("failed to update profile", slog.Any("error", err))
		h.respondError(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	h.respondJSON(w, map[string]interface{}{"profile": ProfileDTO{
		ID:        profile.ID.String(),
		FirstName: profile.FirstName,
		LastName:  profile.LastName,
		Email:     profile.Email,
	}}, http.StatusOK)
}

type UpdatePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (h *Handler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.respondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.profileService.UpdatePassword(r.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			h.respondError(w, "Current password is incorrect", http.StatusBadRequest)
			return
		}
		h.logger.Error("failed to update password", slog.Any("error", err))
		h.respondError(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type GetYouTubeCookieResponse struct {
	Cookie *string `json:"cookie"`
}

func (h *Handler) GetYouTubeCookie(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.respondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	cookie, err := h.profileService.GetYouTubeCookie(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get youtube cookie", slog.Any("error", err))
		h.respondError(w, "Failed to get YouTube cookie", http.StatusInternalServerError)
		return
	}

	h.respondJSON(w, GetYouTubeCookieResponse{Cookie: cookie}, http.StatusOK)
}

type UpdateYouTubeCookieRequest struct {
	Cookie *string `json:"cookie"`
}

func (h *Handler) UpdateYouTubeCookie(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.respondError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req UpdateYouTubeCookieRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.profileService.UpdateYouTubeCookie(r.Context(), userID, req.Cookie); err != nil {
		h.logger.Error("failed to update youtube cookie", slog.Any("error", err))
		h.respondError(w, "Failed to update YouTube cookie", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) respondJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", slog.Any("error", err))
	}
}

func (h *Handler) respondError(w http.ResponseWriter, message string, status int) {
	h.respondJSON(w, map[string]string{"error": message}, status)
}
