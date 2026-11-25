package rental

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	DefaultListLimit = 20
	MaxListLimit     = 60
	DefaultChatLimit = 50
	MaxChatLimit     = 100
	maxRentalDays    = 30
	maxAvailability  = 90
	maxMessageLength = 2000
)

// Jakarta tidak memakai DST, jadi fixed zone cukup dan tidak bergantung tzdata.
var jakarta = time.FixedZone("WIB", 7*60*60)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Availability(ctx context.Context, listingID, fromRaw, toRaw string) ([]Unavailable, error) {
	listingID = strings.ToLower(strings.TrimSpace(listingID))
	if !uuidPattern.MatchString(listingID) {
		return nil, ErrNotFound
	}

	today := todayWIB()
	from, to := today, today.AddDate(0, 0, maxAvailability-1)
	fields := map[string]string{}

	if raw := strings.TrimSpace(fromRaw); raw != "" {
		parsed, err := time.ParseInLocation(DateFormat, raw, jakarta)
		if err != nil {
			fields["from"] = "Format tanggal harus YYYY-MM-DD"
		} else {
			from = parsed
		}
	}
	if raw := strings.TrimSpace(toRaw); raw != "" {
		parsed, err := time.ParseInLocation(DateFormat, raw, jakarta)
		if err != nil {
			fields["to"] = "Format tanggal harus YYYY-MM-DD"
		} else {
			to = parsed
		}
	}
	if len(fields) == 0 {
		if to.Before(from) {
			fields["to"] = "Tanggal akhir tidak boleh sebelum tanggal mulai"
		} else if daysBetween(from, to)+1 > maxAvailability {
			fields["to"] = fmt.Sprintf("Rentang maksimal %d hari", maxAvailability)
		}
	}
	if len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}
	return s.store.Availability(ctx, listingID, from, to)
}

func (s *Service) Submit(ctx context.Context, renterID, listingID string, in DateInput) (Rental, error) {
	start, end, err := validateRentalDates(in)
	if err != nil {
		return Rental{}, err
	}

	verified, err := s.store.IsVerified(ctx, renterID)
	if err != nil {
		return Rental{}, err
	}
	if !verified {
		return Rental{}, ErrNotVerified
	}

	ownerID, active, err := s.store.ListingInfo(ctx, listingID)
	if err != nil {
		return Rental{}, err
	}
	if ownerID == strings.TrimSpace(renterID) {
		return Rental{}, ErrForbidden
	}
	if !active {
		return Rental{}, ErrInvalidTransition
	}

	rental, err := s.store.Create(ctx, listingID, renterID, start, end)
	if err != nil {
		return Rental{}, err
	}

	pending := StatusPending
	s.addEvent(ctx, rental.ID, nil, pending, renterID, nil)
	s.notify(ctx, rental.Owner.ID, "rental_requested",
		"Pengajuan sewa baru",
		fmt.Sprintf("%s mengajukan sewa %s.", rental.Renter.Name, rental.Listing.Title),
		rental.ID)

	return s.decorate(rental, renterID), nil
}

func (s *Service) Mine(ctx context.Context, userID, status string, limit, offset int) (RentalPage, error) {
	status = strings.TrimSpace(strings.ToLower(status))
	switch Status(status) {
	case "", StatusPending, StatusApproved, StatusRejected, StatusCancelled, StatusCompleted:
	default:
		return RentalPage{}, &ValidationError{Fields: map[string]string{"status": "Status rental tidak valid"}}
	}

	limit = clampLimit(limit, DefaultListLimit, MaxListLimit)
	if offset < 0 {
		offset = 0
	}

	items, total, err := s.store.ListMine(ctx, userID, status, limit, offset)
	if err != nil {
		return RentalPage{}, err
	}
	for i := range items {
		items[i] = s.decorate(items[i], userID)
	}
	return RentalPage{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *Service) Get(ctx context.Context, userID, id string) (Rental, error) {
	rental, err := s.loadForParticipant(ctx, userID, id)
	if err != nil {
		return Rental{}, err
	}
	events, err := s.store.Events(ctx, rental.ID)
	if err != nil {
		return Rental{}, err
	}
	rental.Events = events
	return s.decorate(rental, userID), nil
}

func (s *Service) Approve(ctx context.Context, ownerID, id string) (Rental, error) {
	rental, err := s.loadForOwner(ctx, ownerID, id)
	if err != nil {
		return Rental{}, err
	}
	if rental.Status != StatusPending {
		return Rental{}, ErrInvalidTransition
	}

	if err := s.store.UpdateStatus(ctx, rental.ID, StatusApproved, nil); err != nil {
		return Rental{}, err
	}
	from := rental.Status
	s.addEvent(ctx, rental.ID, &from, StatusApproved, ownerID, nil)
	s.notify(ctx, rental.Renter.ID, "rental_approved",
		"Pengajuan sewa disetujui",
		fmt.Sprintf("%s menyetujui sewa %s.", rental.Owner.Name, rental.Listing.Title),
		rental.ID)

	return s.reload(ctx, ownerID, rental.ID)
}

func (s *Service) Reject(ctx context.Context, ownerID, id, reason string) (Rental, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return Rental{}, ErrReasonRequired
	}

	rental, err := s.loadForOwner(ctx, ownerID, id)
	if err != nil {
		return Rental{}, err
	}
	if rental.Status != StatusPending {
		return Rental{}, ErrInvalidTransition
	}

	if err := s.store.UpdateStatus(ctx, rental.ID, StatusRejected, &reason); err != nil {
		return Rental{}, err
	}
	from := rental.Status
	s.addEvent(ctx, rental.ID, &from, StatusRejected, ownerID, &reason)
	s.notify(ctx, rental.Renter.ID, "rental_rejected",
		"Pengajuan sewa ditolak",
		fmt.Sprintf("%s menolak sewa %s: %s", rental.Owner.Name, rental.Listing.Title, reason),
		rental.ID)

	return s.reload(ctx, ownerID, rental.ID)
}

func (s *Service) Cancel(ctx context.Context, userID, id, reason string) (Rental, error) {
	rental, err := s.loadForParticipant(ctx, userID, id)
	if err != nil {
		return Rental{}, err
	}

	reason = strings.TrimSpace(reason)
	if rental.MyRole == "owner" {
		if rental.Status != StatusApproved {
			return Rental{}, ErrInvalidTransition
		}
		if reason == "" {
			return Rental{}, ErrReasonRequired
		}
	} else if rental.Status != StatusPending && rental.Status != StatusApproved {
		return Rental{}, ErrInvalidTransition
	}

	if err := s.store.UpdateStatus(ctx, rental.ID, StatusCancelled, nil); err != nil {
		return Rental{}, err
	}
	from := rental.Status
	var note *string
	if reason != "" {
		note = &reason
	}
	s.addEvent(ctx, rental.ID, &from, StatusCancelled, userID, note)

	other := rental.Owner
	if rental.MyRole == "owner" {
		other = rental.Renter
	}
	s.notify(ctx, other.ID, "rental_cancelled",
		"Rental dibatalkan",
		fmt.Sprintf("%s membatalkan sewa %s.", displayName(rental, rental.MyRole), rental.Listing.Title),
		rental.ID)

	return s.reload(ctx, userID, rental.ID)
}

func (s *Service) Complete(ctx context.Context, ownerID, id string) (Rental, error) {
	rental, err := s.loadForOwner(ctx, ownerID, id)
	if err != nil {
		return Rental{}, err
	}
	if rental.Status != StatusApproved {
		return Rental{}, ErrInvalidTransition
	}
	end, err := time.ParseInLocation(DateFormat, rental.EndDate, jakarta)
	if err != nil || end.After(todayWIB()) {
		return Rental{}, ErrInvalidTransition
	}

	if err := s.store.UpdateStatus(ctx, rental.ID, StatusCompleted, nil); err != nil {
		return Rental{}, err
	}
	from := rental.Status
	s.addEvent(ctx, rental.ID, &from, StatusCompleted, ownerID, nil)
	s.notify(ctx, rental.Renter.ID, "rental_completed",
		"Rental selesai",
		fmt.Sprintf("Rental %s telah ditandai selesai oleh %s.", rental.Listing.Title, rental.Owner.Name),
		rental.ID)

	return s.reload(ctx, ownerID, rental.ID)
}

func (s *Service) Notifications(ctx context.Context, userID string, unreadOnly bool, limit, offset int) (NotificationPage, error) {
	limit = clampLimit(limit, DefaultListLimit, MaxListLimit)
	if offset < 0 {
		offset = 0
	}
	items, total, unread, err := s.store.Notifications(ctx, userID, unreadOnly, limit, offset)
	if err != nil {
		return NotificationPage{}, err
	}
	return NotificationPage{Items: items, Total: total, UnreadCount: unread, Limit: limit, Offset: offset}, nil
}

func (s *Service) MarkNotificationRead(ctx context.Context, userID, id string) error {
	if !uuidPattern.MatchString(strings.ToLower(strings.TrimSpace(id))) {
		return ErrNotFound
	}
	return s.store.MarkNotificationRead(ctx, userID, id)
}

// ChatAccess memastikan user adalah peserta rental. Dipakai saat menerbitkan tiket WS.
func (s *Service) ChatAccess(ctx context.Context, userID, rentalID string) error {
	_, err := s.loadForParticipant(ctx, userID, rentalID)
	return err
}

func (s *Service) Messages(ctx context.Context, userID, rentalID, afterRaw string, limit int) ([]Message, error) {
	rental, err := s.loadForParticipant(ctx, userID, rentalID)
	if err != nil {
		return nil, err
	}

	var after *time.Time
	if raw := strings.TrimSpace(afterRaw); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil, &ValidationError{Fields: map[string]string{"after": "Format after harus RFC3339"}}
		}
		after = &parsed
	}

	messages, err := s.store.Messages(ctx, rental.ID, after, clampLimit(limit, DefaultChatLimit, MaxChatLimit))
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func (s *Service) SendMessage(ctx context.Context, userID, rentalID, body string) (Message, error) {
	rental, err := s.loadForParticipant(ctx, userID, rentalID)
	if err != nil {
		return Message{}, err
	}
	if rental.Status != StatusPending && rental.Status != StatusApproved {
		return Message{}, ErrChatClosed
	}

	body = strings.TrimSpace(body)
	if n := utf8.RuneCountInString(body); n == 0 || n > maxMessageLength {
		return Message{}, &ValidationError{Fields: map[string]string{
			"body": fmt.Sprintf("Pesan harus 1-%d karakter", maxMessageLength),
		}}
	}
	return s.store.SaveMessage(ctx, rental.ID, userID, body)
}

// AgreementAccess memastikan user adalah peserta rental sebelum membaca PDF.
func (s *Service) AgreementAccess(ctx context.Context, userID, rentalID string) error {
	_, err := s.loadForParticipant(ctx, userID, rentalID)
	return err
}

func (s *Service) loadForParticipant(ctx context.Context, userID, id string) (Rental, error) {
	rental, err := s.store.Get(ctx, id)
	if err != nil {
		return Rental{}, err
	}
	switch strings.TrimSpace(userID) {
	case rental.Renter.ID:
		rental.MyRole = "renter"
	case rental.Owner.ID:
		rental.MyRole = "owner"
	default:
		return Rental{}, ErrForbidden
	}
	return rental, nil
}

func (s *Service) loadForOwner(ctx context.Context, userID, id string) (Rental, error) {
	rental, err := s.loadForParticipant(ctx, userID, id)
	if err != nil {
		return Rental{}, err
	}
	if rental.MyRole != "owner" {
		return Rental{}, ErrForbidden
	}
	return rental, nil
}

func (s *Service) reload(ctx context.Context, userID, id string) (Rental, error) {
	rental, err := s.store.Get(ctx, id)
	if err != nil {
		return Rental{}, err
	}
	events, err := s.store.Events(ctx, id)
	if err != nil {
		return Rental{}, err
	}
	rental.Events = events
	return s.decorate(rental, userID), nil
}

func (s *Service) decorate(rental Rental, userID string) Rental {
	if rental.MyRole == "" {
		switch strings.TrimSpace(userID) {
		case rental.Renter.ID:
			rental.MyRole = "renter"
		case rental.Owner.ID:
			rental.MyRole = "owner"
		}
	}
	rental.AgreementURL = fmt.Sprintf("/api/rentals/%s/agreement/file", rental.ID)
	return rental
}

func (s *Service) addEvent(ctx context.Context, rentalID string, from *Status, to Status, actorID string, note *string) {
	if err := s.store.AddEvent(ctx, rentalID, from, to, actorID, note); err != nil {
		slog.Warn("gagal mencatat riwayat rental", "rental", rentalID, "status", to, "error", err)
	}
}

func (s *Service) notify(ctx context.Context, userID, kind, title, body, rentalID string) {
	if err := s.store.CreateNotification(ctx, userID, kind, title, body, map[string]any{"rental_id": rentalID}); err != nil {
		slog.Warn("gagal membuat notifikasi rental", "user", userID, "type", kind, "error", err)
	}
}

func validateRentalDates(in DateInput) (time.Time, time.Time, error) {
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
		switch {
		case end.Before(start):
			fields["end_date"] = "Tanggal selesai tidak boleh sebelum tanggal mulai"
		case start.Before(todayWIB()):
			fields["start_date"] = "Tanggal mulai tidak boleh di masa lalu"
		case daysBetween(start, end)+1 > maxRentalDays:
			fields["end_date"] = fmt.Sprintf("Durasi maksimal %d hari", maxRentalDays)
		}
	}
	if len(fields) > 0 {
		return time.Time{}, time.Time{}, &ValidationError{Fields: fields}
	}
	return start, end, nil
}

func daysBetween(from, to time.Time) int {
	return int(to.Sub(from).Hours() / 24)
}

func clampLimit(limit, fallback, max int) int {
	switch {
	case limit <= 0:
		return fallback
	case limit > max:
		return max
	default:
		return limit
	}
}

func displayName(r Rental, role string) string {
	if role == "owner" {
		return r.Owner.Name
	}
	return r.Renter.Name
}

func todayWIB() time.Time {
	now := time.Now().In(jakarta)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, jakarta)
}
