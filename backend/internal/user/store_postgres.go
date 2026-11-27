package user

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

const userColumns = `
	id::text,
	name,
	email,
	password_hash,
	role,
	COALESCE(nim, ''),
	COALESCE(study_program_code, ''),
	status,
	token_version,
	email_verified_at,
	last_login_at,
	created_at,
	updated_at`

func (s *PostgresStore) Create(ctx context.Context, u User) (User, error) {
	role := u.Role
	if role == "" {
		role = RoleMahasiswa
	}
	status := u.Status
	if status == "" {
		status = StatusActive
	}

	var nim any
	if u.NIM != "" {
		nim = u.NIM
	}

	row := s.pool.QueryRow(ctx, `
		INSERT INTO users (name, email, password_hash, role, nim, status, email_verified_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING `+userColumns,
		u.Name, u.Email, u.PasswordHash, string(role), nim, string(status), u.EmailVerifiedAt,
	)

	created, err := scanUser(row)
	if err != nil {
		return User{}, mapPostgresError(err)
	}
	return created, nil
}

func (s *PostgresStore) FindByEmail(ctx context.Context, email string) (User, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT `+userColumns+`
		FROM users
		WHERE lower(email) = $1`,
		normalizeEmail(email),
	)

	u, err := scanUser(row)
	if err != nil {
		return User{}, mapPostgresError(err)
	}
	return u, nil
}

func (s *PostgresStore) FindByID(ctx context.Context, id string) (User, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT `+userColumns+`
		FROM users
		WHERE id = $1::uuid`,
		strings.TrimSpace(id),
	)

	u, err := scanUser(row)
	if err != nil {
		return User{}, mapPostgresError(err)
	}
	return u, nil
}

func (s *PostgresStore) FindAuthState(ctx context.Context, id string) (AuthState, error) {
	var a AuthState
	var role, status string
	err := s.pool.QueryRow(ctx,
		`SELECT role, status, token_version FROM users WHERE id = $1::uuid`,
		strings.TrimSpace(id),
	).Scan(&role, &status, &a.TokenVersion)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AuthState{}, ErrNotFound
		}
		return AuthState{}, mapPostgresError(err)
	}
	a.Role, a.Status = Role(role), Status(status)
	return a, nil
}

func (s *PostgresStore) FindNIMStatus(ctx context.Context, nim string) (NIMStatus, error) {
	var (
		userID   string
		verified bool
	)
	err := s.pool.QueryRow(ctx, `
		SELECT u.id::text, is_ktm_verified(u.id)
		  FROM users u WHERE u.nim = $1`,
		strings.TrimSpace(nim),
	).Scan(&userID, &verified)
	if errors.Is(err, pgx.ErrNoRows) {
		return NIMStatus{}, nil
	}
	if err != nil {
		return NIMStatus{}, mapPostgresError(err)
	}
	return NIMStatus{Exists: true, Verified: verified, UserID: userID}, nil
}

func (s *PostgresStore) MarkLogin(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE users SET last_login_at = now() WHERE id = $1::uuid`,
		strings.TrimSpace(userID))
	return mapPostgresError(err)
}

func (s *PostgresStore) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE users
		   SET password_hash = $2,
		       token_version = token_version + 1,
		       updated_at = now()
		 WHERE id = $1::uuid`,
		strings.TrimSpace(userID), passwordHash)
	if err != nil {
		return mapPostgresError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanUser(row pgx.Row) (User, error) {
	var (
		u      User
		role   string
		status string
	)

	err := row.Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.PasswordHash,
		&role,
		&u.NIM,
		&u.StudyProgramCode,
		&status,
		&u.TokenVersion,
		&u.EmailVerifiedAt,
		&u.LastLoginAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}

	u.Role = Role(role)
	u.Status = Status(status)
	return u, nil
}

func mapPostgresError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return err
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case "23505":
		switch pgErr.ConstraintName {
		case "users_nim_key":
			return ErrNIMTaken
		case "users_email_lower_key":
			return ErrEmailTaken
		}
		return err

	case "23514":
		switch pgErr.ConstraintName {
		case "users_nim_format", "users_mahasiswa_nim_required", "users_nim_prodi":
			return ErrInvalidNIM
		}
		return err

	case "22P02":
		return ErrNotFound
	}

	return err
}
