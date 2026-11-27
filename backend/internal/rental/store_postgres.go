package rental

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func checksumHex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

type PostgresStore struct {
	pool  *pgxpool.Pool
	files Storage
}

func NewPostgresStore(pool *pgxpool.Pool, files Storage) *PostgresStore {
	return &PostgresStore{pool: pool, files: files}
}

const rentalColumns = `
	r.id::text,
	r.listing_id::text,
	l.title,
	COALESCE(cover.id::text, ''),
	r.renter_id::text,
	renter.name,
	r.owner_id::text,
	owner.name,
	r.start_date::text,
	r.end_date::text,
	r.total_days,
	r.total_price,
	r.price_per_day_snapshot,
	r.status,
	r.rejection_reason,
	r.approved_at,
	r.rejected_at,
	r.cancelled_at,
	r.completed_at,
	r.created_at,
	r.updated_at`

const rentalFrom = `
	FROM rentals r
	JOIN listings l ON l.id = r.listing_id
	JOIN users renter ON renter.id = r.renter_id
	JOIN users owner ON owner.id = r.owner_id
	LEFT JOIN LATERAL (
		SELECT lp.id
		  FROM listing_photos lp
		 WHERE lp.listing_id = l.id
		 ORDER BY lp.position
		 LIMIT 1
	) cover ON true`

func (s *PostgresStore) IsVerified(ctx context.Context, userID string) (bool, error) {
	var verified bool
	err := s.pool.QueryRow(ctx, `SELECT is_ktm_verified($1::uuid)`, strings.TrimSpace(userID)).Scan(&verified)
	if err != nil {
		return false, mapPostgresError(err)
	}
	return verified, nil
}

func (s *PostgresStore) ParticipantNIM(ctx context.Context, userID string) (string, error) {
	var nim string
	err := s.pool.QueryRow(ctx, `SELECT COALESCE(nim, '') FROM users WHERE id = $1::uuid`, strings.TrimSpace(userID)).Scan(&nim)
	if err != nil {
		return "", mapPostgresError(err)
	}
	return nim, nil
}

func (s *PostgresStore) DefaultLocation(ctx context.Context) string {
	var name string
	if err := s.pool.QueryRow(ctx, `SELECT name FROM campus_locations ORDER BY name LIMIT 1`).Scan(&name); err != nil {
		return "Lokasi FT UNESA"
	}
	return name
}

func (s *PostgresStore) ListingInfo(ctx context.Context, listingID string) (string, bool, error) {
	var (
		ownerID string
		active  bool
	)
	err := s.pool.QueryRow(ctx,
		`SELECT owner_id::text, is_active FROM listings WHERE id = $1::uuid`,
		strings.TrimSpace(listingID),
	).Scan(&ownerID, &active)
	if err != nil {
		return "", false, mapPostgresError(err)
	}
	return ownerID, active, nil
}

func (s *PostgresStore) Create(ctx context.Context, listingID, renterID string, start, end time.Time) (Rental, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO rentals (listing_id, renter_id, start_date, end_date)
		VALUES ($1::uuid, $2::uuid, $3::date, $4::date)
		RETURNING id::text`,
		strings.TrimSpace(listingID), strings.TrimSpace(renterID),
		start.Format(DateFormat), end.Format(DateFormat),
	).Scan(&id)
	if err != nil {
		return Rental{}, mapPostgresError(err)
	}
	return s.Get(ctx, id)
}

func (s *PostgresStore) Get(ctx context.Context, id string) (Rental, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+rentalColumns+rentalFrom+` WHERE r.id = $1::uuid`,
		strings.TrimSpace(id),
	)
	rental, err := scanRental(row)
	if err != nil {
		return Rental{}, mapPostgresError(err)
	}
	return rental, nil
}

func (s *PostgresStore) ListMine(ctx context.Context, userID, role, status, from, to, query string, limit, offset int) ([]Rental, int, error) {
	where := `(r.renter_id = $1::uuid OR r.owner_id = $1::uuid)
		AND ($2 = '' OR r.status = $2)
		AND ($3 = '' OR r.renter_id = $1::uuid)
		AND ($4 = '' OR r.owner_id = $1::uuid)
		AND ($5::date IS NULL OR r.created_at::date >= $5::date)
		AND ($6::date IS NULL OR r.created_at::date <= $6::date)
		AND ($7 = '' OR l.title ILIKE '%' || $7 || '%')`
	args := []any{strings.TrimSpace(userID), status, roleValue(role, "renter"), roleValue(role, "owner"), nullableDate(from), nullableDate(to), strings.TrimSpace(query)}

	var total int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM rentals r JOIN listings l ON l.id = r.listing_id WHERE `+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	args = append(args, limit, offset)
	rows, err := s.pool.Query(ctx,
		`SELECT `+rentalColumns+rentalFrom+` WHERE `+where+`
		 ORDER BY r.created_at DESC
		 LIMIT $8 OFFSET $9`, args...,
	)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	list := make([]Rental, 0)
	for rows.Next() {
		rental, err := scanRental(rows)
		if err != nil {
			return nil, 0, mapPostgresError(err)
		}
		list = append(list, rental)
	}
	return list, total, mapPostgresError(rows.Err())
}

func roleValue(role, wanted string) string {
	if strings.EqualFold(strings.TrimSpace(role), wanted) {
		return wanted
	}
	return ""
}

func nullableDate(raw string) any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	return raw
}

func (s *PostgresStore) UpdateStatus(ctx context.Context, id string, to Status, rejectionReason *string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE rentals
		   SET status = $2,
		       rejection_reason = $3
		 WHERE id = $1::uuid`,
		strings.TrimSpace(id), string(to), rejectionReason,
	)
	if err != nil {
		return mapPostgresError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) AddEvent(ctx context.Context, rentalID string, from *Status, to Status, actorID string, note *string) error {
	var fromArg any
	if from != nil {
		fromArg = string(*from)
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO rental_events (rental_id, from_status, to_status, actor_id, note)
		VALUES ($1::uuid, $2, $3, $4::uuid, $5)`,
		strings.TrimSpace(rentalID), fromArg, string(to), strings.TrimSpace(actorID), note,
	)
	return mapPostgresError(err)
}

func (s *PostgresStore) Events(ctx context.Context, rentalID string) ([]Event, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT e.id,
		       e.from_status,
		       e.to_status,
		       e.actor_id::text,
		       u.name,
		       e.note,
		       e.created_at
		  FROM rental_events e
		  LEFT JOIN users u ON u.id = e.actor_id
		 WHERE e.rental_id = $1::uuid
		 ORDER BY e.created_at, e.id`,
		strings.TrimSpace(rentalID),
	)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	list := make([]Event, 0)
	for rows.Next() {
		var (
			event   Event
			actorID *string
			name    *string
		)
		if err := rows.Scan(&event.ID, &event.FromStatus, &event.ToStatus,
			&actorID, &name, &event.Note, &event.CreatedAt); err != nil {
			return nil, mapPostgresError(err)
		}
		if actorID != nil && name != nil {
			event.Actor = &Party{ID: *actorID, Name: *name}
		}
		list = append(list, event)
	}
	return list, mapPostgresError(rows.Err())
}

func (s *PostgresStore) Availability(ctx context.Context, listingID string, from, to time.Time) ([]Unavailable, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT lower(period)::text,
		       (upper(period) - 1)::text,
		       'block'
		  FROM listing_blocks
		 WHERE listing_id = $1::uuid
		   AND period && daterange($2::date, $3::date, '[]')
		UNION ALL
		SELECT start_date::text,
		       end_date::text,
		       'rental'
		  FROM rentals
		 WHERE listing_id = $1::uuid
		   AND status = 'approved'
		   AND daterange(start_date, end_date, '[]') && daterange($2::date, $3::date, '[]')
		 ORDER BY 1`,
		strings.TrimSpace(listingID), from.Format(DateFormat), to.Format(DateFormat),
	)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	list := make([]Unavailable, 0)
	for rows.Next() {
		var u Unavailable
		if err := rows.Scan(&u.StartDate, &u.EndDate, &u.Source); err != nil {
			return nil, mapPostgresError(err)
		}
		list = append(list, u)
	}
	return list, mapPostgresError(rows.Err())
}

func (s *PostgresStore) CreateNotification(ctx context.Context, userID, kind, title, body string, data map[string]any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		VALUES ($1::uuid, $2, $3, $4, $5::jsonb)`,
		strings.TrimSpace(userID), kind, title, body, raw,
	)
	return mapPostgresError(err)
}

func (s *PostgresStore) Notifications(ctx context.Context, userID string, unreadOnly bool, limit, offset int) ([]Notification, int, int, error) {
	const where = `user_id = $1::uuid AND (NOT $2 OR read_at IS NULL)`

	var total, unread int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FILTER (WHERE TRUE), count(*) FILTER (WHERE read_at IS NULL)
		   FROM notifications WHERE user_id = $1::uuid`,
		strings.TrimSpace(userID),
	).Scan(&total, &unread); err != nil {
		return nil, 0, 0, mapPostgresError(err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id::text, type, title, body, data, read_at, created_at
		  FROM notifications
		 WHERE `+where+`
		 ORDER BY created_at DESC
		 LIMIT $3 OFFSET $4`,
		strings.TrimSpace(userID), unreadOnly, limit, offset,
	)
	if err != nil {
		return nil, 0, 0, mapPostgresError(err)
	}
	defer rows.Close()

	list := make([]Notification, 0)
	for rows.Next() {
		var (
			n   Notification
			raw []byte
		)
		if err := rows.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &raw, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, 0, 0, mapPostgresError(err)
		}
		if err := json.Unmarshal(raw, &n.Data); err != nil {
			return nil, 0, 0, err
		}
		list = append(list, n)
	}
	return list, total, unread, mapPostgresError(rows.Err())
}

func (s *PostgresStore) MarkNotificationRead(ctx context.Context, userID, id string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE notifications
		   SET read_at = now()
		 WHERE id = $1::uuid AND user_id = $2::uuid AND read_at IS NULL`,
		strings.TrimSpace(id), strings.TrimSpace(userID),
	)
	return mapPostgresError(err)
}

func (s *PostgresStore) SaveMessage(ctx context.Context, rentalID, senderID string, ciphertext, nonce []byte, version, length int, sha256 string) (Message, error) {
	var msg Message
	err := s.pool.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO rental_messages
				(rental_id, sender_id, body_ciphertext, body_nonce, key_version, body_length, body_sha256)
			VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7)
			RETURNING id, rental_id, sender_id, body_ciphertext, body_nonce, key_version, body_length, body_sha256, created_at
		)
		SELECT i.id::text, i.rental_id::text, i.sender_id::text, u.name,
		       i.body_ciphertext, i.body_nonce, i.key_version, i.body_length, i.body_sha256, i.created_at
		  FROM inserted i
		  JOIN users u ON u.id = i.sender_id`,
		strings.TrimSpace(rentalID), strings.TrimSpace(senderID),
		ciphertext, nonce, version, length, sha256,
	).Scan(&msg.ID, &msg.RentalID, &msg.Sender.ID, &msg.Sender.Name,
		&msg.Ciphertext, &msg.Nonce, &msg.KeyVersion, &msg.BodyLength, &msg.BodySHA256, &msg.CreatedAt)
	if err != nil {
		return Message{}, mapPostgresError(err)
	}
	return msg, nil
}

func (s *PostgresStore) Messages(ctx context.Context, rentalID string, after *time.Time, limit int) ([]Message, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT m.id::text, m.rental_id::text, m.sender_id::text, u.name,
		       m.body_ciphertext, m.body_nonce, m.key_version, m.body_length, m.body_sha256, m.created_at
		  FROM rental_messages m
		  JOIN users u ON u.id = m.sender_id
		 WHERE m.rental_id = $1::uuid
		   AND ($2::timestamptz IS NULL OR m.created_at > $2)
		 ORDER BY m.created_at, m.id
		 LIMIT $3`,
		strings.TrimSpace(rentalID), after, limit,
	)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	list := make([]Message, 0)
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.RentalID, &m.Sender.ID, &m.Sender.Name,
			&m.Ciphertext, &m.Nonce, &m.KeyVersion, &m.BodyLength, &m.BodySHA256, &m.CreatedAt); err != nil {
			return nil, mapPostgresError(err)
		}
		list = append(list, m)
	}
	return list, mapPostgresError(rows.Err())
}

func (s *PostgresStore) AdminMessages(ctx context.Context, rentalID string, limit, offset int) ([]AdminMessage, int, error) {
	var total int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM rental_messages WHERE rental_id = $1::uuid`, strings.TrimSpace(rentalID),
	).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT m.id::text, m.sender_id::text, u.name, m.body_length, m.body_sha256, m.created_at
		  FROM rental_messages m
		  JOIN users u ON u.id = m.sender_id
		 WHERE m.rental_id = $1::uuid
		 ORDER BY m.created_at, m.id
		 LIMIT $2 OFFSET $3`,
		strings.TrimSpace(rentalID), limit, offset,
	)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	list := make([]AdminMessage, 0)
	for rows.Next() {
		var m AdminMessage
		if err := rows.Scan(&m.ID, &m.Sender.ID, &m.Sender.Name, &m.BodyLength, &m.BodySHA256, &m.CreatedAt); err != nil {
			return nil, 0, mapPostgresError(err)
		}
		list = append(list, m)
	}
	return list, total, mapPostgresError(rows.Err())
}

func (s *PostgresStore) AdminMessageByID(ctx context.Context, rentalID, messageID string) (Message, error) {
	var msg Message
	err := s.pool.QueryRow(ctx, `
		SELECT m.id::text, m.rental_id::text, m.sender_id::text, u.name,
		       m.body_ciphertext, m.body_nonce, m.key_version, m.body_length, m.body_sha256, m.created_at
		  FROM rental_messages m
		  JOIN users u ON u.id = m.sender_id
		 WHERE m.rental_id = $1::uuid AND m.id = $2::uuid`,
		strings.TrimSpace(rentalID), strings.TrimSpace(messageID),
	).Scan(&msg.ID, &msg.RentalID, &msg.Sender.ID, &msg.Sender.Name,
		&msg.Ciphertext, &msg.Nonce, &msg.KeyVersion, &msg.BodyLength, &msg.BodySHA256, &msg.CreatedAt)
	if err != nil {
		return Message{}, mapPostgresError(err)
	}
	return msg, nil
}

func (s *PostgresStore) InsertChatAccessLog(ctx context.Context, adminID, rentalID, messageID, action, reason, ip, userAgent string) error {
	var messageArg any
	if strings.TrimSpace(messageID) != "" {
		messageArg = strings.TrimSpace(messageID)
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO chat_access_log (admin_id, rental_id, message_id, action, reason, ip_address, user_agent)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7)`,
		strings.TrimSpace(adminID), strings.TrimSpace(rentalID), messageArg, action, reason, nullableString(ip), nullableString(userAgent),
	)
	return mapPostgresError(err)
}

func nullableString(raw string) any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	return raw
}

func scanRental(row pgx.Row) (Rental, error) {
	var (
		rental Rental
		status string
		cover  string
	)
	err := row.Scan(
		&rental.ID,
		&rental.Listing.ID,
		&rental.Listing.Title,
		&cover,
		&rental.Renter.ID,
		&rental.Renter.Name,
		&rental.Owner.ID,
		&rental.Owner.Name,
		&rental.StartDate,
		&rental.EndDate,
		&rental.TotalDays,
		&rental.TotalPrice,
		&rental.PricePerDaySnapshot,
		&status,
		&rental.RejectionReason,
		&rental.ApprovedAt,
		&rental.RejectedAt,
		&rental.CancelledAt,
		&rental.CompletedAt,
		&rental.CreatedAt,
		&rental.UpdatedAt,
	)
	if err != nil {
		return Rental{}, err
	}
	rental.Status = Status(status)
	if cover != "" {
		rental.Listing.CoverPhotoURL = "/api/listings/" + rental.Listing.ID + "/photos/" + cover + "/file"
	}
	return rental, nil
}

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
	case "23P01":
		if strings.Contains(pgErr.Message, "diblokir") {
			return ErrDatesBlocked
		}
		return ErrDateUnavailable
	case "23503":
		return ErrNotFound
	case "23514":
		switch {
		case strings.Contains(pgErr.Message, "start_date tidak boleh"),
			strings.Contains(pgErr.Message, "end_date harus"),
			pgErr.ConstraintName == "rentals_max_duration":
			return ErrInvalidDates
		case strings.Contains(pgErr.Message, "penyewa harus"),
			strings.Contains(pgErr.Message, "penyedia"):
			return ErrNotVerified
		case strings.Contains(pgErr.Message, "diblokir penyedia"):
			return ErrDatesBlocked
		case strings.Contains(pgErr.Message, "rental_messages_body_length"):
			return ErrInvalidBody
		default:
			return ErrInvalidTransition
		}
	case "22P02":
		return ErrNotFound
	}
	return err
}

func (s *PostgresStore) AdminList(ctx context.Context, filter RentalFilter) ([]Rental, int, error) {
	where := `($1 = '' OR r.status = $1)
		AND ($2::date IS NULL OR r.created_at::date >= $2::date)
		AND ($3::date IS NULL OR r.created_at::date <= $3::date)
		AND ($4 = '' OR l.title ILIKE '%' || $4 || '%')
		AND (NOT $5 OR (r.status = 'approved' AND r.end_date < current_date))`
	args := []any{filter.Status, nullableDate(filter.From), nullableDate(filter.To), strings.TrimSpace(filter.Query), filter.Overdue}

	var total int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM rentals r JOIN listings l ON l.id = r.listing_id WHERE `+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	limit := clampLimit(filter.Limit, DefaultListLimit, MaxListLimit)
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)
	rows, err := s.pool.Query(ctx,
		`SELECT `+rentalColumns+rentalFrom+` WHERE `+where+`
		 ORDER BY r.created_at DESC
		 LIMIT $6 OFFSET $7`, args...,
	)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	list := make([]Rental, 0)
	for rows.Next() {
		rental, err := scanRental(rows)
		if err != nil {
			return nil, 0, mapPostgresError(err)
		}
		list = append(list, rental)
	}
	return list, total, mapPostgresError(rows.Err())
}

func (s *PostgresStore) AdminStats(ctx context.Context) (RentalStats, error) {
	stats := RentalStats{ByStatus: map[string]int{}}
	rows, err := s.pool.Query(ctx, `SELECT status, count(*) FROM rentals GROUP BY status`)
	if err != nil {
		return stats, mapPostgresError(err)
	}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			rows.Close()
			return stats, mapPostgresError(err)
		}
		stats.ByStatus[status] = count
		stats.Total += count
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return stats, mapPostgresError(err)
	}
	rows.Close()

	err = s.pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE status = 'approved' AND end_date < current_date),
		       count(*), COALESCE(sum(file_size_bytes), 0)
		  FROM rentals r
		  LEFT JOIN rental_documents d ON d.rental_id = r.id`,
	).Scan(&stats.Overdue, &stats.Documents, &stats.StorageByte)
	return stats, mapPostgresError(err)
}

func (s *PostgresStore) History(ctx context.Context, userID, role string, year int) ([]HistoryMonth, error) {
	roleWhere := `(r.renter_id = $1::uuid OR r.owner_id = $1::uuid)`
	if role == "renter" {
		roleWhere = `r.renter_id = $1::uuid`
	} else if role == "owner" {
		roleWhere = `r.owner_id = $1::uuid`
	}
	rows, err := s.pool.Query(ctx, `
		SELECT to_char(date_trunc('month', r.created_at), 'YYYY-MM') AS month,
		       count(*) AS total,
		       COALESCE(sum(r.total_price), 0) AS value,
		       count(*) FILTER (WHERE r.status = 'completed') AS completed,
		       count(*) FILTER (WHERE r.status = 'cancelled') AS cancelled
		  FROM rentals r
		 WHERE `+roleWhere+` AND extract(year from r.created_at) = $2
		 GROUP BY 1 ORDER BY 1`, strings.TrimSpace(userID), year,
	)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	list := make([]HistoryMonth, 0)
	for rows.Next() {
		var m HistoryMonth
		if err := rows.Scan(&m.Month, &m.Total, &m.Value, &m.Completed, &m.Cancelled); err != nil {
			return nil, mapPostgresError(err)
		}
		list = append(list, m)
	}
	return list, mapPostgresError(rows.Err())
}

func (s *PostgresStore) ExportRows(ctx context.Context, filter RentalFilter) ([]Rental, error) {
	filter.Limit = 10000
	filter.Offset = 0
	list, _, err := s.AdminList(ctx, filter)
	return list, err
}

func (s *PostgresStore) EnsureCOD(ctx context.Context, rentalID string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO rental_cod (rental_id, agreed_at) VALUES ($1::uuid, now()) ON CONFLICT (rental_id) DO NOTHING`,
		strings.TrimSpace(rentalID),
	)
	return mapPostgresError(err)
}

func (s *PostgresStore) GetCOD(ctx context.Context, rentalID string) (CODState, error) {
	var cod CODState
	err := s.pool.QueryRow(ctx, `
		SELECT agreed_at, handover_at, handover_confirmed_at, return_at, return_confirmed_at,
		       amount_due, amount_paid, payment_method, settled_at, notes
		  FROM rental_cod WHERE rental_id = $1::uuid`, strings.TrimSpace(rentalID),
	).Scan(&cod.AgreedAt, &cod.HandoverAt, &cod.HandoverConfirmedAt, &cod.ReturnAt,
		&cod.ReturnConfirmedAt, &cod.AmountDue, &cod.AmountPaid, &cod.PaymentMethod,
		&cod.SettledAt, &cod.Notes)
	if err != nil {
		return CODState{}, mapPostgresError(err)
	}
	return cod, nil
}

func (s *PostgresStore) CODAgreed(ctx context.Context, rentalID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE rental_cod SET agreed_at = coalesce(agreed_at, now()), updated_at = now()
		 WHERE rental_id = $1::uuid`, strings.TrimSpace(rentalID))
	return mapPostgresError(err)
}

func (s *PostgresStore) CODHandover(ctx context.Context, rentalID, actorID, note string, proof []byte, mimeType string) (Document, error) {
	var docID string
	err := s.pool.QueryRow(ctx, `
		UPDATE rental_cod
		   SET handover_at = now(), handover_by = $2::uuid, notes = nullif($3, ''), updated_at = now()
		 WHERE rental_id = $1::uuid AND handover_at IS NULL
		 RETURNING rental_id::text`, strings.TrimSpace(rentalID), strings.TrimSpace(actorID), note,
	).Scan(&docID)
	if err != nil {
		return Document{}, mapPostgresError(err)
	}
	return s.saveProof(ctx, rentalID, "handover_proof", actorID, note, proof, mimeType)
}

func (s *PostgresStore) CODHandoverConfirm(ctx context.Context, rentalID, actorID string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE rental_cod
		   SET handover_confirmed_at = now(), handover_confirmed_by = $2::uuid, updated_at = now()
		 WHERE rental_id = $1::uuid AND handover_at IS NOT NULL AND handover_confirmed_at IS NULL`,
		strings.TrimSpace(rentalID), strings.TrimSpace(actorID),
	)
	if err != nil {
		return mapPostgresError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInvalidCODState
	}
	return nil
}

func (s *PostgresStore) CODReturn(ctx context.Context, rentalID, actorID, note string, proof []byte, mimeType string) (Document, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		UPDATE rental_cod
		   SET return_at = now(), return_by = $2::uuid, notes = coalesce(nullif($3, ''), notes), updated_at = now()
		 WHERE rental_id = $1::uuid AND handover_confirmed_at IS NOT NULL AND return_at IS NULL
		 RETURNING rental_id::text`, strings.TrimSpace(rentalID), strings.TrimSpace(actorID), note,
	).Scan(&id)
	if err != nil {
		return Document{}, mapPostgresError(err)
	}
	return s.saveProof(ctx, rentalID, "return_proof", actorID, note, proof, mimeType)
}

func (s *PostgresStore) CODReturnConfirm(ctx context.Context, rentalID, actorID string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE rental_cod
		   SET return_confirmed_at = now(), return_confirmed_by = $2::uuid, updated_at = now()
		 WHERE rental_id = $1::uuid AND return_at IS NOT NULL AND return_confirmed_at IS NULL`,
		strings.TrimSpace(rentalID), strings.TrimSpace(actorID),
	)
	if err != nil {
		return mapPostgresError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInvalidCODState
	}
	return nil
}

func (s *PostgresStore) CODSettle(ctx context.Context, rentalID string, in CODSettleInput) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE rental_cod
		   SET amount_paid = $2, payment_method = $3,
		       notes = CASE WHEN nullif($4, '') IS NULL THEN notes ELSE $4 END,
		       settled_at = now(), updated_at = now()
		 WHERE rental_id = $1::uuid`,
		strings.TrimSpace(rentalID), in.AmountPaid, in.PaymentMethod, in.Notes,
	)
	if err != nil {
		return mapPostgresError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInvalidCODState
	}
	return nil
}

func (s *PostgresStore) SaveDocument(ctx context.Context, rentalID, docType, key, mimeType string, size int64, checksum, uploadedBy, visibility string, metadata map[string]any) (Document, error) {
	raw, err := json.Marshal(metadata)
	if err != nil {
		return Document{}, err
	}
	var uploader any
	if strings.TrimSpace(uploadedBy) != "" {
		uploader = strings.TrimSpace(uploadedBy)
	}
	var doc Document
	err = s.pool.QueryRow(ctx, `
		INSERT INTO rental_documents
			(rental_id, type, storage_key, mime_type, file_size_bytes, checksum_sha256, uploaded_by, visibility, metadata)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7::uuid, $8, $9::jsonb)
		RETURNING id::text, rental_id::text, type, mime_type, file_size_bytes, visibility, created_at`,
		strings.TrimSpace(rentalID), docType, key, mimeType, size, checksum, uploader, visibility, raw,
	).Scan(&doc.ID, &doc.RentalID, &doc.Type, &doc.MimeType, &doc.SizeBytes, &doc.Visibility, &doc.CreatedAt)
	if err != nil {
		return Document{}, mapPostgresError(err)
	}
	return doc, nil
}

func (s *PostgresStore) Documents(ctx context.Context, rentalID string) ([]Document, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT d.id::text, d.rental_id::text, d.type, d.mime_type, d.file_size_bytes,
		       d.visibility, d.created_at, u.id::text, u.name
		  FROM rental_documents d
		  LEFT JOIN users u ON u.id = d.uploaded_by
		 WHERE d.rental_id = $1::uuid
		 ORDER BY d.created_at DESC, d.id DESC`, strings.TrimSpace(rentalID),
	)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	list := make([]Document, 0)
	for rows.Next() {
		var doc Document
		var userID, userName *string
		if err := rows.Scan(&doc.ID, &doc.RentalID, &doc.Type, &doc.MimeType, &doc.SizeBytes,
			&doc.Visibility, &doc.CreatedAt, &userID, &userName); err != nil {
			return nil, mapPostgresError(err)
		}
		if userID != nil && userName != nil {
			doc.UploadedBy = &Party{ID: *userID, Name: *userName}
		}
		list = append(list, doc)
	}
	return list, mapPostgresError(rows.Err())
}

func (s *PostgresStore) DocumentByID(ctx context.Context, rentalID, docID string) (Document, error) {
	var doc Document
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, rental_id::text, type, mime_type, file_size_bytes, visibility, created_at
		  FROM rental_documents WHERE rental_id = $1::uuid AND id = $2::uuid`,
		strings.TrimSpace(rentalID), strings.TrimSpace(docID),
	).Scan(&doc.ID, &doc.RentalID, &doc.Type, &doc.MimeType, &doc.SizeBytes, &doc.Visibility, &doc.CreatedAt)
	if err != nil {
		return Document{}, mapPostgresError(err)
	}
	return doc, nil
}

func (s *PostgresStore) DocumentBytes(ctx context.Context, documentID string) ([]byte, error) {
	var key string
	if err := s.pool.QueryRow(ctx,
		`SELECT storage_key FROM rental_documents WHERE id = $1::uuid`, strings.TrimSpace(documentID),
	).Scan(&key); err != nil {
		return nil, mapPostgresError(err)
	}
	file, err := s.files.Open(ctx, key)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

func (s *PostgresStore) saveProof(ctx context.Context, rentalID, docType, actorID, note string, proof []byte, mimeType string) (Document, error) {
	if len(proof) == 0 {
		return Document{}, ErrMissingProof
	}
	ext := ".jpg"
	if mimeType == "image/png" {
		ext = ".png"
	}
	key, err := newDocumentKey(rentalID, ext)
	if err != nil {
		return Document{}, err
	}
	if err := s.files.Save(ctx, key, proof); err != nil {
		return Document{}, err
	}
	return s.SaveDocument(ctx, rentalID, docType, key, mimeType, int64(len(proof)), checksumHex(proof), actorID, "participants", map[string]any{"note": note})
}
