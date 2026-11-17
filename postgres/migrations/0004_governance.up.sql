-- 0004_governance.up.sql
-- Pengaduan, audit tindakan admin, dan notifikasi in-app.

CREATE TABLE reports (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id     UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    rental_id       UUID REFERENCES rentals(id) ON DELETE SET NULL,
    listing_id      UUID REFERENCES listings(id) ON DELETE SET NULL,
    type            TEXT NOT NULL,
    subject         VARCHAR(150) NOT NULL,
    body            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'open',
    assigned_to     UUID REFERENCES users(id) ON DELETE SET NULL,
    resolution_note TEXT,
    resolved_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT reports_type_check CHECK (type IN ('technical', 'safety')),
    CONSTRAINT reports_status_check CHECK (status IN ('open', 'in_review', 'resolved', 'rejected')),
    CONSTRAINT reports_subject_length CHECK (char_length(btrim(subject)) BETWEEN 3 AND 150),
    CONSTRAINT reports_body_length CHECK (char_length(btrim(body)) BETWEEN 10 AND 5000)
);

-- target_id sengaja tanpa FK karena polimorfik (listing/user/report/template).
CREATE TABLE moderation_actions (
    id          BIGSERIAL PRIMARY KEY,
    actor_id    UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action      TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id   UUID NOT NULL,
    reason      TEXT,
    metadata    JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT moderation_actions_action_check CHECK (
        action IN (
            'takedown_listing',
            'restore_listing',
            'suspend_user',
            'reactivate_user',
            'resolve_report',
            'reject_report',
            'approve_template'
        )
    ),
    CONSTRAINT moderation_actions_target_type_check CHECK (
        target_type IN ('listing', 'user', 'report', 'letter_template')
    )
);

CREATE TABLE notifications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       VARCHAR(60) NOT NULL,
    title      VARCHAR(150) NOT NULL,
    body       TEXT,
    data       JSONB NOT NULL DEFAULT '{}'::jsonb,
    read_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT notifications_title_length CHECK (char_length(btrim(title)) BETWEEN 3 AND 150)
);

DROP TRIGGER IF EXISTS reports_set_updated_at ON reports;
CREATE TRIGGER reports_set_updated_at
BEFORE UPDATE ON reports
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- Audit trail admin bersifat append-only.
CREATE OR REPLACE FUNCTION moderation_actions_append_only()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'moderation_actions bersifat append-only'
        USING ERRCODE = '23514';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER moderation_actions_append_only
BEFORE UPDATE OR DELETE ON moderation_actions
FOR EACH ROW
EXECUTE FUNCTION moderation_actions_append_only();

CREATE INDEX reports_status_idx ON reports (status, created_at DESC);
CREATE INDEX reports_reporter_idx ON reports (reporter_id, created_at DESC);
CREATE INDEX reports_assigned_idx ON reports (assigned_to, created_at DESC) WHERE assigned_to IS NOT NULL;
CREATE INDEX moderation_actions_target_idx ON moderation_actions (target_type, target_id, created_at DESC);
CREATE INDEX moderation_actions_actor_idx ON moderation_actions (actor_id, created_at DESC);
CREATE INDEX notifications_user_idx ON notifications (user_id, created_at DESC);
CREATE INDEX notifications_unread_idx ON notifications (user_id, created_at DESC) WHERE read_at IS NULL;

COMMENT ON TABLE reports IS 'Pengaduan pengguna; technical ditangani admin_dev, safety ditangani admin_bem.';
COMMENT ON TABLE moderation_actions IS 'Audit trail tindakan admin, termasuk takedown (FR-06).';
COMMENT ON TABLE notifications IS 'Notifikasi in-app, mis. pengajuan sewa baru untuk penyedia.';
