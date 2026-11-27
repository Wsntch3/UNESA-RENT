-- 0013_admin_governance.down.sql
-- Kembalikan constraint lama. Pastikan tidak ada baris dengan nilai baru
-- (fraud, harassment, other, download_document, export_data,
-- delete_unverified_user, rental, rental_document) sebelum menjalankan ini.

DROP INDEX IF EXISTS moderation_actions_created_idx;
DROP INDEX IF EXISTS reports_created_idx;
DROP INDEX IF EXISTS users_created_idx;
DROP INDEX IF EXISTS rentals_admin_status_created_idx;

ALTER TABLE moderation_actions DROP CONSTRAINT IF EXISTS moderation_actions_target_type_check;
ALTER TABLE moderation_actions ADD CONSTRAINT moderation_actions_target_type_check CHECK (
    target_type IN ('listing', 'user', 'report', 'letter_template', 'ktm_verification')
);

ALTER TABLE moderation_actions DROP CONSTRAINT IF EXISTS moderation_actions_action_check;
ALTER TABLE moderation_actions ADD CONSTRAINT moderation_actions_action_check CHECK (
    action IN (
        'takedown_listing', 'restore_listing',
        'suspend_user', 'reactivate_user',
        'resolve_report', 'reject_report', 'approve_template',
        'verify_ktm', 'reject_ktm', 'revoke_ktm', 'view_ktm_file'
    )
);

ALTER TABLE reports DROP CONSTRAINT IF EXISTS reports_type_check;
ALTER TABLE reports ADD CONSTRAINT reports_type_check CHECK (
    type IN ('technical', 'safety')
);
