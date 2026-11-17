package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"unesarent/backend/internal/user"
)

type AuthHandler struct {
	svc *user.Service
}

func NewAuthHandler(svc *user.Service) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type authResponse struct {
	Token     string          `json:"token"`
	TokenType string          `json:"token_type"`
	ExpiresAt time.Time       `json:"expires_at"`
	User      user.PublicUser `json:"user"`
}

func newAuthResponse(result user.AuthResult) authResponse {
	return authResponse{
		Token:     result.Token,
		TokenType: "Bearer",
		ExpiresAt: result.ExpiresAt,
		User:      result.User,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req user.RegisterInput
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Body request harus berupa JSON yang valid")
		return
	}

	result, err := h.svc.Register(r.Context(), req)
	if err != nil {
		h.handleError(w, "register", err)
		return
	}

	writeJSON(w, http.StatusCreated, newAuthResponse(result))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req user.LoginInput
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Body request harus berupa JSON yang valid")
		return
	}

	result, err := h.svc.Login(r.Context(), req)
	if err != nil {
		h.handleError(w, "login", err)
		return
	}

	writeJSON(w, http.StatusOK, newAuthResponse(result))
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "invalid_token", "Token tidak valid")
		return
	}

	current, err := h.svc.Me(r.Context(), userID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "user_not_found", "User tidak ditemukan, silakan login ulang")
			return
		}
		h.handleError(w, "me", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"user": current})
}

func Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) handleError(w http.ResponseWriter, operation string, err error) {
	var validationErr *user.ValidationError
	switch {
	case errors.As(err, &validationErr):
		writeValidationError(w, validationErr.Fields)

	case errors.Is(err, user.ErrEmailTaken):
		writeError(w, http.StatusConflict, "email_taken", "Email sudah terdaftar")

	case errors.Is(err, user.ErrNIMTaken):
		writeError(w, http.StatusConflict, "nim_taken", "NIM sudah terdaftar")

	case errors.Is(err, user.ErrInvalidNIM):
		writeValidationError(w, map[string]string{
			"nim": "NIM tidak cocok dengan fakultas/program studi Fakultas Teknik",
		})
	
	case errors.Is(err, user.ErrAccountSuspended):
		writeError(w, http.StatusForbidden, "account_suspended", "Akun Anda ditangguhkan, hubungi admin")

	case errors.Is(err, user.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Email atau password salah")

	case errors.Is(err, user.ErrNotFound):
		writeError(w, http.StatusUnauthorized, "user_not_found", "User tidak ditemukan")

	default:
		var syntaxErr *json.SyntaxError
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &syntaxErr) || errors.As(err, &typeErr) {
			writeError(w, http.StatusBadRequest, "invalid_body", "Body request harus berupa JSON yang valid")
			return
		}
		slog.Error("kesalahan internal", "operation", operation, "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan pada server")
	}
}
