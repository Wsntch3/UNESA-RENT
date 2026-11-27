package authotp

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) Insert(ctx context.Context, rec NewRecord) (Record, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Record{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE auth_otps
		   SET consumed_at = now()
		 WHERE lower(email) = lower($1) AND purpose = $2 AND consumed_at IS NULL`,
		rec.Email, string(rec.Purpose),
	); err != nil {
		return Record{}, err
	}

	var (
		out       Record
		purpose   string
		userID    *string
		challenge *string
		requestIP *string
	)
	err = tx.QueryRow(ctx, `
		INSERT INTO auth_otps
			(email, user_id, purpose, code_hash, challenge_hash, max_attempts, expires_at, request_ip)
		VALUES (lower($1), $2::uuid, $3, $4, $5, $6, $7, $8)
		RETURNING id::text, email, user_id::text, purpose, code_hash, challenge_hash,
		          attempts, max_attempts, expires_at, verified_at, consumed_at, request_ip, created_at`,
		rec.Email, rec.UserID, string(rec.Purpose), rec.CodeHash, rec.ChallengeHash,
		rec.MaxAttempts, rec.ExpiresAt, nullableString(rec.RequestIP),
	).Scan(
		&out.ID, &out.Email, &userID, &purpose, &out.CodeHash, &challenge,
		&out.Attempts, &out.MaxAttempts, &out.ExpiresAt, &out.VerifiedAt,
		&out.ConsumedAt, &requestIP, &out.CreatedAt,
	)
	if err != nil {
		return Record{}, err
	}
	out.Purpose = Purpose(purpose)
	out.UserID = userID
	out.ChallengeHash = challenge
	out.RequestIP = requestIP

	if err := tx.Commit(ctx); err != nil {
		return Record{}, err
	}
	return out, nil
}

func (s *PostgresStore) CountRecent(ctx context.Context, email string, purpose Purpose, window time.Duration) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM auth_otps
		 WHERE lower(email) = lower($1) AND purpose = $2 AND created_at > now() - $3::interval`,
		email, string(purpose), window.String(),
	).Scan(&n)
	return n, err
}

func (s *PostgresStore) CountRecentIP(ctx context.Context, ip string, window time.Duration) (int, error) {
	if ip == "" {
		return 0, nil
	}
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM auth_otps
		 WHERE request_ip = $1 AND created_at > now() - $2::interval`,
		ip, window.String(),
	).Scan(&n)
	return n, err
}

func (s *PostgresStore) LatestActive(ctx context.Context, email string, purpose Purpose) (Record, error) {
	rec, err := scanRecord(s.pool.QueryRow(ctx, `
		SELECT id::text, email, user_id::text, purpose, code_hash, challenge_hash,
		       attempts, max_attempts, expires_at, verified_at, consumed_at, request_ip, created_at
		  FROM auth_otps
		 WHERE lower(email) = lower($1) AND purpose = $2
		   AND consumed_at IS NULL AND expires_at > now()
		 ORDER BY created_at DESC
		 LIMIT 1`,
		email, string(purpose),
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return Record{}, ErrNoActive
	}
	return rec, err
}

func (s *PostgresStore) IncrementAttempts(ctx context.Context, id string) (int, error) {
	var attempts int
	err := s.pool.QueryRow(ctx, `
		UPDATE auth_otps SET attempts = attempts + 1
		 WHERE id = $1::uuid
		 RETURNING attempts`, id).Scan(&attempts)
	return attempts, err
}

func (s *PostgresStore) MarkVerified(ctx context.Context, id, challengeHash string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE auth_otps SET verified_at = now(), challenge_hash = $2
		 WHERE id = $1::uuid`, id, challengeHash)
	return err
}

func (s *PostgresStore) Consume(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE auth_otps SET consumed_at = now()
		 WHERE id = $1::uuid AND consumed_at IS NULL`, id)
	return err
}

func (s *PostgresStore) ConsumeByChallenge(ctx context.Context, email string, purpose Purpose, challengeHash string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE auth_otps SET consumed_at = now()
		 WHERE lower(email) = lower($1) AND purpose = $2 AND challenge_hash = $3
		   AND verified_at IS NOT NULL AND consumed_at IS NULL AND expires_at > now()`,
		email, string(purpose), challengeHash)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *PostgresStore) FindByChallenge(ctx context.Context, challengeHash string) (Record, error) {
	rec, err := scanRecord(s.pool.QueryRow(ctx, `
		SELECT id::text, email, user_id::text, purpose, code_hash, challenge_hash,
		       attempts, max_attempts, expires_at, verified_at, consumed_at, request_ip, created_at
		  FROM auth_otps
		 WHERE challenge_hash = $1 AND consumed_at IS NULL AND expires_at > now()
		 ORDER BY created_at DESC
		 LIMIT 1`, challengeHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return Record{}, ErrNoActive
	}
	return rec, err
}

func scanRecord(row pgx.Row) (Record, error) {
	var (
		out       Record
		purpose   string
		userID    *string
		challenge *string
		requestIP *string
	)
	err := row.Scan(
		&out.ID, &out.Email, &userID, &purpose, &out.CodeHash, &challenge,
		&out.Attempts, &out.MaxAttempts, &out.ExpiresAt, &out.VerifiedAt,
		&out.ConsumedAt, &requestIP, &out.CreatedAt,
	)
	if err != nil {
		return Record{}, err
	}
	out.Purpose = Purpose(purpose)
	out.UserID = userID
	out.ChallengeHash = challenge
	out.RequestIP = requestIP
	return out, nil
}

func nullableString(v string) any {
	if v == "" {
		return nil
	}
	return v
}
