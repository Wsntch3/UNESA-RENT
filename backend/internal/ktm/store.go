package ktm

import "context"

// Store adalah kontrak penyimpanan metadata verifikasi KTM.
// Implementasi saat ini: PostgresStore.
type Store interface {
	Create(ctx context.Context, in NewSubmission) (string, error)
	GetByID(ctx context.Context, id string) (Submission, error)
	ListByUser(ctx context.Context, userID string, limit int) ([]Submission, error)
	ListByStatus(ctx context.Context, status string, limit, offset int) ([]Submission, error)
	UserState(ctx context.Context, userID string) (UserState, error)

	MarkVerified(ctx context.Context, id, reviewerID string) error
	MarkRejected(ctx context.Context, id, reviewerID, reason string) error
	MarkRevoked(ctx context.Context, id, reviewerID, reason string) error

	LogAction(ctx context.Context, actorID, action, targetID, reason string) error
}
