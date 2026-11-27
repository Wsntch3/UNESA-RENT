package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound        = errors.New("data tidak ditemukan")
	ErrUserHasActivity = errors.New("pengguna masih memiliki aktivitas")
	ErrUserVerified    = errors.New("pengguna sudah terverifikasi")
	ErrInvalidReport   = errors.New("laporan tidak valid")
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func nullable(raw string) any {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return strings.TrimSpace(raw)
}

func (s *PostgresStore) CreateReport(ctx context.Context, reporterID string, in ReportInput) (Report, error) {
	var report Report
	var rentalID, listingID *string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO reports (reporter_id, rental_id, listing_id, type, subject, body)
		VALUES ($1::uuid, nullif($2, '')::uuid, nullif($3, '')::uuid, $4, $5, $6)
		RETURNING id::text, reporter_id::text,
		          coalesce(rental_id::text, ''), coalesce(listing_id::text, ''),
		          type, subject, body, status, created_at`,
		reporterID, nullable(in.RentalID), nullable(in.ListingID), in.Type, in.Subject, in.Body,
	).Scan(&report.ID, &report.ReporterID, &rentalID, &listingID,
		&report.Type, &report.Subject, &report.Body, &report.Status, &report.CreatedAt)
	if err != nil {
		return Report{}, mapError(err)
	}
	if rentalID != nil && *rentalID != "" {
		report.RentalID = rentalID
	}
	if listingID != nil && *listingID != "" {
		report.ListingID = listingID
	}
	report.ReporterName, _ = s.userName(ctx, reporterID)
	return report, nil
}

func (s *PostgresStore) Reports(ctx context.Context, f ReportFilter) ([]Report, int, error) {
	where := []string{"TRUE"}
	args := []any{}
	if status := strings.TrimSpace(f.Status); status != "" {
		args = append(args, status)
		where = append(where, fmt.Sprintf("r.status = $%d", len(args)))
	}
	if typ := strings.TrimSpace(f.Type); typ != "" {
		args = append(args, typ)
		where = append(where, fmt.Sprintf("r.type = $%d", len(args)))
	}
	clause := strings.Join(where, " AND ")
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM reports r WHERE `+clause, args...).Scan(&total); err != nil {
		return nil, 0, mapError(err)
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := s.pool.Query(ctx, `
		SELECT r.id::text, r.reporter_id::text, u.name, coalesce(r.rental_id::text, ''),
		       coalesce(r.listing_id::text, ''), r.type, r.subject, r.body, r.status,
		       r.resolution_note, r.resolved_at, r.created_at
		  FROM reports r JOIN users u ON u.id = r.reporter_id
		 WHERE `+clause+`
		 ORDER BY r.created_at DESC, r.id DESC
		 LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, mapError(err)
	}
	defer rows.Close()
	list := make([]Report, 0)
	for rows.Next() {
		var item Report
		var rentalID, listingID string
		if err := rows.Scan(&item.ID, &item.ReporterID, &item.ReporterName, &rentalID, &listingID,
			&item.Type, &item.Subject, &item.Body, &item.Status, &item.ResolutionNote,
			&item.ResolvedAt, &item.CreatedAt); err != nil {
			return nil, 0, mapError(err)
		}
		if rentalID != "" {
			item.RentalID = &rentalID
		}
		if listingID != "" {
			item.ListingID = &listingID
		}
		list = append(list, item)
	}
	return list, total, mapError(rows.Err())
}

func (s *PostgresStore) ReportByID(ctx context.Context, id string) (Report, error) {
	var item Report
	var rentalID, listingID string
	err := s.pool.QueryRow(ctx, `
		SELECT r.id::text, r.reporter_id::text, u.name, coalesce(r.rental_id::text, ''),
		       coalesce(r.listing_id::text, ''), r.type, r.subject, r.body, r.status,
		       r.resolution_note, r.resolved_at, r.created_at
		  FROM reports r JOIN users u ON u.id = r.reporter_id
		 WHERE r.id = $1::uuid`, id).Scan(
		&item.ID, &item.ReporterID, &item.ReporterName, &rentalID, &listingID,
		&item.Type, &item.Subject, &item.Body, &item.Status, &item.ResolutionNote,
		&item.ResolvedAt, &item.CreatedAt)
	if err != nil {
		return Report{}, mapError(err)
	}
	if rentalID != "" {
		item.RentalID = &rentalID
	}
	if listingID != "" {
		item.ListingID = &listingID
	}
	return item, nil
}

func (s *PostgresStore) ResolveReport(ctx context.Context, id, adminID, status, note string) (Report, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE reports
		   SET status = $2, assigned_to = $3::uuid, resolution_note = $4, resolved_at = now(), updated_at = now()
		 WHERE id = $1::uuid AND status IN ('open', 'in_review')`, id, status, adminID, note)
	if err != nil {
		return Report{}, mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return Report{}, ErrNotFound
	}
	return s.ReportByID(ctx, id)
}

func (s *PostgresStore) Notify(ctx context.Context, userID, title, body string, data map[string]any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		VALUES ($1::uuid, 'report_status', $2, $3, $4::jsonb)`, userID, title, body, raw)
	return mapError(err)
}

func (s *PostgresStore) userName(ctx context.Context, id string) (string, error) {
	var name string
	err := s.pool.QueryRow(ctx, `SELECT name FROM users WHERE id = $1::uuid`, id).Scan(&name)
	return name, mapError(err)
}

func (s *PostgresStore) AdminUsers(ctx context.Context, f UserFilter) ([]AdminUser, int, error) {
	where, args := userFilter(f)
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE `+strings.Join(where, " AND "), args...).Scan(&total); err != nil {
		return nil, 0, mapError(err)
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := s.pool.Query(ctx, `
		SELECT u.id::text, u.name, u.email, u.role, u.status, coalesce(u.nim, ''),
		       coalesce(is_ktm_verified(u.id), false), u.created_at, u.last_login_at
		  FROM users u
		 WHERE `+strings.Join(where, " AND ")+`
		 ORDER BY u.created_at DESC
		 LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, mapError(err)
	}
	defer rows.Close()
	list := make([]AdminUser, 0)
	for rows.Next() {
		var item AdminUser
		if err := rows.Scan(&item.ID, &item.Name, &item.Email, &item.Role, &item.Status, &item.NIM,
			&item.IsVerified, &item.CreatedAt, &item.LastLoginAt); err != nil {
			return nil, 0, mapError(err)
		}
		list = append(list, item)
	}
	return list, total, mapError(rows.Err())
}

func (s *PostgresStore) AdminUserByID(ctx context.Context, id string) (AdminUser, error) {
	var item AdminUser
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, name, email, role, status, coalesce(nim, ''),
		       coalesce(is_ktm_verified(id), false), created_at, last_login_at
		  FROM users WHERE id = $1::uuid`, id).Scan(
		&item.ID, &item.Name, &item.Email, &item.Role, &item.Status, &item.NIM,
		&item.IsVerified, &item.CreatedAt, &item.LastLoginAt)
	return item, mapError(err)
}

func (s *PostgresStore) ExportUsers(ctx context.Context, f UserFilter) ([]AdminUser, error) {
	where, args := userFilter(f)
	rows, err := s.pool.Query(ctx, `
		SELECT u.id::text, u.name, u.email, u.role, u.status, coalesce(u.nim, ''),
		       coalesce(is_ktm_verified(u.id), false), u.created_at, u.last_login_at
		  FROM users u
		 WHERE `+strings.Join(where, " AND ")+`
		 ORDER BY u.created_at DESC`, args...)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	list := make([]AdminUser, 0)
	for rows.Next() {
		var item AdminUser
		if err := rows.Scan(&item.ID, &item.Name, &item.Email, &item.Role, &item.Status, &item.NIM,
			&item.IsVerified, &item.CreatedAt, &item.LastLoginAt); err != nil {
			return nil, mapError(err)
		}
		list = append(list, item)
	}
	return list, mapError(rows.Err())
}

func userFilter(f UserFilter) ([]string, []any) {
	where := []string{"TRUE"}
	args := []any{}
	if q := strings.TrimSpace(f.Query); q != "" {
		args = append(args, "%"+q+"%")
		where = append(where, fmt.Sprintf("(u.name ILIKE $%d OR u.email ILIKE $%d OR u.nim ILIKE $%d)", len(args), len(args), len(args)))
	}
	if role := strings.TrimSpace(f.Role); role != "" {
		args = append(args, role)
		where = append(where, fmt.Sprintf("u.role = $%d", len(args)))
	}
	if status := strings.TrimSpace(f.Status); status != "" {
		args = append(args, status)
		where = append(where, fmt.Sprintf("u.status = $%d", len(args)))
	}
	if v := strings.ToLower(strings.TrimSpace(f.Verified)); v == "true" || v == "false" {
		where = append(where, fmt.Sprintf("coalesce(is_ktm_verified(u.id), false) = %t", v == "true"))
	}
	return where, args
}

func (s *PostgresStore) SetUserStatus(ctx context.Context, id, status string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE users SET status = $2, token_version = token_version + 1, updated_at = now()
		 WHERE id = $1::uuid`, id, status)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) DeleteUnverifiedUser(ctx context.Context, id string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var verified bool
	var role string
	if err := tx.QueryRow(ctx, `
		SELECT coalesce(is_ktm_verified(id), false), role FROM users WHERE id = $1::uuid FOR UPDATE`, id).
		Scan(&verified, &role); err != nil {
		return mapError(err)
	}
	if verified || role != "mahasiswa" {
		return ErrUserVerified
	}
	var count int
	if err := tx.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM listings WHERE owner_id = $1::uuid) +
			(SELECT count(*) FROM rentals WHERE renter_id = $1::uuid OR owner_id = $1::uuid) +
			(SELECT count(*) FROM rental_messages WHERE sender_id = $1::uuid) +
			(SELECT count(*) FROM reports WHERE reporter_id = $1::uuid)`, id).Scan(&count); err != nil {
		return mapError(err)
	}
	if count > 0 {
		return ErrUserHasActivity
	}
	if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1::uuid`, id); err != nil {
		return mapError(err)
	}
	return tx.Commit(ctx)
}

func (s *PostgresStore) LogAction(ctx context.Context, actorID, action, targetType, targetID, reason string, metadata map[string]any) error {
	raw, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO moderation_actions (actor_id, action, target_type, target_id, reason, metadata)
		VALUES ($1::uuid, $2, $3, $4::uuid, $5, $6::jsonb)`,
		actorID, action, targetType, targetID, reason, raw)
	return mapError(err)
}

func (s *PostgresStore) Audit(ctx context.Context, f AuditFilter) ([]AuditEntry, int, error) {
	conditions, args := auditFilter(f)
	render := func(alias, actorColumn string) string {
		replacer := strings.NewReplacer("{alias}", alias, "{actor}", actorColumn)
		parts := make([]string, 0, len(conditions))
		for _, condition := range conditions {
			parts = append(parts, replacer.Replace(condition))
		}
		return strings.Join(parts, " AND ")
	}
	union := `
		SELECT m.id::text, m.actor_id::text, coalesce(u.name, '') AS actor_name, m.action, m.target_type,
		       m.target_id::text, coalesce(m.reason, '') AS reason, ''::text AS ip_address, m.created_at
		  FROM moderation_actions m LEFT JOIN users u ON u.id = m.actor_id
		 WHERE ` + render("m.", "actor_id") + `
		UNION ALL
		SELECT c.id::text, c.admin_id::text, coalesce(u.name, '') AS actor_name, c.action, 'rental' AS target_type,
		       c.rental_id::text, c.reason, coalesce(c.ip_address, '') AS ip_address, c.created_at
		  FROM chat_access_log c LEFT JOIN users u ON u.id = c.admin_id
		 WHERE ` + render("c.", "admin_id")
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM (`+union+`) a`, args...).Scan(&total); err != nil {
		return nil, 0, mapError(err)
	}
	limitArg, offsetArg := len(args)+1, len(args)+2
	rows, err := s.pool.Query(ctx, `SELECT id, actor_id, actor_name, action, target_type, target_id, reason, ip_address, created_at FROM (`+
		union+`) a ORDER BY created_at DESC, id DESC LIMIT $`+fmt.Sprint(limitArg)+` OFFSET $`+fmt.Sprint(offsetArg)+``,
		append(args, f.Limit, f.Offset)...)
	if err != nil {
		return nil, 0, mapError(err)
	}
	defer rows.Close()
	list := make([]AuditEntry, 0)
	for rows.Next() {
		var item AuditEntry
		if err := rows.Scan(&item.ID, &item.ActorID, &item.ActorName, &item.Action, &item.TargetType,
			&item.TargetID, &item.Reason, &item.IPAddress, &item.CreatedAt); err != nil {
			return nil, 0, mapError(err)
		}
		list = append(list, item)
	}
	return list, total, mapError(rows.Err())
}

func auditFilter(f AuditFilter) ([]string, []any) {
	conditions := []string{"TRUE"}
	args := []any{}
	if actor := strings.TrimSpace(f.ActorID); actor != "" {
		args = append(args, actor)
		conditions = append(conditions, fmt.Sprintf("{alias}{actor} = $%d::uuid", len(args)))
	}
	if action := strings.TrimSpace(f.Action); action != "" {
		args = append(args, action)
		conditions = append(conditions, fmt.Sprintf("{alias}action = $%d", len(args)))
	}
	if from := strings.TrimSpace(f.From); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			args = append(args, t)
			conditions = append(conditions, fmt.Sprintf("{alias}created_at >= $%d", len(args)))
		}
	}
	if to := strings.TrimSpace(f.To); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			args = append(args, t.AddDate(0, 0, 1))
			conditions = append(conditions, fmt.Sprintf("{alias}created_at < $%d", len(args)))
		}
	}
	return conditions, args
}

func (s *PostgresStore) Stats(ctx context.Context) (Stats, error) {
	var out Stats
	err := s.pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM users),
			(SELECT count(*) FROM users u WHERE coalesce(is_ktm_verified(u.id), false)),
			(SELECT count(*) FROM listings WHERE is_active = true),
			(SELECT count(*) FROM reports WHERE status IN ('open', 'in_review')),
			(SELECT count(*) FROM rental_documents),
			(SELECT coalesce(sum(file_size_bytes), 0) FROM rental_documents)
	`).Scan(&out.TotalUsers, &out.VerifiedUsers, &out.ActiveListings, &out.OpenReports,
		&out.TotalDocuments, &out.StorageBytes)
	if err != nil {
		return Stats{}, mapError(err)
	}
	out.RentalsByStatus = make(map[string]int)
	rows, err := s.pool.Query(ctx, `SELECT status, count(*) FROM rentals GROUP BY status`)
	if err != nil {
		return Stats{}, mapError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return Stats{}, mapError(err)
		}
		out.RentalsByStatus[status] = count
	}
	return out, mapError(rows.Err())
}
