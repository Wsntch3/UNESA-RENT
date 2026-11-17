package ktm

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore menyimpan metadata verifikasi KTM di PostgreSQL.
// File fisik tidak pernah masuk database; hanya storage_key.
type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// Urutan kolom HARUS sama dengan urutan Scan di scanSubmission.
const submissionColumns = `
	k.id::text,
	k.user_id::text,
	u.name,
	u.email,
	COALESCE(u.nim, ''),
	k.storage_key,
	k.mime_type,
	k.file_size_bytes,
	k.status,
	k.submitted_at,
	k.reviewed_at,
	k.rejection_reason,
	k.revoked_at,
	k.revoke_reason,
	k.reviewer_confirmed_nim,
	k.file_purged_at,
	k.created_at,
	k.updated_at`

const submissionFrom = `
	FROM ktm_verifications k
	JOIN users u ON u.id = k.user_id`

func (s *PostgresStore) Create(ctx context.Context, in NewSubmission) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO ktm_verifications (user_id, storage_key, mime_type, file_size_bytes, checksum_sha256)
		VALUES ($1::uuid, $2, $3, $4, $5)
		RETURNING id::text`,
		in.UserID, in.StorageKey, in.MimeType, in.FileSizeBytes, in.ChecksumSHA256,
	).Scan(&id)
	if err != nil {
		return "", mapPostgresError(err)
	}
	return id, nil
}

func (s *PostgresStore) GetByID(ctx context.Context, id string) (Submission, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+submissionColumns+submissionFrom+` WHERE k.id = $1::uuid`,
		strings.TrimSpace(id),
	)
	sub, err := scanSubmission(row)
	if err != nil {
		return Submission{}, mapPostgresError(err)
	}
	return sub, nil
}

func (s *PostgresStore) ListByUser(ctx context.Context, userID string, limit int) ([]Submission, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+submissionColumns+submissionFrom+`
		 WHERE k.user_id = $1::uuid
		 ORDER BY k.submitted_at DESC
		 LIMIT $2`,
		strings.TrimSpace(userID), limit,
	)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	return collectSubmissions(rows)
}

func (s *PostgresStore) ListByStatus(ctx context.Context, status string, limit, offset int) ([]Submission, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+submissionColumns+submissionFrom+`
		 WHERE k.status = $1
		 ORDER BY k.submitted_at ASC
		 LIMIT $2 OFFSET $3`,
		status, limit, offset,
	)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	return collectSubmissions(rows)
}

func (s *PostgresStore) UserState(ctx context.Context, userID string) (UserState, error) {
	var st UserState
	err := s.pool.QueryRow(ctx, `
		SELECT u.id::text, u.name, u.email, COALESCE(u.nim, ''), u.role, u.status,
		       is_ktm_verified(u.id)
		  FROM users u
		 WHERE u.id = $1::uuid`,
		strings.TrimSpace(userID),
	).Scan(&st.ID, &st.Name, &st.Email, &st.NIM, &st.Role, &st.Status, &st.Verified)
	if err != nil {
		return UserState{}, mapPostgresError(err)
	}
	return st, nil
}

func (s *PostgresStore) MarkVerified(ctx context.Context, id, reviewerID string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE ktm_verifications
		   SET status = 'verified',
		       reviewed_by = $2::uuid,
		       reviewed_at = now(),
		       reviewer_confirmed_nim = true
		 WHERE id = $1::uuid
		   AND status = 'pending'`,
		strings.TrimSpace(id), strings.TrimSpace(reviewerID),
	)
	if err != nil {
		return mapPostgresError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInvalidTransition
	}
	return nil
}

func (s *PostgresStore) MarkRejected(ctx context.Context, id, reviewerID, reason string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE ktm_verifications
		   SET status = 'rejected',
		       reviewed_by = $2::uuid,
		       reviewed_at = now(),
		       rejection_reason = $3
		 WHERE id = $1::uuid
		   AND status = 'pending'`,
		strings.TrimSpace(id), strings.TrimSpace(reviewerID), strings.TrimSpace(reason),
	)
	if err != nil {
		return mapPostgresError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInvalidTransition
	}
	return nil
}

func (s *PostgresStore) MarkRevoked(ctx context.Context, id, reviewerID, reason string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE ktm_verifications
		   SET status = 'revoked',
		       revoked_by = $2::uuid,
		       revoked_at = now(),
		       revoke_reason = $3
		 WHERE id = $1::uuid
		   AND status = 'verified'`,
		strings.TrimSpace(id), strings.TrimSpace(reviewerID), strings.TrimSpace(reason),
	)
	if err != nil {
		return mapPostgresError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInvalidTransition
	}
	return nil
}

func (s *PostgresStore) LogAction(ctx context.Context, actorID, action, targetID, reason string) error {
	var reasonArg any
	if trimmed := strings.TrimSpace(reason); trimmed != "" {
		reasonArg = trimmed
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO moderation_actions (actor_id, action, target_type, target_id, reason)
		VALUES ($1::uuid, $2, 'ktm_verification', $3::uuid, $4)`,
		strings.TrimSpace(actorID), action, strings.TrimSpace(targetID), reasonArg,
	)
	return err
}

func collectSubmissions(rows pgx.Rows) ([]Submission, error) {
	list := make([]Submission, 0)
	for rows.Next() {
		sub, err := scanSubmission(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}
	return list, nil
}

func scanSubmission(row pgx.Row) (Submission, error) {
	var (
		sub    Submission
		status string
	)

	err := row.Scan(
		&sub.ID,
		&sub.UserID,
		&sub.UserName,
		&sub.UserEmail,
		&sub.UserNIM,
		&sub.StorageKey,
		&sub.MimeType,
		&sub.FileSizeBytes,
		&status,
		&sub.SubmittedAt,
		&sub.ReviewedAt,
		&sub.RejectionReason,
		&sub.RevokedAt,
		&sub.RevokeReason,
		&sub.ReviewerConfirmedNIM,
		&sub.FilePurgedAt,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
	if err != nil {
		return Submission{}, err
	}
	sub.Status = Status(status)
	return sub, nil
}

// mapPostgresError menerjemahkan constraint/trigger PostgreSQL menjadi error domain.
func mapPostgresError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case "23505": // unique_violation
		if pgErr.ConstraintName == "ktm_one_pending_per_user" {
			return ErrPendingExists
		}
	case "23514": // check_violation atau RAISE dari trigger
		switch pgErr.ConstraintName {
		case "users_nim_format", "users_nim_prodi", "users_mahasiswa_nim_required":
			return ErrIneligible
		}
		switch {
		case strings.Contains(pgErr.Message, "sudah terverifikasi"):
			return ErrAlreadyVerified
		case strings.Contains(pgErr.Message, "hanya mahasiswa"):
			return ErrIneligible
		case strings.Contains(pgErr.Message, "akun tidak aktif"):
			return ErrIneligible
		case strings.Contains(pgErr.Message, "transisi status"):
			return ErrInvalidTransition
		}
	case "22P02": // invalid_text_representation, mis. id bukan UUID
		return ErrNotFound
	}

	return err
}
