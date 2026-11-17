package httpapi

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"unesarent/backend/internal/ktm"
)

// maxMultipartOverhead memberi ruang untuk boundary dan header multipart.
const maxMultipartOverhead = 1 << 20

type KTMHandler struct {
	svc *ktm.Service
}

func NewKTMHandler(svc *ktm.Service) *KTMHandler {
	return &KTMHandler{svc: svc}
}

// Submit menerima multipart/form-data dengan field "file".
func (h *KTMHandler) Submit(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, ktm.MaxUploadBytes+maxMultipartOverhead)

	if err := r.ParseMultipartForm(ktm.MaxUploadBytes); err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			writeError(w, http.StatusRequestEntityTooLarge, "file_too_large", "Ukuran file melebihi 5 MB")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_body", "Gunakan multipart/form-data dengan field 'file'")
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing_file", "Field 'file' wajib diisi")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, ktm.MaxUploadBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_file", "Gagal membaca file KTM")
		return
	}
	if int64(len(data)) > ktm.MaxUploadBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "file_too_large", "Ukuran file melebihi 5 MB")
		return
	}

	sub, err := h.svc.Submit(r.Context(), userIDFromContext(r.Context()), data)
	if err != nil {
		h.handleError(w, "submit_ktm", err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"submission": sub})
}

// Mine mengembalikan riwayat pengajuan milik user yang sedang login.
func (h *KTMHandler) Mine(w http.ResponseWriter, r *http.Request) {
	list, verified, err := h.svc.Mine(r.Context(), userIDFromContext(r.Context()), queryInt(r, "limit", 0))
	if err != nil {
		h.handleError(w, "list_my_ktm", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"verified":    verified,
		"submissions": list,
	})
}

// AdminList menampilkan antrean verifikasi untuk admin_dev.
func (h *KTMHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("status")))
	if status == "" {
		status = string(ktm.StatusPending)
	}

	list, limit, offset, err := h.svc.AdminList(
		r.Context(),
		status,
		queryInt(r, "limit", 0),
		queryInt(r, "offset", 0),
	)
	if err != nil {
		h.handleError(w, "admin_list_ktm", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":      status,
		"limit":       limit,
		"offset":      offset,
		"submissions": list,
	})
}

// Verify memerlukan body {"confirm_nim": true}.
func (h *KTMHandler) Verify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConfirmNIM bool `json:"confirm_nim"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Body harus JSON dengan field confirm_nim")
		return
	}

	sub, err := h.svc.Verify(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"), req.ConfirmNIM)
	if err != nil {
		h.handleError(w, "verify_ktm", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"submission": sub})
}

// Reject memerlukan body {"reason": "..."}.
func (h *KTMHandler) Reject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Body harus JSON dengan field reason")
		return
	}

	sub, err := h.svc.Reject(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"), req.Reason)
	if err != nil {
		h.handleError(w, "reject_ktm", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"submission": sub})
}

// Revoke memerlukan body {"reason": "..."}.
func (h *KTMHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Body harus JSON dengan field reason")
		return
	}

	sub, err := h.svc.Revoke(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"), req.Reason)
	if err != nil {
		h.handleError(w, "revoke_ktm", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"submission": sub})
}

// File mengalirkan file KTM untuk pemilik akun atau admin_dev.
func (h *KTMHandler) File(w http.ResponseWriter, r *http.Request) {
	sub, file, err := h.svc.File(
		r.Context(),
		userIDFromContext(r.Context()),
		roleFromContext(r.Context()),
		r.PathValue("id"),
	)
	if err != nil {
		h.handleError(w, "ktm_file", err)
		return
	}
	defer file.Close()

	filename := "ktm-" + sub.ID + fileExtension(sub.MimeType)

	w.Header().Set("Content-Type", sub.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(sub.FileSizeBytes, 10))
	w.Header().Set("Content-Disposition", `inline; filename="`+filename+`"`)
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, file); err != nil {
		slog.Warn("gagal mengirim file KTM", "id", sub.ID, "error", err)
	}
}

func (h *KTMHandler) handleError(w http.ResponseWriter, operation string, err error) {
	switch {
	case errors.Is(err, ktm.ErrNotFound):
		writeError(w, http.StatusNotFound, "ktm_not_found", "Verifikasi KTM tidak ditemukan")

	case errors.Is(err, ktm.ErrPendingExists):
		writeError(w, http.StatusConflict, "ktm_pending_exists", "Masih ada pengajuan KTM yang menunggu verifikasi")

	case errors.Is(err, ktm.ErrAlreadyVerified):
		writeError(w, http.StatusConflict, "ktm_already_verified", "Akun sudah terverifikasi KTM")

	case errors.Is(err, ktm.ErrInvalidTransition):
		writeError(w, http.StatusConflict, "ktm_invalid_transition", "Status KTM tidak memungkinkan aksi ini")

	case errors.Is(err, ktm.ErrIneligible):
		writeError(w, http.StatusForbidden, "ktm_not_eligible", "Hanya mahasiswa aktif ber-NIM yang dapat mengajukan KTM")

	case errors.Is(err, ktm.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "Anda tidak memiliki akses ke file KTM ini")

	case errors.Is(err, ktm.ErrFilePurged):
		writeError(w, http.StatusGone, "ktm_file_purged", "File KTM sudah dihapus")

	case errors.Is(err, ktm.ErrFileTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "file_too_large", "Ukuran file melebihi 5 MB")

	case errors.Is(err, ktm.ErrInvalidFile):
		writeValidationError(w, map[string]string{"file": "File KTM wajib diisi"})

	case errors.Is(err, ktm.ErrUnsupportedType):
		writeValidationError(w, map[string]string{"file": "Format file harus JPEG atau PNG"})

	case errors.Is(err, ktm.ErrNIMNotConfirmed):
		writeValidationError(w, map[string]string{"confirm_nim": "Admin wajib mengonfirmasi NIM pada KTM"})

	case errors.Is(err, ktm.ErrReasonRequired):
		writeValidationError(w, map[string]string{"reason": "Alasan wajib diisi"})

	case errors.Is(err, ktm.ErrInvalidStatus):
		writeValidationError(w, map[string]string{"status": "Status harus pending, verified, rejected, atau revoked"})

	default:
		slog.Error("kesalahan KTM", "operation", operation, "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan pada server")
	}
}

func queryInt(r *http.Request, key string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func fileExtension(mimeType string) string {
	if mimeType == "image/png" {
		return ".png"
	}
	return ".jpg"
}
