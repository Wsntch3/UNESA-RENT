package authotp

import (
	"context"
	"sync"
	"time"
)

type MemoryStore struct {
	mu      sync.Mutex
	records []Record
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (s *MemoryStore) Insert(_ context.Context, rec NewRecord) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	for i := range s.records {
		if s.records[i].Email == rec.Email && s.records[i].Purpose == rec.Purpose &&
			s.records[i].ConsumedAt == nil {
			consumed := now
			s.records[i].ConsumedAt = &consumed
		}
	}

	out := Record{
		ID:            newID(),
		Email:         rec.Email,
		UserID:        rec.UserID,
		Purpose:       rec.Purpose,
		CodeHash:      rec.CodeHash,
		ChallengeHash: rec.ChallengeHash,
		MaxAttempts:   rec.MaxAttempts,
		ExpiresAt:     rec.ExpiresAt,
		RequestIP:     stringPtr(rec.RequestIP),
		CreatedAt:     now,
	}
	s.records = append(s.records, out)
	return out, nil
}

func (s *MemoryStore) CountRecent(_ context.Context, email string, purpose Purpose, window time.Duration) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-window)
	n := 0
	for _, r := range s.records {
		if r.Email == email && r.Purpose == purpose && r.CreatedAt.After(cutoff) {
			n++
		}
	}
	return n, nil
}

func (s *MemoryStore) CountRecentIP(_ context.Context, ip string, window time.Duration) (int, error) {
	if ip == "" {
		return 0, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-window)
	n := 0
	for _, r := range s.records {
		if r.RequestIP != nil && *r.RequestIP == ip && r.CreatedAt.After(cutoff) {
			n++
		}
	}
	return n, nil
}

func (s *MemoryStore) LatestActive(_ context.Context, email string, purpose Purpose) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for i := len(s.records) - 1; i >= 0; i-- {
		r := s.records[i]
		if r.Email == email && r.Purpose == purpose && r.ConsumedAt == nil && r.ExpiresAt.After(now) {
			return r, nil
		}
	}
	return Record{}, ErrNoActive
}

func (s *MemoryStore) IncrementAttempts(_ context.Context, id string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.records {
		if s.records[i].ID == id {
			s.records[i].Attempts++
			return s.records[i].Attempts, nil
		}
	}
	return 0, ErrNoActive
}

func (s *MemoryStore) MarkVerified(_ context.Context, id, challengeHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for i := range s.records {
		if s.records[i].ID == id {
			s.records[i].VerifiedAt = &now
			s.records[i].ChallengeHash = &challengeHash
			return nil
		}
	}
	return ErrNoActive
}

func (s *MemoryStore) Consume(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for i := range s.records {
		if s.records[i].ID == id {
			s.records[i].ConsumedAt = &now
			return nil
		}
	}
	return ErrNoActive
}

func (s *MemoryStore) ConsumeByChallenge(_ context.Context, email string, purpose Purpose, challengeHash string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for i := range s.records {
		r := &s.records[i]
		if r.Email == email && r.Purpose == purpose && r.ChallengeHash != nil &&
			*r.ChallengeHash == challengeHash && r.VerifiedAt != nil &&
			r.ConsumedAt == nil && r.ExpiresAt.After(now) {
			r.ConsumedAt = &now
			return true, nil
		}
	}
	return false, nil
}

func (s *MemoryStore) FindByChallenge(_ context.Context, challengeHash string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for i := len(s.records) - 1; i >= 0; i-- {
		r := s.records[i]
		if r.ChallengeHash != nil && *r.ChallengeHash == challengeHash &&
			r.ConsumedAt == nil && r.ExpiresAt.After(now) {
			return r, nil
		}
	}
	return Record{}, ErrNoActive
}

func stringPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
