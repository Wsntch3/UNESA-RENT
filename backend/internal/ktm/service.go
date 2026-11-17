package ktm

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
)

const (
	// MaxUploadBytes sesuai constraint ktm_verifications_file_size_check.
	MaxUploadBytes int64 = 5 * 1024 * 1024

	DefaultListLimit = 50
	MaxListLimit     = 100

	roleMahasiswa = "mahasiswa"
	roleAdminDev  = "admin_dev"
	statusActive  = "active"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Service berisi aturan bisnis verifikasi KTM.
type Service struct {
	store Store
	files Storage
}

func NewService(store Store, files Storage) *Service {
	return &Service{store: store, files: files}
}

// Submit memvalidasi file, menyimpannya, lalu membuat baris pending.
// Jika insert database gagal, file yang sudah tersimpan dihapus kembali.
func (s *Service) Submit(ctx context.Context, userID string, data []byte) (Submission, error) {
	if len(data) == 0 {
		return Submission{}, ErrInvalidFile
	}
	if int64(len(data)) > MaxUploadBytes {
		return Submission{}, ErrFileTooLarge
	}

	mimeType := http.DetectContentType(data)
	ext, ok := allowedImageExtension(mimeType)
	if !ok {
		return Submission{}, ErrUnsupportedType
	}

	state, err := s.store.UserState(ctx, userID)
	if err != nil {
		return Submission{}, err
	}
	if state.Role != roleMahasiswa || state.NIM == "" || state.Status != statusActive {
		return Submission{}, ErrIneligible
	}
	if state.Verified {
		return Submission{}, ErrAlreadyVerified
	}

	key, err := newStorageKey(userID, ext)
	if err != nil {
		return Submission{}, err
	}

	sum := sha256.Sum256(data)
	checksum := hex.EncodeToString(sum[:])

	if err := s.files.Save(ctx, key, data, mimeType); err != nil {
		return Submission{}, fmt.Errorf("menyimpan file KTM: %w", err)
	}

	id, err := s.store.Create(ctx, NewSubmission{
		UserID:         userID,
		StorageKey:     key,
		MimeType:       mimeType,
		FileSizeBytes:  int64(len(data)),
		ChecksumSHA256: checksum,
	})
	if err != nil {
		if delErr := s.files.Delete(ctx, key); delErr != nil {
			slog.Error("gagal menghapus file KTM setelah insert gagal", "key", key, "error", delErr)
		}
		return Submission{}, err
	}

	return s.store.GetByID(ctx, id)
}

// Mine mengembalikan riwayat pengajuan milik user beserta status verifikasi efektif.
func (s *Service) Mine(ctx context.Context, userID string, limit int) ([]Submission, bool, error) {
	limit = clampLimit(limit)

	state, err := s.store.UserState(ctx, userID)
	if err != nil {
		return nil, false, err
	}

	list, err := s.store.ListByUser(ctx, userID, limit)
	if err != nil {
		return nil, false, err
	}
	return list, state.Verified, nil
}

// AdminList mengembalikan antrean verifikasi. Default status adalah pending.
func (s *Service) AdminList(ctx context.Context, status string, limit, offset int) ([]Submission, int, int, error) {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		status = string(StatusPending)
	}
	switch Status(status) {
	case StatusPending, StatusVerified, StatusRejected, StatusRevoked:
	default:
		return nil, 0, 0, ErrInvalidStatus
	}

	limit = clampLimit(limit)
	if offset < 0 {
		offset = 0
	}

	list, err := s.store.ListByStatus(ctx, status, limit, offset)
	if err != nil {
		return nil, 0, 0, err
	}
	return list, limit, offset, nil
}

// Verify menandai pengajuan pending sebagai verified.
// confirmNIM harus true: admin menyatakan NIM di kartu sama dengan NIM akun.
func (s *Service) Verify(ctx context.Context, adminID, id string, confirmNIM bool) (Submission, error) {
	if !confirmNIM {
		return Submission{}, ErrNIMNotConfirmed
	}

	sub, err := s.store.GetByID(ctx, id)
	if err != nil {
		return Submission{}, err
	}
	if sub.Status != StatusPending {
		return Submission{}, ErrInvalidTransition
	}

	if err := s.store.MarkVerified(ctx, sub.ID, adminID); err != nil {
		return Submission{}, err
	}
	s.logAction(ctx, adminID, "verify_ktm", sub.ID, "")

	return s.store.GetByID(ctx, sub.ID)
}

// Reject menolak pengajuan pending dengan alasan wajib.
func (s *Service) Reject(ctx context.Context, adminID, id, reason string) (Submission, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return Submission{}, ErrReasonRequired
	}

	sub, err := s.store.GetByID(ctx, id)
	if err != nil {
		return Submission{}, err
	}
	if sub.Status != StatusPending {
		return Submission{}, ErrInvalidTransition
	}

	if err := s.store.MarkRejected(ctx, sub.ID, adminID, reason); err != nil {
		return Submission{}, err
	}
	s.logAction(ctx, adminID, "reject_ktm", sub.ID, reason)

	return s.store.GetByID(ctx, sub.ID)
}

// Revoke mencabut verifikasi yang sudah ada dengan alasan wajib.
func (s *Service) Revoke(ctx context.Context, adminID, id, reason string) (Submission, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return Submission{}, ErrReasonRequired
	}

	sub, err := s.store.GetByID(ctx, id)
	if err != nil {
		return Submission{}, err
	}
	if sub.Status != StatusVerified {
		return Submission{}, ErrInvalidTransition
	}

	if err := s.store.MarkRevoked(ctx, sub.ID, adminID, reason); err != nil {
		return Submission{}, err
	}
	s.logAction(ctx, adminID, "revoke_ktm", sub.ID, reason)

	return s.store.GetByID(ctx, sub.ID)
}

// File membuka file KTM. Hanya pemilik akun dan admin_dev yang boleh mengakses.
func (s *Service) File(ctx context.Context, requesterID, requesterRole, id string) (Submission, io.ReadCloser, error) {
	sub, err := s.store.GetByID(ctx, id)
	if err != nil {
		return Submission{}, nil, err
	}
	if requesterRole != roleAdminDev && sub.UserID != requesterID {
		return Submission{}, nil, ErrForbidden
	}
	if sub.FilePurgedAt != nil {
		return Submission{}, nil, ErrFilePurged
	}

	file, err := s.files.Open(ctx, sub.StorageKey)
	if err != nil {
		return Submission{}, nil, err
	}

	s.logAction(ctx, requesterID, "view_ktm_file", sub.ID, "")
	return sub, file, nil
}

func (s *Service) logAction(ctx context.Context, actorID, action, targetID, reason string) {
	if err := s.store.LogAction(ctx, actorID, action, targetID, reason); err != nil {
		slog.Warn("gagal mencatat audit KTM", "action", action, "target", targetID, "error", err)
	}
}

func clampLimit(limit int) int {
	switch {
	case limit <= 0:
		return DefaultListLimit
	case limit > MaxListLimit:
		return MaxListLimit
	default:
		return limit
	}
}

func allowedImageExtension(mimeType string) (string, bool) {
	switch mimeType {
	case "image/jpeg":
		return "jpg", true
	case "image/png":
		return "png", true
	default:
		return "", false
	}
}

func newStorageKey(userID, ext string) (string, error) {
	userID = strings.ToLower(strings.TrimSpace(userID))
	if !uuidPattern.MatchString(userID) {
		return "", ErrNotFound
	}

	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("membuat nama file KTM: %w", err)
	}
	return fmt.Sprintf("ktm/%s/%s.%s", userID, hex.EncodeToString(b), ext), nil
}
