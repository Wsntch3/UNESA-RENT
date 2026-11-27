-- 0013_admin_governance.up.sql

ALTER TABLE reports DROP CONSTRAINT IF EXISTS reports_type_check;
ALTER TABLE reports ADD CONSTRAINT reports_type_check CHECK (
    type IN ('technical', 'safety', 'fraud', 'harassment', 'other')
);

ALTER TABLE moderation_actions DROP CONSTRAINT IF EXISTS moderation_actions_action_check;
ALTER TABLE moderation_actions ADD CONSTRAINT moderation_actions_action_check CHECK (
    action IN (
        'takedown_listing', 'restore_listing',
        'suspend_user', 'reactivate_user',
        'resolve_report', 'reject_report', 'approve_template',
        'delete_unverified_user', 'download_document', 'export_data', 'revoke_ktm',
        'verify_ktm', 'reject_ktm', 'view_ktm_file'
    )
);

ALTER TABLE moderation_actions DROP CONSTRAINT IF EXISTS moderation_actions_target_type_check;
ALTER TABLE moderation_actions ADD CONSTRAINT moderation_actions_target_type_check CHECK (
    target_type IN (
        'listing', 'user', 'report', 'letter_template',
        'rental', 'rental_document', 'ktm', 'ktm_verification'
    )
);

CREATE INDEX IF NOT EXISTS rentals_admin_status_created_idx
    ON rentals (status, created_at DESC);
CREATE INDEX IF NOT EXISTS users_created_idx
    ON users (created_at DESC);
CREATE INDEX IF NOT EXISTS reports_created_idx
    ON reports (created_at DESC);
CREATE INDEX IF NOT EXISTS moderation_actions_created_idx
    ON moderation_actions (created_at DESC);
