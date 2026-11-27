package authotp

import (
	"context"
	"errors"
	"time"
)

var ErrNoActive = errors.New("tidak ada OTP aktif")

type Record struct {
	ID            string
	Email         string
	UserID        *string
	Purpose       Purpose
	CodeHash      string
	ChallengeHash *string
	Attempts      int
	MaxAttempts   int
	ExpiresAt     time.Time
	VerifiedAt    *time.Time
	ConsumedAt    *time.Time
	RequestIP     *string
	CreatedAt     time.Time
}

type NewRecord struct {
	Email         string
	UserID        *string
	Purpose       Purpose
	CodeHash      string
	ChallengeHash *string
	MaxAttempts   int
	ExpiresAt     time.Time
	RequestIP     string
}

// Store menyimpan OTP. Implementasi Postgres dipakai di produksi; MemoryStore
// dipakai saat DATABASE_URL kosong (mode JSON development).
type Store interface {
	Insert(ctx context.Context, rec NewRecord) (Record, error)
	CountRecent(ctx context.Context, email string, purpose Purpose, window time.Duration) (int, error)
	CountRecentIP(ctx context.Context, ip string, window time.Duration) (int, error)
	LatestActive(ctx context.Context, email string, purpose Purpose) (Record, error)
	IncrementAttempts(ctx context.Context, id string) (int, error)
	MarkVerified(ctx context.Context, id, challengeHash string) error
	Consume(ctx context.Context, id string) error
	ConsumeByChallenge(ctx context.Context, email string, purpose Purpose, challengeHash string) (bool, error)
	FindByChallenge(ctx context.Context, challengeHash string) (Record, error)
}
