-- 0006_hardening.up.sql
-- Hardening berdasarkan review: format NIM UNESA, token_version, state machine KTM,
-- penegakan "KTM verified" di listings/rentals, dan celah kecil lain.
--
-- Constraint bertanda NOT VALID tetap berlaku untuk INSERT/UPDATE baru tetapi tidak
-- memblokir baris lama. Setelah data dev dibersihkan (make db-reset), boleh
-- dijalankan: ALTER TABLE ... VALIDATE CONSTRAINT <nama>;
-- BELUM DIUJI: jalankan di database dev kosong dulu.

-- ===========================================================================
-- 1. NIM UNESA: AA FF PPP J UUU (11 digit)
--    AA=angkatan, FF=kode fakultas, PPP=kode prodi, J=jenjang, UUU=nomor urut
-- ===========================================================================
ALTER TABLE study_programs ADD COLUMN IF NOT EXISTS faculty_code CHAR(2);

ALTER TABLE study_programs DROP CONSTRAINT IF EXISTS study_programs_code_format;
ALTER TABLE study_programs ADD CONSTRAINT study_programs_code_format
    CHECK (code ~ '^[0-9]{3}$') NOT VALID;
ALTER TABLE study_programs ADD CONSTRAINT study_programs_faculty_code_format
    CHECK (faculty_code IS NULL OR faculty_code ~ '^[0-9]{2}$');

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_nim_format;
ALTER TABLE users ADD CONSTRAINT users_nim_format
    CHECK (nim IS NULL OR nim ~ '^[0-9]{11}$') NOT VALID;

-- Dipakai untuk mencabut sesi: naikkan nilainya saat suspend/ganti password/ganti role.
ALTER TABLE users ADD COLUMN IF NOT EXISTS token_version INTEGER NOT NULL DEFAULT 0;

CREATE OR REPLACE FUNCTION users_resolve_study_program()
RETURNS TRIGGER AS $$
DECLARE
    matched_code VARCHAR(3);
BEGIN
    IF NEW.role = 'mahasiswa' THEN
        NEW.nim := btrim(NEW.nim);

        IF NEW.nim IS NULL OR NEW.nim = '' THEN
            RAISE EXCEPTION 'NIM wajib diisi untuk mahasiswa'
                USING ERRCODE = '23514', CONSTRAINT = 'users_mahasiswa_nim_required';
        END IF;

        IF NEW.nim !~ '^[0-9]{11}$' THEN
            RAISE EXCEPTION 'NIM harus 11 digit angka'
                USING ERRCODE = '23514', CONSTRAINT = 'users_nim_format';
        END IF;

        -- digit 3-4 = fakultas, digit 5-7 = prodi
        SELECT sp.code INTO matched_code
        FROM study_programs sp
        WHERE sp.faculty = 'Fakultas Teknik'
          AND sp.faculty_code = substr(NEW.nim, 3, 2)
          AND sp.code = substr(NEW.nim, 5, 3)
        LIMIT 1;

        IF matched_code IS NULL THEN
            RAISE EXCEPTION 'NIM % tidak cocok dengan fakultas/prodi Fakultas Teknik', NEW.nim
                USING ERRCODE = '23514', CONSTRAINT = 'users_nim_prodi';
        END IF;

        NEW.study_program_code := matched_code;
    ELSE
        NEW.study_program_code := NULL;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ===========================================================================
-- 2. KTM: terikat NIM, state machine, pencabutan, retensi file
-- ===========================================================================
ALTER TABLE ktm_verifications
    ADD COLUMN nim_snapshot           VARCHAR(20),
    ADD COLUMN reviewer_confirmed_nim BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN revoked_at             TIMESTAMPTZ,
    ADD COLUMN revoked_by             UUID REFERENCES users(id) ON DELETE RESTRICT,
    ADD COLUMN revoke_reason          TEXT,
    ADD COLUMN file_purged_at         TIMESTAMPTZ;

UPDATE ktm_verifications k
   SET nim_snapshot = u.nim
  FROM users u
 WHERE u.id = k.user_id;

ALTER TABLE ktm_verifications DROP CONSTRAINT ktm_verifications_status_check;
ALTER TABLE ktm_verifications DROP CONSTRAINT ktm_verifications_review_consistency;
ALTER TABLE ktm_verifications DROP CONSTRAINT ktm_verifications_mime_check;

ALTER TABLE ktm_verifications ADD CONSTRAINT ktm_verifications_status_check
    CHECK (status IN ('pending', 'verified', 'rejected', 'revoked'));

-- Hanya gambar; PDF punya permukaan serangan lebih besar.
ALTER TABLE ktm_verifications ADD CONSTRAINT ktm_verifications_mime_check
    CHECK (mime_type IN ('image/jpeg', 'image/png')) NOT VALID;

ALTER TABLE ktm_verifications ADD CONSTRAINT ktm_verifications_review_consistency CHECK (
    (status = 'pending'
        AND reviewed_by IS NULL AND reviewed_at IS NULL AND rejection_reason IS NULL
        AND revoked_at IS NULL AND revoked_by IS NULL AND revoke_reason IS NULL)
    OR (status = 'verified'
        AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL AND rejection_reason IS NULL
        AND revoked_at IS NULL AND revoked_by IS NULL AND revoke_reason IS NULL)
    OR (status = 'rejected'
        AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL
        AND rejection_reason IS NOT NULL AND char_length(btrim(rejection_reason)) > 0
        AND revoked_at IS NULL AND revoked_by IS NULL AND revoke_reason IS NULL)
    OR (status = 'revoked'
        AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL AND rejection_reason IS NULL
        AND revoked_at IS NOT NULL AND revoked_by IS NOT NULL
        AND revoke_reason IS NOT NULL AND char_length(btrim(revoke_reason)) > 0)
);

ALTER TABLE ktm_verifications ADD CONSTRAINT ktm_reviewer_not_self
    CHECK (reviewed_by IS NULL OR reviewed_by <> user_id);
ALTER TABLE ktm_verifications ADD CONSTRAINT ktm_revoker_not_self
    CHECK (revoked_by IS NULL OR revoked_by <> user_id);
ALTER TABLE ktm_verifications ADD CONSTRAINT ktm_checksum_required
    CHECK (checksum_sha256 IS NOT NULL) NOT VALID;
ALTER TABLE ktm_verifications ADD CONSTRAINT ktm_nim_snapshot_required
    CHECK (nim_snapshot IS NOT NULL) NOT VALID;
-- Reviewer wajib mencentang bahwa NIM di kartu sama dengan NIM akun.
ALTER TABLE ktm_verifications ADD CONSTRAINT ktm_nim_confirmed_when_verified
    CHECK (status NOT IN ('verified', 'revoked') OR reviewer_confirmed_nim) NOT VALID;
ALTER TABLE ktm_verifications ADD CONSTRAINT ktm_purge_only_final
    CHECK (file_purged_at IS NULL OR status <> 'pending');

-- Satu-satunya definisi "user terverifikasi": KTM verified yang NIM-nya masih
-- sama dengan NIM akun sekarang, dan akun masih aktif.
CREATE OR REPLACE FUNCTION is_ktm_verified(p_user_id UUID)
RETURNS BOOLEAN
LANGUAGE sql STABLE AS $$
    SELECT EXISTS (
        SELECT 1
          FROM users u
          JOIN ktm_verifications k ON k.user_id = u.id
         WHERE u.id = p_user_id
           AND u.status = 'active'
           AND u.role = 'mahasiswa'
           AND k.status = 'verified'
           AND k.nim_snapshot = u.nim
    )
$$;

CREATE OR REPLACE FUNCTION ktm_verifications_guard()
RETURNS TRIGGER AS $$
DECLARE
    v_role   TEXT;
    v_nim    VARCHAR(20);
    v_status TEXT;
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.status <> 'pending' THEN
            RAISE EXCEPTION 'submission baru harus berstatus pending' USING ERRCODE = '23514';
        END IF;

        SELECT role, nim, status INTO v_role, v_nim, v_status
          FROM users WHERE id = NEW.user_id;

        IF v_role IS DISTINCT FROM 'mahasiswa' OR v_nim IS NULL THEN
            RAISE EXCEPTION 'hanya mahasiswa ber-NIM yang boleh mengajukan KTM' USING ERRCODE = '23514';
        END IF;
        IF v_status <> 'active' THEN
            RAISE EXCEPTION 'akun tidak aktif' USING ERRCODE = '23514';
        END IF;
        IF is_ktm_verified(NEW.user_id) THEN
            RAISE EXCEPTION 'akun sudah terverifikasi' USING ERRCODE = '23514';
        END IF;

        NEW.nim_snapshot := v_nim;
        RETURN NEW;
    END IF;

    -- UPDATE
    IF NEW.user_id <> OLD.user_id
       OR NEW.storage_key <> OLD.storage_key
       OR NEW.submitted_at <> OLD.submitted_at
       OR NEW.nim_snapshot IS DISTINCT FROM OLD.nim_snapshot THEN
        RAISE EXCEPTION 'user_id, storage_key, submitted_at, nim_snapshot tidak boleh diubah'
            USING ERRCODE = '23514';
    END IF;

    IF NEW.status <> OLD.status THEN
        IF NOT (
            (OLD.status = 'pending'  AND NEW.status IN ('verified', 'rejected'))
            OR (OLD.status = 'verified' AND NEW.status = 'revoked')
        ) THEN
            RAISE EXCEPTION 'transisi status KTM tidak valid: % -> %', OLD.status, NEW.status
                USING ERRCODE = '23514';
        END IF;
    ELSIF OLD.status <> 'pending' THEN
        -- Baris final: hanya file_purged_at yang boleh berubah.
        IF (NEW.mime_type, NEW.file_size_bytes, NEW.checksum_sha256,
            NEW.reviewed_by, NEW.reviewed_at, NEW.rejection_reason,
            NEW.reviewer_confirmed_nim,
            NEW.revoked_at, NEW.revoked_by, NEW.revoke_reason)
           IS DISTINCT FROM
           (OLD.mime_type, OLD.file_size_bytes, OLD.checksum_sha256,
            OLD.reviewed_by, OLD.reviewed_at, OLD.rejection_reason,
            OLD.reviewer_confirmed_nim,
            OLD.revoked_at, OLD.revoked_by, OLD.revoke_reason) THEN
            RAISE EXCEPTION 'keputusan KTM bersifat final' USING ERRCODE = '23514';
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS ktm_verifications_guard ON ktm_verifications;
CREATE TRIGGER ktm_verifications_guard
BEFORE INSERT OR UPDATE ON ktm_verifications
FOR EACH ROW
EXECUTE FUNCTION ktm_verifications_guard();

-- Reviewer/revoker harus admin_dev yang AKTIF. Dicek hanya saat nilainya berubah,
-- supaya admin yang kemudian di-suspend tidak memblokir update lain (mis. purge).
CREATE OR REPLACE FUNCTION ktm_verifications_validate_reviewer()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.reviewed_by IS NOT NULL
       AND (TG_OP = 'INSERT' OR NEW.reviewed_by IS DISTINCT FROM OLD.reviewed_by) THEN
        IF NOT EXISTS (
            SELECT 1 FROM users
             WHERE id = NEW.reviewed_by AND role = 'admin_dev' AND status = 'active'
        ) THEN
            RAISE EXCEPTION 'reviewed_by harus admin_dev yang aktif' USING ERRCODE = '23514';
        END IF;
    END IF;

    IF NEW.revoked_by IS NOT NULL
       AND (TG_OP = 'INSERT' OR NEW.revoked_by IS DISTINCT FROM OLD.revoked_by) THEN
        IF NOT EXISTS (
            SELECT 1 FROM users
             WHERE id = NEW.revoked_by AND role = 'admin_dev' AND status = 'active'
        ) THEN
            RAISE EXCEPTION 'revoked_by harus admin_dev yang aktif' USING ERRCODE = '23514';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMENT ON COLUMN ktm_verifications.nim_snapshot IS 'NIM akun saat submission; verified hanya berlaku bila masih sama dengan users.nim.';
COMMENT ON COLUMN ktm_verifications.file_purged_at IS 'Waktu file di object storage dihapus (kebijakan retensi).';
COMMENT ON FUNCTION is_ktm_verified(UUID) IS 'Definisi tunggal user terverifikasi; dipakai trigger listings/rentals dan backend.';

-- ===========================================================================
-- 3. Penegakan verified + active pada listings dan rentals
-- ===========================================================================
CREATE OR REPLACE FUNCTION listings_require_verified_owner()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT is_ktm_verified(NEW.owner_id) THEN
        RAISE EXCEPTION 'penyedia harus mahasiswa aktif dengan KTM terverifikasi'
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS listings_require_verified_owner ON listings;
CREATE TRIGGER listings_require_verified_owner
BEFORE INSERT ON listings
FOR EACH ROW
EXECUTE FUNCTION listings_require_verified_owner();

-- Nama berurutan setelah rentals_before_write (owner_id sudah diturunkan dari listing).
CREATE OR REPLACE FUNCTION rentals_extra_guard()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.start_date < (now() AT TIME ZONE 'Asia/Jakarta')::date THEN
            RAISE EXCEPTION 'start_date tidak boleh di masa lalu' USING ERRCODE = '23514';
        END IF;
        IF NOT is_ktm_verified(NEW.renter_id) THEN
            RAISE EXCEPTION 'penyewa harus mahasiswa aktif dengan KTM terverifikasi'
                USING ERRCODE = '23514';
        END IF;
    ELSIF NEW.status = 'approved' AND OLD.status <> 'approved' THEN
        IF NOT is_ktm_verified(NEW.owner_id) THEN
            RAISE EXCEPTION 'penyedia tidak lagi terverifikasi/aktif' USING ERRCODE = '23514';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS rentals_extra_guard ON rentals;
CREATE TRIGGER rentals_extra_guard
BEFORE INSERT OR UPDATE ON rentals
FOR EACH ROW
EXECUTE FUNCTION rentals_extra_guard();

-- Batas durasi 30 hari (asumsi; sesuaikan) dan batas harga agar INTEGER tidak overflow:
-- 30 hari x Rp10.000.000 = Rp300.000.000 < 2,1 miliar.
ALTER TABLE rentals ADD CONSTRAINT rentals_max_duration
    CHECK (end_date - start_date < 30) NOT VALID;
ALTER TABLE listings ADD CONSTRAINT listings_price_max
    CHECK (price_per_day <= 10000000) NOT VALID;

-- ===========================================================================
-- 4. listing_blocks: tidak boleh menimpa rental approved, dan harus berbatas
-- ===========================================================================
ALTER TABLE listing_blocks ADD CONSTRAINT listing_blocks_period_bounded
    CHECK (NOT lower_inf(period) AND NOT upper_inf(period)) NOT VALID;

CREATE OR REPLACE FUNCTION listing_blocks_no_approved_overlap()
RETURNS TRIGGER AS $$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM rentals r
         WHERE r.listing_id = NEW.listing_id
           AND r.status = 'approved'
           AND daterange(r.start_date, r.end_date, '[]') && NEW.period
    ) THEN
        RAISE EXCEPTION 'periode blokir bertabrakan dengan rental yang sudah approved'
            USING ERRCODE = '23P01';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS listing_blocks_no_approved_overlap ON listing_blocks;
CREATE TRIGGER listing_blocks_no_approved_overlap
BEFORE INSERT OR UPDATE ON listing_blocks
FOR EACH ROW
EXECUTE FUNCTION listing_blocks_no_approved_overlap();

-- ===========================================================================
-- 5. Governance: audit KTM, TRUNCATE, konsistensi report
-- ===========================================================================
ALTER TABLE moderation_actions DROP CONSTRAINT moderation_actions_action_check;
ALTER TABLE moderation_actions DROP CONSTRAINT moderation_actions_target_type_check;

ALTER TABLE moderation_actions ADD CONSTRAINT moderation_actions_action_check CHECK (
    action IN (
        'takedown_listing', 'restore_listing',
        'suspend_user', 'reactivate_user',
        'resolve_report', 'reject_report',
        'approve_template',
        'verify_ktm', 'reject_ktm', 'revoke_ktm', 'view_ktm_file'
    )
);
ALTER TABLE moderation_actions ADD CONSTRAINT moderation_actions_target_type_check CHECK (
    target_type IN ('listing', 'user', 'report', 'letter_template', 'ktm_verification')
);

CREATE TRIGGER moderation_actions_no_truncate
BEFORE TRUNCATE ON moderation_actions
FOR EACH STATEMENT
EXECUTE FUNCTION moderation_actions_append_only();

ALTER TABLE reports ADD CONSTRAINT reports_resolution_consistency
    CHECK ((status IN ('resolved', 'rejected')) = (resolved_at IS NOT NULL)) NOT VALID;