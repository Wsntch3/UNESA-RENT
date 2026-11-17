-- 0005_ktm_verification.up.sql
-- Verifikasi KTM mahasiswa Fakultas Teknik.
--
-- Satu user boleh punya banyak submission (riwayat), tetapi hanya satu yang
-- boleh berstatus pending pada satu waktu. File KTM disimpan di object storage;
-- database hanya menyimpan metadata dan storage key.

CREATE TABLE ktm_verifications (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    storage_key      TEXT NOT NULL UNIQUE,
    mime_type        VARCHAR(80) NOT NULL,
    file_size_bytes  BIGINT NOT NULL,
    checksum_sha256  CHAR(64),
    status           TEXT NOT NULL DEFAULT 'pending',
    submitted_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_by      UUID REFERENCES users(id) ON DELETE RESTRICT,
    reviewed_at      TIMESTAMPTZ,
    rejection_reason TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ktm_verifications_status_check CHECK (
        status IN ('pending', 'verified', 'rejected')
    ),
    CONSTRAINT ktm_verifications_mime_check CHECK (
        mime_type IN ('image/jpeg', 'image/png', 'application/pdf')
    ),
    CONSTRAINT ktm_verifications_file_size_check CHECK (
        file_size_bytes > 0 AND file_size_bytes <= 5 * 1024 * 1024
    ),
    CONSTRAINT ktm_verifications_checksum_format CHECK (
        checksum_sha256 IS NULL OR checksum_sha256 ~ '^[0-9a-f]{64}$'
    ),
    CONSTRAINT ktm_verifications_review_consistency CHECK (
        (status = 'pending'
            AND reviewed_by IS NULL
            AND reviewed_at IS NULL
            AND rejection_reason IS NULL)
        OR (status = 'verified'
            AND reviewed_by IS NOT NULL
            AND reviewed_at IS NOT NULL
            AND rejection_reason IS NULL)
        OR (status = 'rejected'
            AND reviewed_by IS NOT NULL
            AND reviewed_at IS NOT NULL
            AND rejection_reason IS NOT NULL
            AND char_length(btrim(rejection_reason)) > 0)
    )
);

CREATE UNIQUE INDEX ktm_one_pending_per_user
    ON ktm_verifications (user_id)
    WHERE status = 'pending';

CREATE INDEX ktm_user_submitted_idx
    ON ktm_verifications (user_id, submitted_at DESC);

CREATE INDEX ktm_status_submitted_idx
    ON ktm_verifications (status, submitted_at DESC);

DROP TRIGGER IF EXISTS ktm_verifications_set_updated_at ON ktm_verifications;
CREATE TRIGGER ktm_verifications_set_updated_at
BEFORE UPDATE ON ktm_verifications
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- Hanya admin_dev yang boleh menjadi reviewer KTM.
CREATE OR REPLACE FUNCTION ktm_verifications_validate_reviewer()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.reviewed_by IS NOT NULL THEN
        IF NOT EXISTS (
            SELECT 1 FROM users
            WHERE id = NEW.reviewed_by
              AND role = 'admin_dev'
        ) THEN
            RAISE EXCEPTION 'reviewed_by harus admin_dev'
                USING ERRCODE = '23514';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS ktm_verifications_validate_reviewer ON ktm_verifications;
CREATE TRIGGER ktm_verifications_validate_reviewer
BEFORE INSERT OR UPDATE ON ktm_verifications
FOR EACH ROW
EXECUTE FUNCTION ktm_verifications_validate_reviewer();

COMMENT ON TABLE ktm_verifications IS 'Riwayat verifikasi KTM; file privat di object storage.';
COMMENT ON COLUMN ktm_verifications.storage_key IS 'Key object storage privat; jangan pernah dikirim ke endpoint publik.';
COMMENT ON COLUMN ktm_verifications.reviewed_by IS 'Harus user dengan role admin_dev.';
