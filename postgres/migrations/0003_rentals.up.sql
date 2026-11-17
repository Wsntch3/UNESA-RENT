-- 0003_rentals.up.sql
-- Transaksi: pengajuan sewa, riwayat status, template surat, dan surat pernyataan.

-- Template otorisasi keamanan dari BEM. Hanya satu yang boleh aktif.
CREATE TABLE letter_templates (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version    VARCHAR(20) NOT NULL UNIQUE,
    title      VARCHAR(150) NOT NULL DEFAULT 'Surat Pernyataan Keamanan',
    body       TEXT NOT NULL,
    is_active  BOOLEAN NOT NULL DEFAULT false,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT letter_templates_body_not_empty CHECK (char_length(btrim(body)) > 0)
);

CREATE UNIQUE INDEX letter_templates_one_active_idx
    ON letter_templates (is_active)
    WHERE is_active;

-- Satu baris = satu pengajuan/transaksi sewa.
-- total_days dan total_price dihitung database; end_date inklusif.
--
-- [MARKER: EVALUASI PAYMENT] BR-03
-- Sistem ini murni pencatatan. TIDAK ADA tabel payment/escrow dan tidak ada
-- integrasi Midtrans/API pembayaran. Pembayaran dilakukan COD/transfer manual
-- antar mahasiswa saat serah terima. Jika versi berikutnya mengintegrasikan
-- payment gateway/escrow, tambahkan tabel pembayaran terpisah dan jangan
-- mengubah total_price menjadi status pembayaran.
CREATE TABLE rentals (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id            UUID NOT NULL REFERENCES listings(id) ON DELETE RESTRICT,
    renter_id             UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    owner_id              UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    start_date            DATE NOT NULL,
    end_date              DATE NOT NULL,
    price_per_day_snapshot INTEGER NOT NULL,
    total_days            INTEGER GENERATED ALWAYS AS ((end_date - start_date) + 1) STORED,
    total_price           INTEGER GENERATED ALWAYS AS (((end_date - start_date) + 1) * price_per_day_snapshot) STORED,
    status                TEXT NOT NULL DEFAULT 'pending',
    rejection_reason      TEXT,
    approved_at           TIMESTAMPTZ,
    rejected_at           TIMESTAMPTZ,
    cancelled_at          TIMESTAMPTZ,
    completed_at          TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT rentals_date_order CHECK (end_date >= start_date),
    CONSTRAINT rentals_price_positive CHECK (price_per_day_snapshot > 0),
    CONSTRAINT rentals_renter_not_owner CHECK (renter_id <> owner_id),
    CONSTRAINT rentals_status_check CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled', 'completed')),
    -- FR-03: hanya transaksi approved yang memblokir tanggal penyewa lain.
    -- Pending boleh overlap; approval kedua akan ditolak constraint ini.
    CONSTRAINT rentals_no_double_booking EXCLUDE USING gist (
        listing_id WITH =,
        (daterange(start_date, end_date, '[]')) WITH &&
    ) WHERE (status = 'approved')
);

-- Riwayat transisi status untuk bukti sengketa (BR-04).
CREATE TABLE rental_events (
    id          BIGSERIAL PRIMARY KEY,
    rental_id   UUID NOT NULL REFERENCES rentals(id) ON DELETE CASCADE,
    from_status TEXT,
    to_status   TEXT NOT NULL,
    actor_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    note        TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT rental_events_from_status_check CHECK (
        from_status IS NULL OR from_status IN ('pending', 'approved', 'rejected', 'cancelled', 'completed')
    ),
    CONSTRAINT rental_events_to_status_check CHECK (
        to_status IN ('pending', 'approved', 'rejected', 'cancelled', 'completed')
    )
);

-- PDF disimpan di object storage; database hanya menyimpan metadata (NFR-03).
CREATE TABLE security_letters (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rental_id       UUID NOT NULL UNIQUE REFERENCES rentals(id) ON DELETE CASCADE,
    template_id     UUID NOT NULL REFERENCES letter_templates(id) ON DELETE RESTRICT,
    letter_number   VARCHAR(60) NOT NULL UNIQUE,
    pdf_storage_key TEXT NOT NULL UNIQUE,
    checksum_sha256 CHAR(64),
    generated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT security_letters_checksum_format CHECK (
        checksum_sha256 IS NULL OR checksum_sha256 ~ '^[0-9a-f]{64}$'
    )
);

-- Surat Pernyataan hanya boleh dibuat untuk transaksi yang sudah approved (FR-05).
CREATE OR REPLACE FUNCTION security_letters_require_approved()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM rentals
        WHERE id = NEW.rental_id
          AND status = 'approved'
    ) THEN
        RAISE EXCEPTION 'Surat Pernyataan hanya boleh dibuat untuk rental berstatus approved'
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER security_letters_require_approved
BEFORE INSERT ON security_letters
FOR EACH ROW
EXECUTE FUNCTION security_letters_require_approved();

-- Validasi lintas tabel dan transisi status rental.
CREATE OR REPLACE FUNCTION rentals_before_write()
RETURNS TRIGGER AS $$
DECLARE
    listing_owner  UUID;
    listing_active BOOLEAN;
    listing_price  INTEGER;
BEGIN
    IF TG_OP = 'UPDATE' AND NEW.listing_id <> OLD.listing_id THEN
        RAISE EXCEPTION 'listing_id tidak boleh diubah'
            USING ERRCODE = '23514';
    END IF;

    IF TG_OP = 'UPDATE' AND NEW.renter_id <> OLD.renter_id THEN
        RAISE EXCEPTION 'renter_id tidak boleh diubah'
            USING ERRCODE = '23514';
    END IF;

    IF TG_OP = 'UPDATE' AND NEW.price_per_day_snapshot <> OLD.price_per_day_snapshot THEN
        RAISE EXCEPTION 'price_per_day_snapshot tidak boleh diubah'
            USING ERRCODE = '23514';
    END IF;

    SELECT l.owner_id, l.is_active, l.price_per_day
      INTO listing_owner, listing_active, listing_price
      FROM listings l
     WHERE l.id = NEW.listing_id;

    IF listing_owner IS NULL THEN
        RAISE EXCEPTION 'Listing tidak ditemukan'
            USING ERRCODE = '23503';
    END IF;

    -- owner_id dan harga selalu diturunkan dari listing, bukan dari input client.
    NEW.owner_id := listing_owner;
    IF TG_OP = 'INSERT' THEN
        NEW.price_per_day_snapshot := listing_price;
    END IF;

    IF TG_OP = 'INSERT' AND NOT listing_active THEN
        RAISE EXCEPTION 'Listing sedang tidak aktif'
            USING ERRCODE = '23514';
    END IF;

    IF TG_OP = 'UPDATE' AND NEW.status = 'approved' AND OLD.status <> 'approved' AND NOT listing_active THEN
        RAISE EXCEPTION 'Listing sudah ditakedown dan tidak bisa disetujui'
            USING ERRCODE = '23514';
    END IF;

    IF TG_OP = 'UPDATE' AND NEW.status <> OLD.status THEN
        IF NOT (
            (OLD.status = 'pending' AND NEW.status IN ('approved', 'rejected', 'cancelled'))
            OR (OLD.status = 'approved' AND NEW.status IN ('completed', 'cancelled'))
        ) THEN
            RAISE EXCEPTION 'Transisi status rental tidak valid: % -> %', OLD.status, NEW.status
                USING ERRCODE = '23514';
        END IF;
    END IF;

    IF NEW.end_date < NEW.start_date THEN
        RAISE EXCEPTION 'end_date harus sama dengan atau setelah start_date'
            USING ERRCODE = '23514';
    END IF;

    -- FR-03: tanggal yang diblokir penyedia tidak boleh dipakai pending/approved.
    IF NEW.status IN ('pending', 'approved') THEN
        IF EXISTS (
            SELECT 1
              FROM listing_blocks lb
             WHERE lb.listing_id = NEW.listing_id
               AND lb.period && daterange(NEW.start_date, NEW.end_date, '[]')
        ) THEN
            RAISE EXCEPTION 'Rentang tanggal bertabrakan dengan tanggal yang diblokir penyedia'
                USING ERRCODE = '23P01';
        END IF;
    END IF;

    IF NEW.status = 'approved' AND NEW.approved_at IS NULL THEN
        NEW.approved_at := now();
    END IF;
    IF NEW.status = 'rejected' AND NEW.rejected_at IS NULL THEN
        NEW.rejected_at := now();
    END IF;
    IF NEW.status = 'cancelled' AND NEW.cancelled_at IS NULL THEN
        NEW.cancelled_at := now();
    END IF;
    IF NEW.status = 'completed' AND NEW.completed_at IS NULL THEN
        NEW.completed_at := now();
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS rentals_before_write ON rentals;
CREATE TRIGGER rentals_before_write
BEFORE INSERT OR UPDATE ON rentals
FOR EACH ROW
EXECUTE FUNCTION rentals_before_write();

DROP TRIGGER IF EXISTS rentals_set_updated_at ON rentals;
CREATE TRIGGER rentals_set_updated_at
BEFORE UPDATE ON rentals
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS letter_templates_set_updated_at ON letter_templates;
CREATE TRIGGER letter_templates_set_updated_at
BEFORE UPDATE ON letter_templates
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE INDEX rentals_renter_idx ON rentals (renter_id, created_at DESC);
CREATE INDEX rentals_owner_idx ON rentals (owner_id, created_at DESC);
CREATE INDEX rentals_listing_idx ON rentals (listing_id, start_date, end_date);
CREATE INDEX rentals_status_idx ON rentals (status, created_at DESC);
CREATE INDEX rental_events_rental_idx ON rental_events (rental_id, created_at);

COMMENT ON TABLE rentals IS 'Pengajuan/transaksi sewa. end_date inklusif; total_price dihitung database.';
COMMENT ON COLUMN rentals.price_per_day_snapshot IS 'Snapshot harga saat pengajuan agar perubahan listing tidak mengubah transaksi.';
COMMENT ON TABLE security_letters IS 'Metadata PDF Surat Pernyataan Keamanan; file disimpan di object storage.';
COMMENT ON TABLE rental_events IS 'Timeline status untuk audit dan penanganan sengketa.';
