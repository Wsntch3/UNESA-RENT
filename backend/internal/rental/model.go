package rental

import (
	"errors"
	"time"
)

// DateFormat adalah format tanggal API untuk rental dan availability.
const DateFormat = "2006-01-02"

type Status string

const (
	StatusPending   Status = "pending"
	StatusApproved  Status = "approved"
	StatusRejected  Status = "rejected"
	StatusCancelled Status = "cancelled"
	StatusCompleted Status = "completed"
)

type Party struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ListingSummary struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	CoverPhotoURL string `json:"cover_photo_url,omitempty"`
}

type Rental struct {
	ID      string         `json:"id"`
	Listing ListingSummary `json:"listing"`
	Renter  Party          `json:"renter"`
	Owner   Party          `json:"owner"`

	StartDate           string `json:"start_date"`
	EndDate             string `json:"end_date"`
	TotalDays           int    `json:"total_days"`
	TotalPrice          int    `json:"total_price"`
	PricePerDaySnapshot int    `json:"price_per_day_snapshot"`

	Status          Status  `json:"status"`
	RejectionReason *string `json:"rejection_reason,omitempty"`

	ApprovedAt  *time.Time `json:"approved_at,omitempty"`
	RejectedAt  *time.Time `json:"rejected_at,omitempty"`
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// MyRole diisi service: "renter" atau "owner".
	MyRole string `json:"my_role,omitempty"`
	// AgreementURL hanya diisi untuk rental approved.
	AgreementURL string `json:"agreement_url,omitempty"`

	Events []Event `json:"events,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Event struct {
	ID         int64     `json:"id"`
	FromStatus *string   `json:"from_status,omitempty"`
	ToStatus   string    `json:"to_status"`
	Actor      *Party    `json:"actor,omitempty"`
	Note       *string   `json:"note,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Message struct {
	ID        string    `json:"id"`
	RentalID  string    `json:"rental_id"`
	Sender    Party     `json:"sender"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`

	// Metadata hanya untuk tampilan admin; body sengaja tidak dikirim.
	BodyLength int    `json:"body_length,omitempty"`
	BodySHA256 string `json:"body_sha256,omitempty"`

	// Field internal untuk dekripsi di service. Tidak pernah di-serialize.
	Ciphertext []byte `json:"-"`
	Nonce      []byte `json:"-"`
	KeyVersion int    `json:"-"`
}

// AdminMessage tidak memuat isi pesan, hanya metadata.
type AdminMessage struct {
	ID         string    `json:"id"`
	Sender     Party     `json:"sender"`
	BodyLength int       `json:"body_length"`
	BodySHA256 string    `json:"body_sha256"`
	CreatedAt  time.Time `json:"created_at"`
}

type Document struct {
	ID          string    `json:"id"`
	RentalID    string    `json:"rental_id"`
	Type        string    `json:"type"`
	MimeType    string    `json:"mime_type"`
	SizeBytes   int64     `json:"size_bytes"`
	UploadedBy  *Party    `json:"uploaded_by,omitempty"`
	Visibility  string    `json:"visibility"`
	CreatedAt   time.Time `json:"created_at"`
	DownloadURL string    `json:"download_url,omitempty"`
}

type CODState struct {
	AgreedAt            *time.Time `json:"agreed_at,omitempty"`
	HandoverAt          *time.Time `json:"handover_at,omitempty"`
	HandoverConfirmedAt *time.Time `json:"handover_confirmed_at,omitempty"`
	ReturnAt            *time.Time `json:"return_at,omitempty"`
	ReturnConfirmedAt   *time.Time `json:"return_confirmed_at,omitempty"`
	AmountDue           *int       `json:"amount_due,omitempty"`
	AmountPaid          *int       `json:"amount_paid,omitempty"`
	PaymentMethod       *string    `json:"payment_method,omitempty"`
	SettledAt           *time.Time `json:"settled_at,omitempty"`
	Notes               *string    `json:"notes,omitempty"`
}

type CODSummary struct {
	CODState
	RentalID  string `json:"rental_id"`
	Location  string `json:"location,omitempty"`
	OwnerNIM  string `json:"owner_nim,omitempty"`
	RenterNIM string `json:"renter_nim,omitempty"`
}

type CODProofInput struct {
	Note string `json:"note"`
}

type CODSettleInput struct {
	AmountPaid    int    `json:"amount_paid"`
	PaymentMethod string `json:"payment_method"`
	Notes         string `json:"notes"`
}

type DocumentFilter struct {
	RentalID string
	Type     string
}

type RentalStats struct {
	Total       int            `json:"total"`
	ByStatus    map[string]int `json:"by_status"`
	Overdue     int            `json:"overdue"`
	Documents   int            `json:"documents"`
	StorageByte int64          `json:"storage_bytes"`
}

type HistoryMonth struct {
	Month     string `json:"month"`
	Total     int    `json:"total"`
	Value     int    `json:"total_value"`
	Completed int    `json:"completed"`
	Cancelled int    `json:"cancelled"`
}

type HistorySummary struct {
	Year   int            `json:"year"`
	Months []HistoryMonth `json:"months"`
	Total  int            `json:"total"`
	Value  int            `json:"total_value"`
}

type RentalFilter struct {
	Role    string
	Status  string
	From    string
	To      string
	Query   string
	Overdue bool
	Limit   int
	Offset  int
}

type Notification struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Title     string         `json:"title"`
	Body      *string        `json:"body,omitempty"`
	Data      map[string]any `json:"data"`
	ReadAt    *time.Time     `json:"read_at,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type Unavailable struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Source    string `json:"source"` // "block" atau "rental"
}

type DateInput struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type MessageInput struct {
	Body string `json:"body"`
}

type RentalPage struct {
	Items  []Rental `json:"items"`
	Total  int      `json:"total"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
}

type NotificationPage struct {
	Items       []Notification `json:"items"`
	Total       int            `json:"total"`
	UnreadCount int            `json:"unread_count"`
	Limit       int            `json:"limit"`
	Offset      int            `json:"offset"`
}

// ValidationError membawa pesan per field agar handler mengirim 422 konsisten.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return "data yang dikirim tidak valid"
}

var (
	ErrNotFound          = errors.New("rental tidak ditemukan")
	ErrForbidden         = errors.New("bukan peserta rental")
	ErrNotVerified       = errors.New("akun harus terverifikasi KTM")
	ErrInvalidStatus     = errors.New("status rental tidak valid")
	ErrInvalidDates      = errors.New("tanggal rental tidak valid")
	ErrReasonRequired    = errors.New("alasan wajib diisi")
	ErrInvalidTransition = errors.New("transisi status rental tidak valid")
	ErrDateUnavailable   = errors.New("tanggal sudah dipesan rental lain")
	ErrDatesBlocked      = errors.New("tanggal diblokir penyedia")
	ErrChatClosed        = errors.New("chat sudah ditutup untuk rental ini")
	ErrInvalidBody       = errors.New("isi pesan tidak valid")
	ErrAgreementMissing  = errors.New("file perjanjian tidak tersedia")
	ErrCODIncomplete     = errors.New("serah terima dan pengembalian belum lengkap")
	ErrInvalidCODState   = errors.New("transisi COD tidak valid")
	ErrInvalidReason     = errors.New("alasan wajib minimal 10 karakter")
	ErrInvalidPassword   = errors.New("password admin tidak benar")
	ErrDocumentNotFound  = errors.New("dokumen tidak ditemukan")
	ErrMissingProof      = errors.New("bukti foto wajib diunggah")
)
