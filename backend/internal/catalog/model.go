package catalog

import (
	"errors"
	"time"
)

const DateFormat = "2006-01-02"

type Category struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type Location struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Owner struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Photo struct {
	ID            string `json:"id"`
	Position      int    `json:"position"`
	URL           string `json:"url"`
	StorageKey    string `json:"-"`
	MimeType      string `json:"mime_type"`
	FileSizeBytes int64  `json:"file_size_bytes"`
}

type Block struct {
	ID        string    `json:"id"`
	StartDate string    `json:"start_date"`
	EndDate   string    `json:"end_date"`
	Reason    *string   `json:"reason,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Period struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type Listing struct {
	ID       string   `json:"id"`
	Owner    Owner    `json:"owner"`
	Category Category `json:"category"`
	Location Location `json:"campus_location"`

	Title            string `json:"title"`
	Description      string `json:"description"`
	PricePerDay      int    `json:"price_per_day"`
	ConditionPercent int    `json:"condition_percent"`

	IsActive       bool       `json:"is_active"`
	TakedownReason *string    `json:"takedown_reason,omitempty"`
	TakedownAt     *time.Time `json:"takedown_at,omitempty"`

	CoverPhoto     *Photo   `json:"cover_photo,omitempty"`
	Photos         []Photo  `json:"photos,omitempty"`
	BlockedPeriods []Period `json:"blocked_periods,omitempty"`

	OwnerVerified bool `json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ListingInput struct {
	Title            string `json:"title"`
	Description      string `json:"description"`
	CategoryID       string `json:"category_id"`
	CampusLocationID string `json:"campus_location_id"`
	PricePerDay      int    `json:"price_per_day"`
	ConditionPercent int    `json:"condition_percent"`
}

type BlockInput struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Reason    string `json:"reason"`
}

type SearchParams struct {
	Query    string
	Category string
	Location string
	MinPrice string
	MaxPrice string
	Sort     string
	Limit    int
	Offset   int
}

type Filter struct {
	Query        string
	CategorySlug string
	LocationID   string
	MinPrice     int
	MaxPrice     int
	Sort         string
	Limit        int
	Offset       int
}

type ListingPage struct {
	Items  []Listing `json:"items"`
	Total  int       `json:"total"`
	Limit  int       `json:"limit"`
	Offset int       `json:"offset"`
}

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return "data yang dikirim tidak valid"
}

var (
	ErrNotFound          = errors.New("listing tidak ditemukan")
	ErrForbidden         = errors.New("bukan pemilik listing")
	ErrNotVerified       = errors.New("penyedia harus mahasiswa aktif dengan KTM terverifikasi")
	ErrPhotoLimit        = errors.New("maksimal 3 foto per listing")
	ErrListingInUse      = errors.New("listing sudah dipakai transaksi")
	ErrBlockOverlap      = errors.New("periode blokir bertabrakan")
	ErrInvalidTransition = errors.New("status listing tidak memungkinkan aksi ini")
	ErrInvalidFile       = errors.New("file foto tidak valid")
	ErrFileTooLarge      = errors.New("ukuran file melebihi 5 MB")
	ErrUnsupportedType   = errors.New("format file harus JPEG atau PNG")
	ErrInvalidReference  = errors.New("kategori atau lokasi tidak ditemukan")
	ErrInvalidStatus     = errors.New("status listing tidak valid")
)
