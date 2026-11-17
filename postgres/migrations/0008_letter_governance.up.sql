-- 0008_letter_governance.up.sql
-- Governance template Surat Pernyataan Keamanan.
--
-- Aturan:
-- - Template aktif wajib punya referensi persetujuan offline.
-- - Naskah yang sudah pernah diaktifkan bersifat immutable; perubahan harus versi baru.
-- - Surat hanya boleh dibuat dari template yang sedang aktif.
-- - Metadata security_letters tidak boleh diubah setelah dibuat.

ALTER TABLE letter_templates
    ADD COLUMN IF NOT EXISTS approval_reference TEXT,
    ADD COLUMN IF NOT EXISTS approved_by_name VARCHAR(120),
    ADD COLUMN IF NOT EXISTS approved_offline_at DATE,
    ADD COLUMN IF NOT EXISTS activated_at TIMESTAMPTZ;

-- Backfill baris lama yang sudah aktif sebelum trigger dibuat.
-- Sengaja tidak mengisi approval_reference agar tidak memalsukan persetujuan.
UPDATE letter_templates
   SET activated_at = COALESCE(activated_at, now())
 WHERE activated_at IS NULL
   AND is_active;

ALTER TABLE letter_templates DROP CONSTRAINT IF EXISTS letter_templates_active_requires_approval;
ALTER TABLE letter_templates ADD CONSTRAINT letter_templates_active_requires_approval CHECK (
    NOT is_active
    OR (approval_reference IS NOT NULL AND char_length(btrim(approval_reference)) > 0)
) NOT VALID;

CREATE OR REPLACE FUNCTION letter_templates_guard()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.is_active THEN
            NEW.activated_at := now();
        END IF;
        RETURN NEW;
    END IF;

    -- Aktivasi pertama mengisi activated_at. Re-aktivasi mempertahankan nilai lama.
    IF NEW.is_active AND NEW.activated_at IS NULL THEN
        NEW.activated_at := COALESCE(OLD.activated_at, now());
    END IF;

    IF OLD.activated_at IS NOT NULL THEN
        IF (NEW.version, NEW.title, NEW.body,
            NEW.approval_reference, NEW.approved_by_name, NEW.approved_offline_at,
            NEW.created_by, NEW.activated_at)
           IS DISTINCT FROM
           (OLD.version, OLD.title, OLD.body,
            OLD.approval_reference, OLD.approved_by_name, OLD.approved_offline_at,
            OLD.created_by, OLD.activated_at)
        THEN
            RAISE EXCEPTION 'template yang sudah pernah diaktifkan bersifat immutable; buat versi baru'
                USING ERRCODE = '23514';
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS letter_templates_guard ON letter_templates;
CREATE TRIGGER letter_templates_guard
BEFORE INSERT OR UPDATE ON letter_templates
FOR EACH ROW
EXECUTE FUNCTION letter_templates_guard();

-- Template yang sudah pernah aktif tidak boleh dihapus; nonaktifkan saja.
CREATE OR REPLACE FUNCTION letter_templates_no_delete_activated()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.activated_at IS NOT NULL THEN
        RAISE EXCEPTION 'template yang sudah pernah diaktifkan tidak boleh dihapus; nonaktifkan saja'
            USING ERRCODE = '23514';
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS letter_templates_no_delete_activated ON letter_templates;
CREATE TRIGGER letter_templates_no_delete_activated
BEFORE DELETE ON letter_templates
FOR EACH ROW
EXECUTE FUNCTION letter_templates_no_delete_activated();

CREATE OR REPLACE FUNCTION security_letters_require_active_template()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM letter_templates
         WHERE id = NEW.template_id
           AND is_active
    ) THEN
        RAISE EXCEPTION 'Surat Pernyataan hanya boleh dibuat dari template aktif'
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS security_letters_require_active_template ON security_letters;
CREATE TRIGGER security_letters_require_active_template
BEFORE INSERT ON security_letters
FOR EACH ROW
EXECUTE FUNCTION security_letters_require_active_template();

CREATE OR REPLACE FUNCTION security_letters_immutable()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'security_letters bersifat immutable' USING ERRCODE = '23514';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS security_letters_immutable ON security_letters;
CREATE TRIGGER security_letters_immutable
BEFORE UPDATE ON security_letters
FOR EACH ROW
EXECUTE FUNCTION security_letters_immutable();

COMMENT ON COLUMN letter_templates.approval_reference IS 'Referensi persetujuan offline naskah; wajib untuk template aktif.';
COMMENT ON COLUMN letter_templates.approved_by_name IS 'Pihak yang menyetujui naskah (mis. BEM FT, Pembina FT, Pengelola).';
COMMENT ON COLUMN letter_templates.approved_offline_at IS 'Tanggal persetujuan offline naskah.';
COMMENT ON COLUMN letter_templates.activated_at IS 'Waktu template pertama kali diaktifkan; naskah menjadi immutable.';
COMMENT ON TABLE reports IS 'Pengaduan pengguna; technical dan safety ditangani admin_dev sampai role BEM tersedia.';
