package rental

import (
	"context"
	"time"
)

type Store interface {
	IsVerified(ctx context.Context, userID string) (bool, error)
	ListingInfo(ctx context.Context, listingID string) (ownerID string, active bool, err error)

	Create(ctx context.Context, listingID, renterID string, start, end time.Time) (Rental, error)
	Get(ctx context.Context, id string) (Rental, error)
	ListMine(ctx context.Context, userID, status string, limit, offset int) ([]Rental, int, error)
	UpdateStatus(ctx context.Context, id string, to Status, rejectionReason *string) error

	AddEvent(ctx context.Context, rentalID string, from *Status, to Status, actorID string, note *string) error
	Events(ctx context.Context, rentalID string) ([]Event, error)

	Availability(ctx context.Context, listingID string, from, to time.Time) ([]Unavailable, error)

	CreateNotification(ctx context.Context, userID, kind, title, body string, data map[string]any) error
	Notifications(ctx context.Context, userID string, unreadOnly bool, limit, offset int) ([]Notification, int, int, error)
	MarkNotificationRead(ctx context.Context, userID, id string) error

	SaveMessage(ctx context.Context, rentalID, senderID, body string) (Message, error)
	Messages(ctx context.Context, rentalID string, after *time.Time, limit int) ([]Message, error)
}
