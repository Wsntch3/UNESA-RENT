package rental

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

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

func (s *PostgresStore) ListMine(ctx context.Context, userID, status string, limit, offset int) ([]Rental, int, error) {
	const where = `
		(r.renter_id = $1::uuid OR r.owner_id = $1::uuid)
		AND ($2 = '' OR r.status = $2)`

	var total int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM rentals r WHERE `+where,
		strings.TrimSpace(userID), status,
	).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	rows, err := s.pool.Query(ctx,
		`SELECT `+rentalColumns+rentalFrom+` WHERE `+where+`
		 ORDER BY r.created_at DESC
		 LIMIT $3 OFFSET $4`,
		strings.TrimSpace(userID), status, limit, offset,
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

func (s *PostgresStore) SaveMessage(ctx context.Context, rentalID, senderID, body string) (Message, error) {
	var msg Message
	err := s.pool.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO rental_messages (rental_id, sender_id, body)
			VALUES ($1::uuid, $2::uuid, $3)
			RETURNING id, rental_id, sender_id, body, created_at
		)
		SELECT i.id::text, i.rental_id::text, i.sender_id::text, u.name, i.body, i.created_at
		  FROM inserted i
		  JOIN users u ON u.id = i.sender_id`,
		strings.TrimSpace(rentalID), strings.TrimSpace(senderID), body,
	).Scan(&msg.ID, &msg.RentalID, &msg.Sender.ID, &msg.Sender.Name, &msg.Body, &msg.CreatedAt)
	if err != nil {
		return Message{}, mapPostgresError(err)
	}
	return msg, nil
}

func (s *PostgresStore) Messages(ctx context.Context, rentalID string, after *time.Time, limit int) ([]Message, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT m.id::text, m.rental_id::text, m.sender_id::text, u.name, m.body, m.created_at
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
		if err := rows.Scan(&m.ID, &m.RentalID, &m.Sender.ID, &m.Sender.Name, &m.Body, &m.CreatedAt); err != nil {
			return nil, mapPostgresError(err)
		}
		list = append(list, m)
	}
	return list, mapPostgresError(rows.Err())
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
