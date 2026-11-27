package rental

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"unesarent/backend/internal/chatcrypto"
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
	store         Store
	cipher        *chatcrypto.Cipher
	files         Storage
	requireReturn bool
}

func NewService(store Store) *Service {
	return &Service{store: store, requireReturn: true}
}

// WithCrypto memasang enkripsi chat dan penyimpanan dokumen.
func (s *Service) WithCrypto(cipher *chatcrypto.Cipher, files Storage) *Service {
	s.cipher, s.files = cipher, files
	return s
}

func (s *Service) WithCODRequireReturn(required bool) *Service {
	s.requireReturn = required
	return s
}

func (s *Service) decrypt(m Message) (Message, error) {
	if s.cipher == nil {
		return m, nil
	}
	if len(m.Ciphertext) == 0 {
		return m, nil
	}
	body, err := s.cipher.Decrypt(m.Ciphertext, m.Nonce, m.KeyVersion)
	if err != nil {
		return Message{}, err
	}
	m.Body = body
	return m, nil
}

func messageHash(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
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

	items, total, err := s.store.ListMine(ctx, userID, "", status, "", "", "", limit, offset)
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

// AdminGet membaca rental tanpa memeriksa kepesertaan.
func (s *Service) AdminGet(ctx context.Context, id string) (Rental, error) {
	rental, err := s.store.Get(ctx, id)
	if err != nil {
		return Rental{}, err
	}
	events, err := s.store.Events(ctx, id)
	if err != nil {
		return Rental{}, err
	}
	rental.Events = events
	return s.decorate(rental, ""), nil
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
	if err := s.prepareAgreement(ctx, rental); err != nil {
		slog.Warn("gagal membuat perjanjian COD", "rental", rental.ID, "error", err)
	}

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

	if cod, err := s.store.GetCOD(ctx, rental.ID); err == nil && cod.HandoverAt != nil {
		return Rental{}, ErrInvalidCODState
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

	if s.requireReturn {
		if cod, err := s.store.GetCOD(ctx, rental.ID); err == nil {
			if cod.HandoverConfirmedAt == nil || cod.ReturnConfirmedAt == nil {
				return Rental{}, ErrCODIncomplete
			}
		}
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
	for i, msg := range messages {
		plain, err := s.decrypt(msg)
		if err != nil {
			return nil, err
		}
		messages[i] = plain
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
	var ciphertext, nonce []byte
	var version int
	if s.cipher != nil {
		ciphertext, nonce, version, err = s.cipher.Encrypt(body)
		if err != nil {
			return Message{}, err
		}
	}
	return s.store.SaveMessage(ctx, rental.ID, userID, ciphertext, nonce, version, utf8.RuneCountInString(body), messageHash(body))
}

// AgreementAccess memastikan user adalah peserta rental sebelum membaca PDF.
func (s *Service) AgreementAccess(ctx context.Context, userID, rentalID string) error {
	_, err := s.loadForParticipant(ctx, userID, rentalID)
	return err
}

// AgreementFile mengembalikan surat perjanjian rental. Dokumen dibuat sekali
// saat rental disetujui; untuk data lama dibuat ulang secara on-demand.
func (s *Service) AgreementFile(ctx context.Context, userID, rentalID string) (Document, []byte, error) {
	rentalItem, err := s.loadForParticipant(ctx, userID, rentalID)
	if err != nil {
		return Document{}, nil, err
	}
	docs, err := s.store.Documents(ctx, rentalID)
	if err != nil {
		return Document{}, nil, err
	}
	for _, doc := range docs {
		if doc.Type != "agreement" {
			continue
		}
		data, err := s.store.DocumentBytes(ctx, doc.ID)
		if err != nil {
			return Document{}, nil, err
		}
		return doc, data, nil
	}
	if s.files == nil {
		return Document{}, nil, fmt.Errorf("penyimpanan dokumen belum dikonfigurasi")
	}
	ownerNIM, _ := s.store.ParticipantNIM(ctx, rentalItem.Owner.ID)
	renterNIM, _ := s.store.ParticipantNIM(ctx, rentalItem.Renter.ID)
	pdf, err := BuildAgreementPDF(rentalItem, ownerNIM, renterNIM, s.store.DefaultLocation(ctx))
	if err != nil {
		return Document{}, nil, err
	}
	key, err := newDocumentKey(rentalID, ".pdf")
	if err != nil {
		return Document{}, nil, err
	}
	if err := s.files.Save(ctx, key, pdf); err != nil {
		return Document{}, nil, err
	}
	doc, err := s.store.SaveDocument(ctx, rentalID, "agreement", key, "application/pdf",
		int64(len(pdf)), checksumHex(pdf), "", "participants", map[string]any{})
	if err != nil {
		return Document{}, nil, err
	}
	return doc, pdf, nil
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

// prepareAgreement membuat state COD dan perjanjian PDF saat rental disetujui.
func (s *Service) prepareAgreement(ctx context.Context, rental Rental) error {
	if s.files == nil {
		return nil
	}
	if err := s.store.EnsureCOD(ctx, rental.ID); err != nil {
		return err
	}
	if err := s.store.CODAgreed(ctx, rental.ID); err != nil {
		return err
	}
	ownerNIM, err := s.store.ParticipantNIM(ctx, rental.Owner.ID)
	if err != nil {
		return err
	}
	renterNIM, err := s.store.ParticipantNIM(ctx, rental.Renter.ID)
	if err != nil {
		return err
	}
	location := s.store.DefaultLocation(ctx)
	pdf, err := BuildAgreementPDF(rental, ownerNIM, renterNIM, location)
	if err != nil {
		return err
	}
	key, err := newDocumentKey(rental.ID, ".pdf")
	if err != nil {
		return err
	}
	if err := s.files.Save(ctx, key, pdf); err != nil {
		return err
	}
	_, err = s.store.SaveDocument(ctx, rental.ID, "agreement", key, "application/pdf",
		int64(len(pdf)), checksumHex(pdf), "", "participants", map[string]any{"location": location})
	return err
}

func (s *Service) AdminMessageMetadata(ctx context.Context, rentalID string, limit, offset int) ([]AdminMessage, int, error) {
	return s.store.AdminMessages(ctx, strings.TrimSpace(rentalID), clampLimit(limit, DefaultChatLimit, MaxChatLimit), offset)
}

func (s *Service) AdminRevealMessage(ctx context.Context, adminID, rentalID, messageID, reason, ip, userAgent string) (string, error) {
	reason = strings.TrimSpace(reason)
	if n := utf8.RuneCountInString(reason); n < 10 || n > 1000 {
		return "", ErrInvalidReason
	}
	msg, err := s.store.AdminMessageByID(ctx, rentalID, messageID)
	if err != nil {
		return "", err
	}
	plain, err := s.decrypt(msg)
	if err != nil {
		return "", err
	}
	if err := s.store.InsertChatAccessLog(ctx, adminID, rentalID, messageID, "reveal", reason, ip, userAgent); err != nil {
		return "", err
	}
	return plain.Body, nil
}

func (s *Service) AdminExportTranscript(ctx context.Context, adminID, rentalID, reason, ip, userAgent string) ([]byte, error) {
	reason = strings.TrimSpace(reason)
	if n := utf8.RuneCountInString(reason); n < 10 || n > 1000 {
		return nil, ErrInvalidReason
	}
	rentalItem, err := s.store.Get(ctx, rentalID)
	if err != nil {
		return nil, err
	}
	messages, err := s.store.Messages(ctx, rentalID, nil, MaxChatLimit*10)
	if err != nil {
		return nil, err
	}
	var out strings.Builder
	fmt.Fprintf(&out, "TRANSKRIP CHAT UNESA RENT\nRental: %s\nBarang: %s\nDiekspor: %s\n\n",
		rentalItem.ID, rentalItem.Listing.Title, time.Now().Format("2006-01-02 15:04"))
	for _, msg := range messages {
		plain, err := s.decrypt(msg)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&out, "[%s] %s: %s\n", plain.CreatedAt.Format("2006-01-02 15:04"), plain.Sender.Name, plain.Body)
	}
	if err := s.store.InsertChatAccessLog(ctx, adminID, rentalID, "", "export_transcript", reason, ip, userAgent); err != nil {
		return nil, err
	}
	if s.files != nil {
		key, err := newDocumentKey(rentalID, ".pdf")
		if err == nil {
			data := []byte(latin1(out.String()))
			if err := s.files.Save(ctx, key, data); err == nil {
				_, _ = s.store.SaveDocument(ctx, rentalID, "other", key, "application/pdf",
					int64(len(data)), checksumHex(data), adminID, "admin", map[string]any{"reason": reason})
			} else {
				slog.Warn("gagal mengarsipkan transkrip chat", "error", err)
			}
		}
	}
	return []byte(out.String()), nil
}

func (s *Service) COD(ctx context.Context, userID, rentalID string) (CODState, error) {
	if _, err := s.loadForParticipant(ctx, userID, rentalID); err != nil {
		return CODState{}, err
	}
	return s.store.GetCOD(ctx, rentalID)
}

func (s *Service) Handover(ctx context.Context, userID, rentalID, note string, proof []byte, mimeType string) (Document, error) {
	rentalItem, err := s.loadForOwner(ctx, userID, rentalID)
	if err != nil {
		return Document{}, err
	}
	if rentalItem.Status != StatusApproved {
		return Document{}, ErrInvalidTransition
	}
	if mimeType != "image/jpeg" && mimeType != "image/png" {
		return Document{}, &ValidationError{Fields: map[string]string{"file": "Bukti harus JPG atau PNG"}}
	}
	doc, err := s.store.CODHandover(ctx, rentalID, userID, note, proof, mimeType)
	if err != nil {
		return Document{}, err
	}
	s.addEvent(ctx, rentalID, nil, StatusApproved, userID, ptr("Barang diserahkan"))
	s.notify(ctx, rentalItem.Renter.ID, "cod_handover", "Barang diserahkan",
		fmt.Sprintf("%s menyerahkan %s. Silakan konfirmasi.", rentalItem.Owner.Name, rentalItem.Listing.Title), rentalID)
	return s.decorateDocument(doc), nil
}

func (s *Service) ConfirmHandover(ctx context.Context, userID, rentalID string) (CODState, error) {
	rentalItem, err := s.loadForParticipant(ctx, userID, rentalID)
	if err != nil {
		return CODState{}, err
	}
	if rentalItem.MyRole != "renter" {
		return CODState{}, ErrForbidden
	}
	if err := s.store.CODHandoverConfirm(ctx, rentalID, userID); err != nil {
		return CODState{}, err
	}
	s.addEvent(ctx, rentalID, nil, StatusApproved, userID, ptr("Serah terima dikonfirmasi penyewa"))
	return s.store.GetCOD(ctx, rentalID)
}

func (s *Service) Return(ctx context.Context, userID, rentalID, note string, proof []byte, mimeType string) (Document, error) {
	rentalItem, err := s.loadForParticipant(ctx, userID, rentalID)
	if err != nil {
		return Document{}, err
	}
	if rentalItem.MyRole != "renter" || rentalItem.Status != StatusApproved {
		return Document{}, ErrForbidden
	}
	if mimeType != "image/jpeg" && mimeType != "image/png" {
		return Document{}, &ValidationError{Fields: map[string]string{"file": "Bukti harus JPG atau PNG"}}
	}
	doc, err := s.store.CODReturn(ctx, rentalID, userID, note, proof, mimeType)
	if err != nil {
		return Document{}, err
	}
	s.addEvent(ctx, rentalID, nil, StatusApproved, userID, ptr("Barang dikembalikan"))
	s.notify(ctx, rentalItem.Owner.ID, "cod_return", "Barang dikembalikan",
		fmt.Sprintf("%s mengembalikan %s. Silakan konfirmasi.", rentalItem.Renter.Name, rentalItem.Listing.Title), rentalID)
	return s.decorateDocument(doc), nil
}

func (s *Service) ConfirmReturn(ctx context.Context, userID, rentalID string) (CODState, error) {
	if _, err := s.loadForOwner(ctx, userID, rentalID); err != nil {
		return CODState{}, err
	}
	if err := s.store.CODReturnConfirm(ctx, rentalID, userID); err != nil {
		return CODState{}, err
	}
	s.addEvent(ctx, rentalID, nil, StatusApproved, userID, ptr("Pengembalian dikonfirmasi penyedia"))
	return s.store.GetCOD(ctx, rentalID)
}

func (s *Service) Settle(ctx context.Context, userID, rentalID string, in CODSettleInput) (Document, error) {
	rentalItem, err := s.loadForOwner(ctx, userID, rentalID)
	if err != nil {
		return Document{}, err
	}
	if rentalItem.Status != StatusApproved && rentalItem.Status != StatusCompleted {
		return Document{}, ErrInvalidTransition
	}
	if in.AmountPaid < 0 {
		return Document{}, &ValidationError{Fields: map[string]string{"amount_paid": "Jumlah tidak boleh negatif"}}
	}
	in.PaymentMethod = strings.ToLower(strings.TrimSpace(in.PaymentMethod))
	if in.PaymentMethod != "cash" && in.PaymentMethod != "transfer" {
		return Document{}, &ValidationError{Fields: map[string]string{"payment_method": "Metode harus cash atau transfer"}}
	}
	cod, err := s.store.GetCOD(ctx, rentalID)
	if err != nil {
		return Document{}, err
	}
	rentalItem, err = s.store.Get(ctx, rentalID)
	if err != nil {
		return Document{}, err
	}
	ownerNIM, _ := s.store.ParticipantNIM(ctx, rentalItem.Owner.ID)
	renterNIM, _ := s.store.ParticipantNIM(ctx, rentalItem.Renter.ID)
	if err := s.store.CODSettle(ctx, rentalID, in); err != nil {
		return Document{}, err
	}
	settledAt := time.Now().In(jakarta)
	cod.AmountPaid = &in.AmountPaid
	cod.PaymentMethod = &in.PaymentMethod
	cod.Notes = ptr(in.Notes)
	cod.SettledAt = &settledAt
	if s.files == nil {
		return Document{}, fmt.Errorf("penyimpanan dokumen belum dikonfigurasi")
	}
	// Kuitansi dihasilkan sekali setelah pembayaran dicatat agar isinya final.
	key, err := newDocumentKey(rentalID, ".pdf")
	if err != nil {
		return Document{}, err
	}
	pdf, err := BuildReceiptPDF(rentalItem, CODSummary{
		RentalID:  rentalID,
		CODState:  cod,
		Location:  s.store.DefaultLocation(ctx),
		OwnerNIM:  ownerNIM,
		RenterNIM: renterNIM,
	})
	if err != nil {
		return Document{}, err
	}
	if err := s.files.Save(ctx, key, pdf); err != nil {
		return Document{}, err
	}
	doc, err := s.store.SaveDocument(ctx, rentalID, "cod_receipt", key, "application/pdf",
		int64(len(pdf)), checksumHex(pdf), userID, "participants", map[string]any{"amount_paid": in.AmountPaid})
	if err != nil {
		return Document{}, err
	}
	s.addEvent(ctx, rentalID, nil, StatusApproved, userID, ptr("Pembayaran dicatat"))
	return s.decorateDocument(doc), nil
}

func (s *Service) Documents(ctx context.Context, userID, rentalID string) ([]Document, error) {
	if _, err := s.loadForParticipant(ctx, userID, rentalID); err != nil {
		return nil, err
	}
	list, err := s.store.Documents(ctx, rentalID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i] = s.decorateDocument(list[i])
	}
	return list, nil
}

func (s *Service) AdminDocuments(ctx context.Context, rentalID string) ([]Document, error) {
	if _, err := s.store.Get(ctx, rentalID); err != nil {
		return nil, err
	}
	list, err := s.store.Documents(ctx, rentalID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i].DownloadURL = "/api/admin/rentals/" + rentalID + "/documents/" + list[i].ID + "/download"
	}
	return list, nil
}

func (s *Service) DocumentFile(ctx context.Context, userID, rentalID, docID string) (Document, []byte, error) {
	if _, err := s.loadForParticipant(ctx, userID, rentalID); err != nil {
		return Document{}, nil, err
	}
	doc, err := s.store.DocumentByID(ctx, rentalID, docID)
	if err != nil {
		return Document{}, nil, err
	}
	data, err := s.store.DocumentBytes(ctx, doc.ID)
	return doc, data, err
}

func (s *Service) AdminDocumentFile(ctx context.Context, rentalID, docID string) (Document, []byte, error) {
	if _, err := s.store.Get(ctx, rentalID); err != nil {
		return Document{}, nil, err
	}
	doc, err := s.store.DocumentByID(ctx, rentalID, docID)
	if err != nil {
		return Document{}, nil, err
	}
	data, err := s.store.DocumentBytes(ctx, doc.ID)
	return doc, data, err
}

func (s *Service) decorateDocument(doc Document) Document {
	doc.DownloadURL = "/api/rentals/" + doc.RentalID + "/documents/" + doc.ID + "/file"
	return doc
}

func (s *Service) MineFiltered(ctx context.Context, userID string, filter RentalFilter) (RentalPage, error) {
	status := strings.TrimSpace(strings.ToLower(filter.Status))
	switch Status(status) {
	case "", StatusPending, StatusApproved, StatusRejected, StatusCancelled, StatusCompleted:
	default:
		return RentalPage{}, &ValidationError{Fields: map[string]string{"status": "Status rental tidak valid"}}
	}
	filter.Status = status
	filter.Limit = clampLimit(filter.Limit, DefaultListLimit, MaxListLimit)
	items, total, err := s.store.ListMine(ctx, userID, filter.Role, filter.Status, filter.From, filter.To, filter.Query, filter.Limit, filter.Offset)
	if err != nil {
		return RentalPage{}, err
	}
	for i := range items {
		items[i] = s.decorate(items[i], userID)
	}
	return RentalPage{Items: items, Total: total, Limit: filter.Limit, Offset: filter.Offset}, nil
}

func (s *Service) History(ctx context.Context, userID string, role string, yearRaw string) (HistorySummary, error) {
	year := time.Now().Year()
	if strings.TrimSpace(yearRaw) != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(yearRaw))
		if err != nil || parsed < 2000 || parsed > year+1 {
			return HistorySummary{}, &ValidationError{Fields: map[string]string{"year": "Tahun tidak valid"}}
		}
		year = parsed
	}
	months, err := s.store.History(ctx, userID, strings.ToLower(strings.TrimSpace(role)), year)
	if err != nil {
		return HistorySummary{}, err
	}
	summary := HistorySummary{Year: year, Months: months}
	for _, m := range months {
		summary.Total += m.Total
		summary.Value += m.Value
	}
	return summary, nil
}

func (s *Service) AdminList(ctx context.Context, filter RentalFilter) (RentalPage, error) {
	filter.Status = strings.TrimSpace(strings.ToLower(filter.Status))
	filter.Limit = clampLimit(filter.Limit, DefaultListLimit, MaxListLimit)
	items, total, err := s.store.AdminList(ctx, filter)
	if err != nil {
		return RentalPage{}, err
	}
	for i := range items {
		items[i] = s.decorate(items[i], "")
	}
	return RentalPage{Items: items, Total: total, Limit: filter.Limit, Offset: filter.Offset}, nil
}

func (s *Service) AdminStats(ctx context.Context) (RentalStats, error) {
	return s.store.AdminStats(ctx)
}

func (s *Service) ExportRentals(ctx context.Context, filter RentalFilter) ([]Rental, error) {
	filter.Status = strings.TrimSpace(strings.ToLower(filter.Status))
	return s.store.ExportRows(ctx, filter)
}

func ptr(raw string) *string {
	if raw == "" {
		return nil
	}
	return &raw
}
