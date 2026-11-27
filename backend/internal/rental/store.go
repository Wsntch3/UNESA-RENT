package rental

import (
	"context"
	"time"
)

type Store interface {
	IsVerified(ctx context.Context, userID string) (bool, error)
	ParticipantNIM(ctx context.Context, userID string) (string, error)
	DefaultLocation(ctx context.Context) string
	ListingInfo(ctx context.Context, listingID string) (ownerID string, active bool, err error)

	Create(ctx context.Context, listingID, renterID string, start, end time.Time) (Rental, error)
	Get(ctx context.Context, id string) (Rental, error)
	ListMine(ctx context.Context, userID, role, status, from, to, query string, limit, offset int) ([]Rental, int, error)
	AdminList(ctx context.Context, filter RentalFilter) ([]Rental, int, error)
	AdminStats(ctx context.Context) (RentalStats, error)
	History(ctx context.Context, userID, role string, year int) ([]HistoryMonth, error)
	ExportRows(ctx context.Context, filter RentalFilter) ([]Rental, error)
	UpdateStatus(ctx context.Context, id string, to Status, rejectionReason *string) error

	AddEvent(ctx context.Context, rentalID string, from *Status, to Status, actorID string, note *string) error
	Events(ctx context.Context, rentalID string) ([]Event, error)

	Availability(ctx context.Context, listingID string, from, to time.Time) ([]Unavailable, error)

	CreateNotification(ctx context.Context, userID, kind, title, body string, data map[string]any) error
	Notifications(ctx context.Context, userID string, unreadOnly bool, limit, offset int) ([]Notification, int, int, error)
	MarkNotificationRead(ctx context.Context, userID, id string) error

	SaveMessage(ctx context.Context, rentalID, senderID string, ciphertext, nonce []byte, version, length int, sha256 string) (Message, error)
	Messages(ctx context.Context, rentalID string, after *time.Time, limit int) ([]Message, error)
	AdminMessages(ctx context.Context, rentalID string, limit, offset int) ([]AdminMessage, int, error)
	AdminMessageByID(ctx context.Context, rentalID, messageID string) (Message, error)
	InsertChatAccessLog(ctx context.Context, adminID, rentalID, messageID, action, reason, ip, userAgent string) error

	EnsureCOD(ctx context.Context, rentalID string) error
	GetCOD(ctx context.Context, rentalID string) (CODState, error)
	CODAgreed(ctx context.Context, rentalID string) error
	CODHandover(ctx context.Context, rentalID, actorID, note string, proof []byte, mimeType string) (Document, error)
	CODHandoverConfirm(ctx context.Context, rentalID, actorID string) error
	CODReturn(ctx context.Context, rentalID, actorID, note string, proof []byte, mimeType string) (Document, error)
	CODReturnConfirm(ctx context.Context, rentalID, actorID string) error
	CODSettle(ctx context.Context, rentalID string, in CODSettleInput) error

	SaveDocument(ctx context.Context, rentalID, docType, key, mimeType string, size int64, checksum, uploadedBy, visibility string, metadata map[string]any) (Document, error)
	Documents(ctx context.Context, rentalID string) ([]Document, error)
	DocumentByID(ctx context.Context, rentalID, docID string) (Document, error)
	DocumentBytes(ctx context.Context, documentID string) ([]byte, error)
}
