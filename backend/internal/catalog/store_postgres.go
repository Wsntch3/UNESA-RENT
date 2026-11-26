package catalog

import (
	"context"
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

// Urutan kolom HARUS sama dengan urutan Scan di scanListing.
const listingColumns = `
	l.id::text,
	l.owner_id::text,
	u.name,
	l.category_id::text,
	c.slug,
	c.name,
	l.campus_location_id::text,
	cl.name,
	l.title,
	l.description,
	l.price_per_day,
	l.condition_percent,
	l.is_active,
	l.takedown_reason,
	l.takedown_at,
	is_ktm_verified(l.owner_id),
	l.created_at,
	l.updated_at,
	COALESCE(p.id::text, ''),
	COALESCE(p.position, 0)`

const listingFrom = `
	FROM listings l
	JOIN users u ON u.id = l.owner_id
	JOIN categories c ON c.id = l.category_id
	JOIN campus_locations cl ON cl.id = l.campus_location_id
	LEFT JOIN LATERAL (
		SELECT lp.id, lp.position
		  FROM listing_photos lp
		 WHERE lp.listing_id = l.id
		 ORDER BY lp.position
		 LIMIT 1
	) p ON true`

const publicWhere = `
	l.is_active
	AND is_ktm_verified(l.owner_id)
	AND ($1 = '' OR l.title ILIKE '%' || $1 || '%')
	AND ($2 = '' OR c.slug = $2)
	AND ($3 = '' OR l.campus_location_id = NULLIF($3, '')::uuid)
	AND ($4 <= 0 OR l.price_per_day >= $4)
	AND ($5 <= 0 OR l.price_per_day <= $5)`

func (s *PostgresStore) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id::text, slug, name FROM categories WHERE is_active ORDER BY name`)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	list := make([]Category, 0)
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Slug, &c.Name); err != nil {
			return nil, mapPostgresError(err)
		}
		list = append(list, c)
	}
	return list, mapPostgresError(rows.Err())
}

func (s *PostgresStore) ListLocations(ctx context.Context) ([]Location, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id::text, name FROM campus_locations WHERE is_active ORDER BY name`)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	list := make([]Location, 0)
	for rows.Next() {
		var l Location
		if err := rows.Scan(&l.ID, &l.Name); err != nil {
			return nil, mapPostgresError(err)
		}
		list = append(list, l)
	}
	return list, mapPostgresError(rows.Err())
}

func (s *PostgresStore) IsVerifiedOwner(ctx context.Context, userID string) (bool, error) {
	var verified bool
	err := s.pool.QueryRow(ctx, `SELECT is_ktm_verified($1::uuid)`, strings.TrimSpace(userID)).Scan(&verified)
	if err != nil {
		return false, mapPostgresError(err)
	}
	return verified, nil
}

func (s *PostgresStore) Create(ctx context.Context, ownerID string, in ListingInput) (Listing, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO listings (owner_id, category_id, campus_location_id, title, description, price_per_day, condition_percent)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7)
		RETURNING id::text`,
		strings.TrimSpace(ownerID), in.CategoryID, in.CampusLocationID,
		in.Title, in.Description, in.PricePerDay, in.ConditionPercent,
	).Scan(&id)
	if err != nil {
		return Listing{}, mapPostgresError(err)
	}
	return s.Get(ctx, id)
}

func (s *PostgresStore) Update(ctx context.Context, id string, in ListingInput) (Listing, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE listings
		   SET category_id = $2::uuid,
		       campus_location_id = $3::uuid,
		       title = $4,
		       description = $5,
		       price_per_day = $6,
		       condition_percent = $7
		 WHERE id = $1::uuid`,
		strings.TrimSpace(id), in.CategoryID, in.CampusLocationID,
		in.Title, in.Description, in.PricePerDay, in.ConditionPercent,
	)
	if err != nil {
		return Listing{}, mapPostgresError(err)
	}
	if tag.RowsAffected() == 0 {
		return Listing{}, ErrNotFound
	}
	return s.Get(ctx, id)
}

func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM listings WHERE id = $1::uuid`, strings.TrimSpace(id))
	if err != nil {
		return mapPostgresError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) Get(ctx context.Context, id string) (Listing, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+listingColumns+listingFrom+` WHERE l.id = $1::uuid`,
		strings.TrimSpace(id),
	)
	listing, err := scanListing(row)
	if err != nil {
		return Listing{}, mapPostgresError(err)
	}

	photos, err := s.ListPhotos(ctx, listing.ID)
	if err != nil {
		return Listing{}, err
	}
	listing.Photos = photos
	return listing, nil
}

func (s *PostgresStore) Search(ctx context.Context, f Filter) ([]Listing, int, error) {
	var total int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) `+listingFrom+` WHERE `+publicWhere,
		f.Query, f.CategorySlug, f.LocationID, f.MinPrice, f.MaxPrice,
	).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	rows, err := s.pool.Query(ctx,
		`SELECT `+listingColumns+listingFrom+` WHERE `+publicWhere+`
		 ORDER BY `+orderClause(f.Sort)+`
		 LIMIT $6 OFFSET $7`,
		f.Query, f.CategorySlug, f.LocationID, f.MinPrice, f.MaxPrice, f.Limit, f.Offset,
	)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	list, err := collectListings(rows)
	return list, total, err
}

func (s *PostgresStore) ListByOwner(ctx context.Context, ownerID string, limit, offset int) ([]Listing, int, error) {
	const where = ` l.owner_id = $1::uuid`

	var total int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) `+listingFrom+` WHERE `+where,
		strings.TrimSpace(ownerID),
	).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	rows, err := s.pool.Query(ctx,
		`SELECT `+listingColumns+listingFrom+` WHERE `+where+`
		 ORDER BY l.created_at DESC
		 LIMIT $2 OFFSET $3`,
		strings.TrimSpace(ownerID), limit, offset,
	)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	list, err := collectListings(rows)
	return list, total, err
}

func (s *PostgresStore) AddPhoto(ctx context.Context, listingID, storageKey, mimeType string, size int64) (Photo, error) {
	var p Photo
	err := s.pool.QueryRow(ctx, `
		INSERT INTO listing_photos (listing_id, storage_key, mime_type, file_size_bytes, position)
		SELECT $1::uuid, $2, $3, $4, slot
		  FROM generate_series(1, 3) AS slot
		 WHERE NOT EXISTS (
		     SELECT 1 FROM listing_photos
		      WHERE listing_id = $1::uuid AND position = slot
		 )
		 ORDER BY slot
		 LIMIT 1
		RETURNING id::text, position, storage_key, mime_type, file_size_bytes`,
		strings.TrimSpace(listingID), storageKey, mimeType, size,
	).Scan(&p.ID, &p.Position, &p.StorageKey, &p.MimeType, &p.FileSizeBytes)
	if err != nil {
		return Photo{}, mapPostgresError(err)
	}
	return p, nil
}

func (s *PostgresStore) DeletePhoto(ctx context.Context, listingID, photoID string) (Photo, error) {
	var p Photo
	err := s.pool.QueryRow(ctx, `
		DELETE FROM listing_photos
		 WHERE id = $1::uuid AND listing_id = $2::uuid
		RETURNING id::text, position, storage_key, mime_type, file_size_bytes`,
		strings.TrimSpace(photoID), strings.TrimSpace(listingID),
	).Scan(&p.ID, &p.Position, &p.StorageKey, &p.MimeType, &p.FileSizeBytes)
	if err != nil {
		return Photo{}, mapPostgresError(err)
	}
	return p, nil
}

func (s *PostgresStore) GetPhoto(ctx context.Context, listingID, photoID string) (Photo, error) {
	var p Photo
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, position, storage_key, mime_type, file_size_bytes
		  FROM listing_photos
		 WHERE id = $1::uuid AND listing_id = $2::uuid`,
		strings.TrimSpace(photoID), strings.TrimSpace(listingID),
	).Scan(&p.ID, &p.Position, &p.StorageKey, &p.MimeType, &p.FileSizeBytes)
	if err != nil {
		return Photo{}, mapPostgresError(err)
	}
	return p, nil
}

func (s *PostgresStore) CountPhotos(ctx context.Context, listingID string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM listing_photos WHERE listing_id = $1::uuid`,
		strings.TrimSpace(listingID),
	).Scan(&n)
	if err != nil {
		return 0, mapPostgresError(err)
	}
	return n, nil
}

func (s *PostgresStore) ListPhotos(ctx context.Context, listingID string) ([]Photo, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, position, storage_key, mime_type, file_size_bytes
		  FROM listing_photos
		 WHERE listing_id = $1::uuid
		 ORDER BY position`,
		strings.TrimSpace(listingID),
	)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	list := make([]Photo, 0)
	for rows.Next() {
		var p Photo
		if err := rows.Scan(&p.ID, &p.Position, &p.StorageKey, &p.MimeType, &p.FileSizeBytes); err != nil {
			return nil, mapPostgresError(err)
		}
		list = append(list, p)
	}
	return list, mapPostgresError(rows.Err())
}

func (s *PostgresStore) AddBlock(ctx context.Context, listingID string, start, end time.Time, reason *string) (Block, error) {
	var b Block
	err := s.pool.QueryRow(ctx, `
		INSERT INTO listing_blocks (listing_id, period, reason)
		VALUES ($1::uuid, daterange($2::date, $3::date, '[]'), $4)
		RETURNING id::text, lower(period)::text, (upper(period) - 1)::text, reason, created_at`,
		strings.TrimSpace(listingID), start.Format(DateFormat), end.Format(DateFormat), reason,
	).Scan(&b.ID, &b.StartDate, &b.EndDate, &b.Reason, &b.CreatedAt)
	if err != nil {
		return Block{}, mapPostgresError(err)
	}
	return b, nil
}

func (s *PostgresStore) ListBlocks(ctx context.Context, listingID string) ([]Block, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, lower(period)::text, (upper(period) - 1)::text, reason, created_at
		  FROM listing_blocks
		 WHERE listing_id = $1::uuid
		 ORDER BY lower(period)`,
		strings.TrimSpace(listingID),
	)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	list := make([]Block, 0)
	for rows.Next() {
		var b Block
		if err := rows.Scan(&b.ID, &b.StartDate, &b.EndDate, &b.Reason, &b.CreatedAt); err != nil {
			return nil, mapPostgresError(err)
		}
		list = append(list, b)
	}
	return list, mapPostgresError(rows.Err())
}

func (s *PostgresStore) DeleteBlock(ctx context.Context, listingID, blockID string) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM listing_blocks WHERE id = $1::uuid AND listing_id = $2::uuid`,
		strings.TrimSpace(blockID), strings.TrimSpace(listingID),
	)
	if err != nil {
		return mapPostgresError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) AdminList(ctx context.Context, status, query string, limit, offset int) ([]Listing, int, error) {
	const where = `
		($1 = '' OR l.is_active = ($1 = 'active'))
		AND ($2 = '' OR l.title ILIKE '%' || $2 || '%')`

	var total int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) `+listingFrom+` WHERE `+where,
		status, query,
	).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	rows, err := s.pool.Query(ctx,
		`SELECT `+listingColumns+listingFrom+` WHERE `+where+`
		 ORDER BY l.created_at DESC
		 LIMIT $3 OFFSET $4`,
		status, query, limit, offset,
	)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	list, err := collectListings(rows)
	return list, total, err
}

func (s *PostgresStore) Takedown(ctx context.Context, id, adminID, reason string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE listings
		   SET is_active = false,
		       takedown_by = $2::uuid,
		       takedown_reason = $3,
		       takedown_at = now()
		 WHERE id = $1::uuid AND is_active`,
		strings.TrimSpace(id), strings.TrimSpace(adminID), strings.TrimSpace(reason),
	)
	if err != nil {
		return mapPostgresError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInvalidTransition
	}
	return nil
}

func (s *PostgresStore) Restore(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE listings
		   SET is_active = true,
		       takedown_by = NULL,
		       takedown_reason = NULL,
		       takedown_at = NULL
		 WHERE id = $1::uuid AND NOT is_active`,
		strings.TrimSpace(id),
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
		VALUES ($1::uuid, $2, 'listing', $3::uuid, $4)`,
		strings.TrimSpace(actorID), action, strings.TrimSpace(targetID), reasonArg,
	)
	return err
}

func collectListings(rows pgx.Rows) ([]Listing, error) {
	list := make([]Listing, 0)
	for rows.Next() {
		listing, err := scanListing(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		list = append(list, listing)
	}
	return list, mapPostgresError(rows.Err())
}

func scanListing(row pgx.Row) (Listing, error) {
	var (
		listing  Listing
		coverID  string
		coverPos int
	)

	err := row.Scan(
		&listing.ID,
		&listing.Owner.ID,
		&listing.Owner.Name,
		&listing.Category.ID,
		&listing.Category.Slug,
		&listing.Category.Name,
		&listing.Location.ID,
		&listing.Location.Name,
		&listing.Title,
		&listing.Description,
		&listing.PricePerDay,
		&listing.ConditionPercent,
		&listing.IsActive,
		&listing.TakedownReason,
		&listing.TakedownAt,
		&listing.OwnerVerified,
		&listing.CreatedAt,
		&listing.UpdatedAt,
		&coverID,
		&coverPos,
	)
	if err != nil {
		return Listing{}, err
	}
	if coverID != "" {
		listing.CoverPhoto = &Photo{ID: coverID, Position: coverPos}
	}
	return listing, nil
}

func orderClause(sort string) string {
	switch sort {
	case "price_asc":
		return "l.price_per_day ASC, l.created_at DESC"
	case "price_desc":
		return "l.price_per_day DESC, l.created_at DESC"
	default:
		return "l.created_at DESC"
	}
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
	case "23505": // unique_violation
		if strings.HasPrefix(pgErr.ConstraintName, "listing_photos_") {
			return ErrPhotoLimit
		}
	case "23503": // foreign_key_violation
		if pgErr.ConstraintName == "rentals_listing_id_fkey" {
			return ErrListingInUse
		}
		return ErrInvalidReference
	case "23P01": // exclusion_violation: blokir bertabrakan dengan blokir/rental lain
		return ErrBlockOverlap
	case "23514": // check_violation atau RAISE dari trigger
		switch {
		case strings.Contains(pgErr.Message, "penyedia harus"):
			return ErrNotVerified
		case strings.Contains(pgErr.Message, "periode blokir"):
			return ErrBlockOverlap
		}
	case "22P02": // invalid_text_representation, mis. id bukan UUID
		return ErrNotFound
	}

	return err
}
