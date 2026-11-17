-- 0006_hardening.down.sql
-- Catatan: rollback gagal bila moderation_actions sudah berisi aksi KTM baru
-- (tabel append-only, barisnya tidak bisa dihapus). Di development gunakan make db-reset.

-- 5. Governance
ALTER TABLE reports DROP CONSTRAINT IF EXISTS reports_resolution_consistency;
DROP TRIGGER IF EXISTS moderation_actions_no_truncate ON moderation_actions;

ALTER TABLE moderation_actions DROP CONSTRAINT moderation_actions_action_check;
ALTER TABLE moderation_actions DROP CONSTRAINT moderation_actions_target_type_check;
ALTER TABLE moderation_actions ADD CONSTRAINT moderation_actions_action_check CHECK (
    action IN (
        'takedown_listing', 'restore_listing', 'suspend_user', 'reactivate_user',
        'resolve_report', 'reject_report', 'approve_template'
    )
);
ALTER TABLE moderation_actions ADD CONSTRAINT moderation_actions_target_type_check CHECK (
    target_type IN ('listing', 'user', 'report', 'letter_template')
);

-- 4. listing_blocks
DROP TRIGGER IF EXISTS listing_blocks_no_approved_overlap ON listing_blocks;
DROP FUNCTION IF EXISTS listing_blocks_no_approved_overlap();
ALTER TABLE listing_blocks DROP CONSTRAINT IF EXISTS listing_blocks_period_bounded;

-- 3. listings / rentals
ALTER TABLE listings DROP CONSTRAINT IF EXISTS listings_price_max;
ALTER TABLE rentals DROP CONSTRAINT IF EXISTS rentals_max_duration;
DROP TRIGGER IF EXISTS rentals_extra_guard ON rentals;
DROP FUNCTION IF EXISTS rentals_extra_guard();
DROP TRIGGER IF EXISTS listings_require_verified_owner ON listings;
DROP FUNCTION IF EXISTS listings_require_verified_owner();

-- 2. KTM
DROP TRIGGER IF EXISTS ktm_verifications_guard ON ktm_verifications;
DROP FUNCTION IF EXISTS ktm_verifications_guard();

UPDATE ktm_verifications
   SET status = 'rejected',
       rejection_reason = 'dicabut (rollback migration 0006)',
       revoked_at = NULL, revoked_by = NULL, revoke_reason = NULL
 WHERE status = 'revoked';

ALTER TABLE ktm_verifications DROP CONSTRAINT IF EXISTS ktm_purge_only_final;
ALTER TABLE ktm_verifications DROP CONSTRAINT IF EXISTS ktm_nim_confirmed_when_verified;
ALTER TABLE ktm_verifications DROP CONSTRAINT IF EXISTS ktm_nim_snapshot_required;
ALTER TABLE ktm_verifications DROP CONSTRAINT IF EXISTS ktm_checksum_required;
ALTER TABLE ktm_verifications DROP CONSTRAINT IF EXISTS ktm_revoker_not_self;
ALTER TABLE ktm_verifications DROP CONSTRAINT IF EXISTS ktm_reviewer_not_self;
ALTER TABLE ktm_verifications DROP CONSTRAINT ktm_verifications_review_consistency;
ALTER TABLE ktm_verifications DROP CONSTRAINT ktm_verifications_mime_check;
ALTER TABLE ktm_verifications DROP CONSTRAINT ktm_verifications_status_check;

DROP FUNCTION IF EXISTS is_ktm_verified(UUID);

ALTER TABLE ktm_verifications
    DROP COLUMN file_purged_at,
    DROP COLUMN revoke_reason,
    DROP COLUMN revoked_by,
    DROP COLUMN revoked_at,
    DROP COLUMN reviewer_confirmed_nim,
    DROP COLUMN nim_snapshot;

ALTER TABLE ktm_verifications ADD CONSTRAINT ktm_verifications_status_check
    CHECK (status IN ('pending', 'verified', 'rejected'));
ALTER TABLE ktm_verifications ADD CONSTRAINT ktm_verifications_mime_check
    CHECK (mime_type IN ('image/jpeg', 'image/png', 'application/pdf'));
ALTER TABLE ktm_verifications ADD CONSTRAINT ktm_verifications_review_consistency CHECK (
    (status = 'pending'
        AND reviewed_by IS NULL AND reviewed_at IS NULL AND rejection_reason IS NULL)
    OR (status = 'verified'
        AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL AND rejection_reason IS NULL)
    OR (status = 'rejected'
        AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL
        AND rejection_reason IS NOT NULL AND char_length(btrim(rejection_reason)) > 0)
);

CREATE OR REPLACE FUNCTION ktm_verifications_validate_reviewer()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.reviewed_by IS NOT NULL THEN
        IF NOT EXISTS (
            SELECT 1 FROM users WHERE id = NEW.reviewed_by AND role = 'admin_dev'
        ) THEN
            RAISE EXCEPTION 'reviewed_by harus admin_dev' USING ERRCODE = '23514';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 1. NIM
CREATE OR REPLACE FUNCTION users_resolve_study_program()
RETURNS TRIGGER AS $$
DECLARE
    matched_code VARCHAR(3);
BEGIN
    IF NEW.role = 'mahasiswa' THEN
        NEW.nim := btrim(NEW.nim);

        IF NEW.nim IS NULL OR NEW.nim = '' THEN
            RAISE EXCEPTION 'NIM wajib diisi untuk mahasiswa'
                USING ERRCODE = '23514';
        END IF;

        SELECT sp.code INTO matched_code
        FROM study_programs sp
        WHERE sp.faculty = 'Fakultas Teknik'
          AND NEW.nim LIKE sp.code || '%'
        ORDER BY char_length(sp.code) DESC
        LIMIT 1;

        IF matched_code IS NULL THEN
            RAISE EXCEPTION 'NIM % tidak cocok dengan kode program studi Fakultas Teknik', NEW.nim
                USING ERRCODE = '23514';
        END IF;

        NEW.study_program_code := matched_code;
    ELSE
        NEW.study_program_code := NULL;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

ALTER TABLE users DROP COLUMN IF EXISTS token_version;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_nim_format;
ALTER TABLE users ADD CONSTRAINT users_nim_format
    CHECK (nim IS NULL OR nim ~ '^[0-9]{6,20}$');

ALTER TABLE study_programs DROP CONSTRAINT IF EXISTS study_programs_faculty_code_format;
ALTER TABLE study_programs DROP CONSTRAINT IF EXISTS study_programs_code_format;
ALTER TABLE study_programs ADD CONSTRAINT study_programs_code_format
    CHECK (code ~ '^[0-9]{2,3}$');
ALTER TABLE study_programs DROP COLUMN IF EXISTS faculty_code;