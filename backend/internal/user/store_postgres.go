package user

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore menyimpan user di tabel users (PostgreSQL).
type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// userColumns dipakai di semua query agar urutan Scan konsisten.
// id dan kode prodi di-cast ke text supaya bisa dipindai ke string.
// Urutan di sini HARUS sama dengan urutan Scan di scanUser.
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

	// NIM kosong dikirim sebagai NULL agar index unik parsial tidak bentrok
	// antar admin yang tidak punya NIM.
	var nim any
	if u.NIM != "" {
		nim = u.NIM
	}

	row := s.pool.QueryRow(ctx, `
		INSERT INTO users (name, email, password_hash, role, nim, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+userColumns,
		u.Name, u.Email, u.PasswordHash, string(role), nim, string(status),
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

// FindAuthState dipakai RequireAuth pada setiap request terproteksi:
// query ringan tanpa password_hash.
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

// mapPostgresError menerjemahkan error constraint PostgreSQL menjadi error domain.
func mapPostgresError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return err
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case "23505": // unique_violation
		switch pgErr.ConstraintName {
		case "users_nim_key":
			return ErrNIMTaken
		case "users_email_lower_key":
			return ErrEmailTaken
		}
		return err

	case "23514": // check_violation (trigger NIM menyebut nama constraint)
		switch pgErr.ConstraintName {
		case "users_nim_format", "users_mahasiswa_nim_required", "users_nim_prodi":
			return ErrInvalidNIM
		}
		return err

	case "22P02": // invalid_text_representation, mis. id bukan UUID
		return ErrNotFound
	}

	return err
}