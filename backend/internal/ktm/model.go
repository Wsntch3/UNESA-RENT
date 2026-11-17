package ktm

import (
	"errors"
	"time"
)

// Status mengikuti constraint ktm_verifications_status_check.
type Status string

const (
	StatusPending  Status = "pending"
	StatusVerified Status = "verified"
	StatusRejected Status = "rejected"
	StatusRevoked  Status = "revoked"
)

// Submission adalah satu baris ktm_verifications beserta data pemiliknya.
// StorageKey tidak pernah dikirim ke client; file diambil lewat endpoint file.
type Submission struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`

	UserName  string `json:"user_name,omitempty"`
	UserEmail string `json:"user_email,omitempty"`
	UserNIM   string `json:"user_nim,omitempty"`

	StorageKey           string `json:"-"`
	MimeType             string `json:"mime_type"`
	FileSizeBytes        int64  `json:"file_size_bytes"`
	Status               Status `json:"status"`
	ReviewerConfirmedNIM bool   `json:"reviewer_confirmed_nim"`

	SubmittedAt time.Time `json:"submitted_at"`

	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	RejectionReason *string    `json:"rejection_reason,omitempty"`

	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	RevokeReason *string    `json:"revoke_reason,omitempty"`

	FilePurgedAt *time.Time `json:"file_purged_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// NewSubmission adalah data minimum untuk membuat pengajuan baru.
// nim_snapshot dan validasi pemilik diisi oleh trigger database.
type NewSubmission struct {
	UserID         string
	StorageKey     string
	MimeType       string
	FileSizeBytes  int64
	ChecksumSHA256 string
}

// UserState dipakai untuk menolak pengajuan lebih awal dengan pesan yang jelas.
type UserState struct {
	ID       string
	Name     string
	Email    string
	NIM      string
	Role     string
	Status   string
	Verified bool
}

var (
	ErrNotFound          = errors.New("verifikasi KTM tidak ditemukan")
	ErrPendingExists     = errors.New("masih ada pengajuan KTM yang menunggu verifikasi")
	ErrAlreadyVerified   = errors.New("akun sudah terverifikasi KTM")
	ErrIneligible        = errors.New("hanya mahasiswa aktif ber-NIM yang dapat mengajukan KTM")
	ErrInvalidTransition = errors.New("status KTM tidak memungkinkan aksi ini")
	ErrInvalidFile       = errors.New("file KTM tidak valid")
	ErrFileTooLarge      = errors.New("ukuran file melebihi 5 MB")
	ErrUnsupportedType   = errors.New("format file harus JPEG atau PNG")
	ErrNIMNotConfirmed   = errors.New("admin wajib mengonfirmasi NIM pada KTM")
	ErrReasonRequired    = errors.New("alasan wajib diisi")
	ErrForbidden         = errors.New("tidak memiliki akses ke file KTM ini")
	ErrFilePurged        = errors.New("file KTM sudah dihapus")
	ErrInvalidStatus     = errors.New("status KTM tidak valid")
)
