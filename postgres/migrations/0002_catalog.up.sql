-- 0002_catalog.up.sql
-- Katalog: kategori, lokasi serah-terima, listing, foto, dan tanggal blokir.

CREATE TABLE categories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug       VARCHAR(60) NOT NULL UNIQUE,
    name       VARCHAR(80) NOT NULL,
    is_active  BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT categories_slug_format CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    CONSTRAINT categories_name_length CHECK (char_length(btrim(name)) BETWEEN 2 AND 80)
);

CREATE TABLE campus_locations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(120) NOT NULL UNIQUE,
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT campus_locations_name_length CHECK (char_length(btrim(name)) BETWEEN 2 AND 120)
);

-- Harga pas integer tanpa diskon/cicilan (FR-02, BR-01).
-- is_active = false adalah mekanisme takedown (FR-06).
CREATE TABLE listings (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id           UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    category_id        UUID NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    campus_location_id UUID NOT NULL REFERENCES campus_locations(id) ON DELETE RESTRICT,
    title              VARCHAR(120) NOT NULL,
    description        TEXT NOT NULL DEFAULT '',
    price_per_day      INTEGER NOT NULL,
    condition_percent  SMALLINT NOT NULL,
    is_active          BOOLEAN NOT NULL DEFAULT true,
    takedown_reason    TEXT,
    takedown_by        UUID REFERENCES users(id) ON DELETE RESTRICT,
    takedown_at        TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT listings_title_length CHECK (char_length(btrim(title)) BETWEEN 3 AND 120),
    CONSTRAINT listings_description_length CHECK (char_length(description) <= 2000),
    CONSTRAINT listings_price_positive CHECK (price_per_day > 0),
    CONSTRAINT listings_condition_range CHECK (condition_percent BETWEEN 1 AND 100),
    CONSTRAINT listings_takedown_consistency CHECK (
        is_active
        OR (
            takedown_reason IS NOT NULL
            AND char_length(btrim(takedown_reason)) > 0
            AND takedown_by IS NOT NULL
            AND takedown_at IS NOT NULL
        )
    )
);

-- Maksimal 3 foto ditegakkan oleh position 1..3 + UNIQUE(listing_id, position).
CREATE TABLE listing_photos (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id      UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    storage_key     TEXT NOT NULL UNIQUE,
    mime_type       VARCHAR(80) NOT NULL DEFAULT 'image/jpeg',
    file_size_bytes BIGINT,
    position        SMALLINT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT listing_photos_position_range CHECK (position BETWEEN 1 AND 3),
    CONSTRAINT listing_photos_file_size_positive CHECK (file_size_bytes IS NULL OR file_size_bytes > 0),
    CONSTRAINT listing_photos_listing_position_key UNIQUE (listing_id, position)
);

-- Tanggal yang diblokir manual oleh penyedia (FR-03).
-- period disimpan sebagai DATERANGE inklusif, mis. [2026-12-01, 2026-12-05].
CREATE TABLE listing_blocks (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    period     DATERANGE NOT NULL,
    reason     TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT listing_blocks_period_not_empty CHECK (NOT isempty(period)),
    CONSTRAINT listing_blocks_no_overlap EXCLUDE USING gist (
        listing_id WITH =,
        period WITH &&
    )
);

DROP TRIGGER IF EXISTS categories_set_updated_at ON categories;
CREATE TRIGGER categories_set_updated_at
BEFORE UPDATE ON categories
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS campus_locations_set_updated_at ON campus_locations;
CREATE TRIGGER campus_locations_set_updated_at
BEFORE UPDATE ON campus_locations
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS listings_set_updated_at ON listings;
CREATE TRIGGER listings_set_updated_at
BEFORE UPDATE ON listings
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- Index katalog untuk NFR-01 (filter kategori + harga < 2 detik).
CREATE INDEX listings_catalog_category_idx ON listings (category_id, price_per_day) WHERE is_active;
CREATE INDEX listings_catalog_price_idx ON listings (price_per_day) WHERE is_active;
CREATE INDEX listings_owner_idx ON listings (owner_id, created_at DESC);
CREATE INDEX listings_title_trgm_idx ON listings USING gin (title gin_trgm_ops);

COMMENT ON TABLE listings IS 'Barang yang dipublikasikan penyedia. is_active=false berarti takedown admin.';
COMMENT ON COLUMN listings.price_per_day IS 'Harga pas integer (Rupiah). Tidak ada diskon/cicilan.';
COMMENT ON TABLE listing_blocks IS 'Tanggal tidak tersedia yang diblokir manual penyedia.';
COMMENT ON COLUMN listing_blocks.period IS 'DATERANGE inklusif; tidak boleh overlap per listing.';
