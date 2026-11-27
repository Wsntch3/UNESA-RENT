-- 0012_cod_documents.up.sql
-- Dokumen per rental + state COD.

CREATE TABLE IF NOT EXISTS rental_documents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rental_id       UUID NOT NULL REFERENCES rentals(id) ON DELETE CASCADE,
    type            TEXT NOT NULL,
    storage_key     TEXT NOT NULL UNIQUE,
    mime_type       TEXT NOT NULL,
    file_size_bytes BIGINT NOT NULL,
    checksum_sha256 CHAR(64),
    uploaded_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    visibility      TEXT NOT NULL DEFAULT 'participants',
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT rental_documents_type_check CHECK (
        type IN ('agreement', 'handover_proof', 'return_proof', 'cod_receipt', 'other')
    ),
    CONSTRAINT rental_documents_mime_check CHECK (
        mime_type IN ('application/pdf', 'image/jpeg', 'image/png')
    ),
    CONSTRAINT rental_documents_size_check CHECK (
        file_size_bytes > 0 AND file_size_bytes <= 10 * 1024 * 1024
    ),
    CONSTRAINT rental_documents_visibility_check CHECK (
        visibility IN ('participants', 'admin')
    )
);

CREATE INDEX IF NOT EXISTS rental_documents_rental_idx
    ON rental_documents (rental_id, type, created_at DESC);

CREATE TABLE IF NOT EXISTS rental_cod (
    rental_id             UUID PRIMARY KEY REFERENCES rentals(id) ON DELETE CASCADE,
    agreed_at             TIMESTAMPTZ,
    handover_at           TIMESTAMPTZ,
    handover_by           UUID REFERENCES users(id) ON DELETE SET NULL,
    handover_confirmed_at TIMESTAMPTZ,
    handover_confirmed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    return_at             TIMESTAMPTZ,
    return_by             UUID REFERENCES users(id) ON DELETE SET NULL,
    return_confirmed_at   TIMESTAMPTZ,
    return_confirmed_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    amount_due            INTEGER CHECK (amount_due IS NULL OR amount_due >= 0),
    amount_paid           INTEGER CHECK (amount_paid IS NULL OR amount_paid >= 0),
    payment_method        TEXT CHECK (payment_method IS NULL OR payment_method IN ('cash', 'transfer')),
    settled_at            TIMESTAMPTZ,
    notes                 TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

DROP TRIGGER IF EXISTS rental_cod_set_updated_at ON rental_cod;
CREATE TRIGGER rental_cod_set_updated_at
BEFORE UPDATE ON rental_cod
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE rental_documents IS 'Arsip dokumen privat per rental; file di Supabase Storage.';
COMMENT ON TABLE rental_cod IS 'State serah terima dan pembayaran COD per rental.';
