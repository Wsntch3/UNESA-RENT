package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"unesarent/backend/internal/authotp"
	"unesarent/backend/internal/user"
)

type AuthHandler struct {
	svc          *user.Service
	trustedProxy bool
}

func NewAuthHandler(svc *user.Service, trustedProxy bool) *AuthHandler {
	return &AuthHandler{svc: svc, trustedProxy: trustedProxy}
}

type authResponse struct {
	Token       string          `json:"token"`
	TokenType   string          `json:"token_type"`
	ExpiresAt   time.Time       `json:"expires_at"`
	User        user.PublicUser `json:"user"`
	OTPRequired bool            `json:"otp_required,omitempty"`
	Challenge   string          `json:"challenge,omitempty"`
}

func newAuthResponse(result user.AuthResult) authResponse {
	return authResponse{
		Token:       result.Token,
		TokenType:   "Bearer",
		ExpiresAt:   result.ExpiresAt,
		User:        result.User,
		OTPRequired: result.OTPRequired,
		Challenge:   result.Challenge,
	}
}

type otpRequestBody struct {
	Email   string `json:"email"`
	Purpose string `json:"purpose"`
}

type otpVerifyBody struct {
	Email   string `json:"email"`
	Purpose string `json:"purpose"`
	Code    string `json:"code"`
}

type loginOTPBody struct {
	Challenge string `json:"challenge"`
	Code      string `json:"code"`
}

type resetPasswordBody struct {
	Email       string `json:"email"`
	Ticket      string `json:"ticket"`
	NewPassword string `json:"new_password"`
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

func (h *AuthHandler) LoginOTP(w http.ResponseWriter, r *http.Request) {
	var req loginOTPBody
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Body harus JSON dengan challenge dan code")
		return
	}
	result, err := h.svc.LoginOTP(r.Context(), req.Challenge, req.Code)
	if err != nil {
		h.handleError(w, "login_otp", err)
		return
	}
	writeJSON(w, http.StatusOK, newAuthResponse(result))
}

// RequestOTP selalu membalas 202 agar keberadaan email tidak bocor.
func (h *AuthHandler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var req otpRequestBody
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Body harus JSON dengan email dan purpose")
		return
	}
	purpose, ok := parsePurpose(req.Purpose)
	if !ok {
		writeValidationError(w, map[string]string{"purpose": "Purpose tidak valid"})
		return
	}
	if err := h.svc.RequestOTP(r.Context(), req.Email, purpose, clientIP(r, h.trustedProxy)); err != nil {
		// Jangan bocorkan status email; cukup catat untuk operasional.
		slog.Warn("permintaan OTP gagal", "purpose", req.Purpose, "error", err)
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "sent"})
}

func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req otpVerifyBody
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Body harus JSON dengan email, purpose, dan code")
		return
	}
	purpose, ok := parsePurpose(req.Purpose)
	if !ok {
		writeValidationError(w, map[string]string{"purpose": "Purpose tidak valid"})
		return
	}
	ticket, err := h.svc.VerifyOTP(r.Context(), req.Email, purpose, req.Code)
	if err != nil {
		h.handleError(w, "verify_otp", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"verification_token": ticket})
}

// ForgotPassword selalu membalas 202.
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req otpRequestBody
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Body harus JSON dengan email")
		return
	}
	if err := h.svc.ForgotPassword(r.Context(), req.Email, clientIP(r, h.trustedProxy)); err != nil {
		slog.Warn("permintaan reset password gagal", "error", err)
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "sent"})
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordBody
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Body harus JSON dengan email, ticket, dan new_password")
		return
	}
	if err := h.svc.ResetPassword(r.Context(), req.Email, req.Ticket, req.NewPassword); err != nil {
		h.handleError(w, "reset_password", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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

func parsePurpose(raw string) (authotp.Purpose, bool) {
	switch authotp.Purpose(raw) {
	case authotp.PurposeRegister, authotp.PurposeResetPassword, authotp.PurposeAdminLogin:
		return authotp.Purpose(raw), true
	default:
		return "", false
	}
}

func (h *AuthHandler) handleError(w http.ResponseWriter, operation string, err error) {
	var validationErr *user.ValidationError
	switch {
	case errors.As(err, &validationErr):
		writeValidationError(w, validationErr.Fields)

	case errors.Is(err, user.ErrEmailTaken):
		writeError(w, http.StatusConflict, "email_taken", "Email sudah terdaftar")

	case errors.Is(err, user.ErrIdentityVerified):
		writeError(w, http.StatusConflict, "identity_already_verified",
			"NIM ini sudah terverifikasi pada akun lain. Masuk dengan akun tersebut atau hubungi admin.")

	case errors.Is(err, user.ErrNIMTaken):
		writeError(w, http.StatusConflict, "nim_taken",
			"NIM sudah terdaftar tetapi belum terverifikasi. Jika ini NIM kamu, masuk dan selesaikan verifikasi KTM atau hubungi admin.")

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

	case errors.Is(err, authotp.ErrTooManyRequests):
		writeError(w, http.StatusTooManyRequests, "too_many_requests",
			"Terlalu banyak permintaan kode. Coba lagi dalam 10 menit.")

	case errors.Is(err, authotp.ErrInvalidCode):
		writeError(w, http.StatusUnauthorized, "invalid_otp", "Kode verifikasi salah atau sudah kedaluwarsa.")

	case errors.Is(err, authotp.ErrExpired):
		writeError(w, http.StatusUnauthorized, "invalid_otp", "Kode verifikasi salah atau sudah kedaluwarsa.")

	case errors.Is(err, authotp.ErrInvalidTicket):
		writeError(w, http.StatusUnauthorized, "invalid_ticket", "Kode verifikasi tidak valid atau sudah kedaluwarsa.")

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
