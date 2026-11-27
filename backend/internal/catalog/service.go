package catalog

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxPhotos                  = 3
	MaxPhotoBytes        int64 = 5 * 1024 * 1024
	DefaultListLimit           = 20
	MaxListLimit               = 60
	minTitleLength             = 3
	maxTitleLength             = 120
	maxDescriptionLength       = 2000
	maxPricePerDay             = 10_000_000
)

var jakarta = time.FixedZone("WIB", 7*60*60)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type Service struct {
	store Store
	files Storage
}

func NewService(store Store, files Storage) *Service {
	return &Service{store: store, files: files}
}

func (s *Service) Categories(ctx context.Context) ([]Category, error) {
	return s.store.ListCategories(ctx)
}

func (s *Service) Locations(ctx context.Context) ([]Location, error) {
	return s.store.ListLocations(ctx)
}

func (s *Service) Search(ctx context.Context, params SearchParams) (ListingPage, error) {
	filter, err := parseSearch(params)
	if err != nil {
		return ListingPage{}, err
	}

	items, total, err := s.store.Search(ctx, filter)
	if err != nil {
		return ListingPage{}, err
	}
	for i := range items {
		s.attachPhotoURLs(&items[i])
	}

	return ListingPage{Items: items, Total: total, Limit: filter.Limit, Offset: filter.Offset}, nil
}

func (s *Service) Detail(ctx context.Context, id string) (Listing, error) {
	listing, err := s.store.Get(ctx, id)
	if err != nil {
		return Listing{}, err
	}
	if !listing.IsActive || !listing.OwnerVerified {
		return Listing{}, ErrNotFound
	}

	s.attachPhotoURLs(&listing)
	listing.BlockedPeriods = upcomingPeriods(s.mustBlocks(ctx, listing.ID))
	return listing, nil
}

func (s *Service) Mine(ctx context.Context, ownerID string, limit, offset int) (ListingPage, error) {
	limit = clampLimit(limit)
	if offset < 0 {
		offset = 0
	}

	items, total, err := s.store.ListByOwner(ctx, ownerID, limit, offset)
	if err != nil {
		return ListingPage{}, err
	}
	for i := range items {
		s.attachPhotoURLs(&items[i])
	}
	return ListingPage{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *Service) Create(ctx context.Context, ownerID string, in ListingInput) (Listing, error) {
	in, err := validateListingInput(in)
	if err != nil {
		return Listing{}, err
	}

	verified, err := s.store.IsVerifiedOwner(ctx, ownerID)
	if err != nil {
		return Listing{}, err
	}
	if !verified {
		return Listing{}, ErrNotVerified
	}

	listing, err := s.store.Create(ctx, ownerID, in)
	if err != nil {
		return Listing{}, err
	}
	s.attachPhotoURLs(&listing)
	return listing, nil
}

func (s *Service) Update(ctx context.Context, ownerID, id string, in ListingInput) (Listing, error) {
	if _, err := s.ownedListing(ctx, ownerID, id); err != nil {
		return Listing{}, err
	}

	in, err := validateListingInput(in)
	if err != nil {
		return Listing{}, err
	}

	listing, err := s.store.Update(ctx, id, in)
	if err != nil {
		return Listing{}, err
	}
	s.attachPhotoURLs(&listing)
	return listing, nil
}

func (s *Service) Delete(ctx context.Context, ownerID, id string) error {
	if _, err := s.ownedListing(ctx, ownerID, id); err != nil {
		return err
	}

	photos, err := s.store.ListPhotos(ctx, id)
	if err != nil {
		return err
	}
	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}

	for _, p := range photos {
		s.deleteFile(ctx, p.StorageKey)
	}
	return nil
}

func (s *Service) AddPhoto(ctx context.Context, ownerID, listingID string, data []byte) (Photo, error) {
	if _, err := s.ownedListing(ctx, ownerID, listingID); err != nil {
		return Photo{}, err
	}

	if len(data) == 0 {
		return Photo{}, ErrInvalidFile
	}
	if int64(len(data)) > MaxPhotoBytes {
		return Photo{}, ErrFileTooLarge
	}

	mimeType := http.DetectContentType(data)
	ext, ok := allowedImageExtension(mimeType)
	if !ok {
		return Photo{}, ErrUnsupportedType
	}

	count, err := s.store.CountPhotos(ctx, listingID)
	if err != nil {
		return Photo{}, err
	}
	if count >= MaxPhotos {
		return Photo{}, ErrPhotoLimit
	}

	key, err := newPhotoKey(listingID, ext)
	if err != nil {
		return Photo{}, err
	}
	if err := s.files.Save(ctx, key, data, mimeType); err != nil {
		return Photo{}, fmt.Errorf("menyimpan foto listing: %w", err)
	}

	photo, err := s.store.AddPhoto(ctx, listingID, key, mimeType, int64(len(data)))
	if err != nil {
		s.deleteFile(ctx, key)
		return Photo{}, err
	}
	photo.URL = photoURL(listingID, photo.ID)
	return photo, nil
}

func (s *Service) DeletePhoto(ctx context.Context, ownerID, listingID, photoID string) error {
	if _, err := s.ownedListing(ctx, ownerID, listingID); err != nil {
		return err
	}

	photo, err := s.store.DeletePhoto(ctx, listingID, photoID)
	if err != nil {
		return err
	}
	s.deleteFile(ctx, photo.StorageKey)
	return nil
}

func (s *Service) PhotoFile(ctx context.Context, listingID, photoID string) (Photo, io.ReadCloser, error) {
	photo, err := s.store.GetPhoto(ctx, listingID, photoID)
	if err != nil {
		return Photo{}, nil, err
	}

	file, err := s.files.Open(ctx, photo.StorageKey)
	if err != nil {
		return Photo{}, nil, err
	}
	return photo, file, nil
}

func (s *Service) Blocks(ctx context.Context, ownerID, listingID string) ([]Block, error) {
	if _, err := s.ownedListing(ctx, ownerID, listingID); err != nil {
		return nil, err
	}
	return s.store.ListBlocks(ctx, listingID)
}

func (s *Service) AddBlock(ctx context.Context, ownerID, listingID string, in BlockInput) (Block, error) {
	if _, err := s.ownedListing(ctx, ownerID, listingID); err != nil {
		return Block{}, err
	}

	start, end, err := parseBlockRange(in, todayWIB())
	if err != nil {
		return Block{}, err
	}

	var reason *string
	if trimmed := strings.TrimSpace(in.Reason); trimmed != "" {
		reason = &trimmed
	}
	return s.store.AddBlock(ctx, listingID, start, end, reason)
}

func (s *Service) DeleteBlock(ctx context.Context, ownerID, listingID, blockID string) error {
	if _, err := s.ownedListing(ctx, ownerID, listingID); err != nil {
		return err
	}
	return s.store.DeleteBlock(ctx, listingID, blockID)
}

func (s *Service) AdminList(ctx context.Context, status, query string, limit, offset int) (ListingPage, error) {
	status = strings.TrimSpace(strings.ToLower(status))
	switch status {
	case "", "active", "taken_down":
	default:
		return ListingPage{}, ErrInvalidStatus
	}

	limit = clampLimit(limit)
	if offset < 0 {
		offset = 0
	}

	items, total, err := s.store.AdminList(ctx, status, strings.TrimSpace(query), limit, offset)
	if err != nil {
		return ListingPage{}, err
	}
	for i := range items {
		s.attachPhotoURLs(&items[i])
	}
	return ListingPage{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *Service) Takedown(ctx context.Context, adminID, id, reason string) (Listing, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return Listing{}, &ValidationError{Fields: map[string]string{"reason": "Alasan wajib diisi"}}
	}

	listing, err := s.store.Get(ctx, id)
	if err != nil {
		return Listing{}, err
	}
	if !listing.IsActive {
		return Listing{}, ErrInvalidTransition
	}

	if err := s.store.Takedown(ctx, id, adminID, reason); err != nil {
		return Listing{}, err
	}
	s.logAction(ctx, adminID, "takedown_listing", id, reason)

	updated, err := s.store.Get(ctx, id)
	if err != nil {
		return Listing{}, err
	}
	s.attachPhotoURLs(&updated)
	return updated, nil
}

func (s *Service) Restore(ctx context.Context, adminID, id string) (Listing, error) {
	listing, err := s.store.Get(ctx, id)
	if err != nil {
		return Listing{}, err
	}
	if listing.IsActive {
		return Listing{}, ErrInvalidTransition
	}

	if err := s.store.Restore(ctx, id); err != nil {
		return Listing{}, err
	}
	s.logAction(ctx, adminID, "restore_listing", id, "")

	updated, err := s.store.Get(ctx, id)
	if err != nil {
		return Listing{}, err
	}
	s.attachPhotoURLs(&updated)
	return updated, nil
}

func (s *Service) ownedListing(ctx context.Context, ownerID, id string) (Listing, error) {
	listing, err := s.store.Get(ctx, id)
	if err != nil {
		return Listing{}, err
	}
	if listing.Owner.ID != ownerID {
		return Listing{}, ErrForbidden
	}
	return listing, nil
}

func (s *Service) attachPhotoURLs(listing *Listing) {
	for i := range listing.Photos {
		listing.Photos[i].URL = photoURL(listing.ID, listing.Photos[i].ID)
	}
	if listing.CoverPhoto != nil {
		listing.CoverPhoto.URL = photoURL(listing.ID, listing.CoverPhoto.ID)
	}
}

func (s *Service) mustBlocks(ctx context.Context, listingID string) []Block {
	blocks, err := s.store.ListBlocks(ctx, listingID)
	if err != nil {
		slog.Warn("gagal membaca blokir listing", "listing", listingID, "error", err)
		return nil
	}
	return blocks
}

func (s *Service) deleteFile(ctx context.Context, key string) {
	if err := s.files.Delete(ctx, key); err != nil {
		slog.Warn("gagal menghapus file foto listing", "key", key, "error", err)
	}
}

func (s *Service) logAction(ctx context.Context, actorID, action, targetID, reason string) {
	if err := s.store.LogAction(ctx, actorID, action, targetID, reason); err != nil {
		slog.Warn("gagal mencatat audit listing", "action", action, "target", targetID, "error", err)
	}
}

func parseSearch(params SearchParams) (Filter, error) {
	fields := map[string]string{}

	f := Filter{
		Query:        strings.TrimSpace(params.Query),
		CategorySlug: strings.TrimSpace(params.Category),
		LocationID:   strings.ToLower(strings.TrimSpace(params.Location)),
		Sort:         strings.TrimSpace(params.Sort),
		Limit:        clampLimit(params.Limit),
		Offset:       params.Offset,
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	if utf8.RuneCountInString(f.Query) > maxTitleLength {
		f.Query = string([]rune(f.Query)[:maxTitleLength])
	}
	if f.LocationID != "" && !uuidPattern.MatchString(f.LocationID) {
		fields["location"] = "ID lokasi tidak valid"
	}

	f.MinPrice = parseOptionalInt(params.MinPrice, "min_price", fields)
	f.MaxPrice = parseOptionalInt(params.MaxPrice, "max_price", fields)
	if f.MinPrice < 0 {
		fields["min_price"] = "min_price tidak boleh negatif"
	}
	if f.MaxPrice < 0 {
		fields["max_price"] = "max_price tidak boleh negatif"
	}
	if f.MinPrice > 0 && f.MaxPrice > 0 && f.MinPrice > f.MaxPrice {
		fields["max_price"] = "max_price harus lebih besar dari min_price"
	}

	switch f.Sort {
	case "", "newest", "price_asc", "price_desc":
	default:
		fields["sort"] = "sort harus newest, price_asc, atau price_desc"
	}

	if len(fields) > 0 {
		return Filter{}, &ValidationError{Fields: fields}
	}
	return f, nil
}

func validateListingInput(in ListingInput) (ListingInput, error) {
	fields := map[string]string{}

	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	in.CategoryID = strings.ToLower(strings.TrimSpace(in.CategoryID))
	in.CampusLocationID = strings.ToLower(strings.TrimSpace(in.CampusLocationID))

	if n := utf8.RuneCountInString(in.Title); n < minTitleLength || n > maxTitleLength {
		fields["title"] = "Judul harus 3-120 karakter"
	}
	if utf8.RuneCountInString(in.Description) > maxDescriptionLength {
		fields["description"] = "Deskripsi maksimal 2000 karakter"
	}
	if !uuidPattern.MatchString(in.CategoryID) {
		fields["category_id"] = "Kategori tidak valid"
	}
	if !uuidPattern.MatchString(in.CampusLocationID) {
		fields["campus_location_id"] = "Lokasi tidak valid"
	}
	if in.PricePerDay <= 0 || in.PricePerDay > maxPricePerDay {
		fields["price_per_day"] = "Harga per hari harus 1-10.000.000"
	}
	if in.ConditionPercent < 1 || in.ConditionPercent > 100 {
		fields["condition_percent"] = "Kondisi harus 1-100"
	}

	if len(fields) > 0 {
		return ListingInput{}, &ValidationError{Fields: fields}
	}
	return in, nil
}

func parseBlockRange(in BlockInput, today time.Time) (time.Time, time.Time, error) {
	fields := map[string]string{}

	start, err := time.ParseInLocation(DateFormat, strings.TrimSpace(in.StartDate), jakarta)
	if err != nil {
		fields["start_date"] = "Format tanggal harus YYYY-MM-DD"
	}
	end, err := time.ParseInLocation(DateFormat, strings.TrimSpace(in.EndDate), jakarta)
	if err != nil {
		fields["end_date"] = "Format tanggal harus YYYY-MM-DD"
	}

	if len(fields) == 0 {
		if end.Before(start) {
			fields["end_date"] = "Tanggal selesai tidak boleh sebelum tanggal mulai"
		} else if start.Before(today) {
			fields["start_date"] = "Tanggal mulai tidak boleh di masa lalu"
		}
	}

	if len(fields) > 0 {
		return time.Time{}, time.Time{}, &ValidationError{Fields: fields}
	}
	return start, end, nil
}

func upcomingPeriods(blocks []Block) []Period {
	today := todayWIB().Format(DateFormat)
	if len(blocks) == 0 {
		return nil
	}

	periods := make([]Period, 0, len(blocks))
	for _, b := range blocks {
		if b.EndDate < today {
			continue
		}
		periods = append(periods, Period{StartDate: b.StartDate, EndDate: b.EndDate})
	}
	if len(periods) == 0 {
		return nil
	}
	return periods
}

func parseOptionalInt(raw, field string, fields map[string]string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		fields[field] = field + " harus berupa angka"
		return 0
	}
	return v
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

func newPhotoKey(listingID, ext string) (string, error) {
	listingID = strings.ToLower(strings.TrimSpace(listingID))
	if !uuidPattern.MatchString(listingID) {
		return "", ErrNotFound
	}

	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("membuat nama file foto: %w", err)
	}
	return fmt.Sprintf("listings/%s/%s.%s", listingID, hex.EncodeToString(b), ext), nil
}

func photoURL(listingID, photoID string) string {
	return fmt.Sprintf("/api/listings/%s/photos/%s/file", listingID, photoID)
}

func todayWIB() time.Time {
	now := time.Now().In(jakarta)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, jakarta)
}
