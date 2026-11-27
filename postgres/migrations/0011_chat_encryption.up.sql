-- 0011_chat_encryption.up.sql
-- Isi chat dienkripsi AES-256-GCM. Kolom body lama dipertahankan sementara
-- untuk kompatibilitas; command backfill akan mengosongkannya.

ALTER TABLE rental_messages
    ADD COLUMN IF NOT EXISTS body_ciphertext BYTEA,
    ADD COLUMN IF NOT EXISTS body_nonce BYTEA,
    ADD COLUMN IF NOT EXISTS key_version SMALLINT,
    ADD COLUMN IF NOT EXISTS body_length INTEGER,
    ADD COLUMN IF NOT EXISTS body_sha256 CHAR(64);

ALTER TABLE rental_messages ALTER COLUMN body DROP NOT NULL;

ALTER TABLE rental_messages DROP CONSTRAINT IF EXISTS rental_messages_body_length;
ALTER TABLE rental_messages ADD CONSTRAINT rental_messages_body_length CHECK (
    (body IS NULL OR char_length(btrim(body)) BETWEEN 1 AND 2000)
    AND (body_length IS NULL OR body_length BETWEEN 1 AND 2000)
);

ALTER TABLE rental_messages DROP CONSTRAINT IF EXISTS rental_messages_payload_check;
ALTER TABLE rental_messages ADD CONSTRAINT rental_messages_payload_check CHECK (
    (body IS NOT NULL AND body_ciphertext IS NULL AND body_nonce IS NULL AND key_version IS NULL)
    OR
    (body IS NULL AND body_ciphertext IS NOT NULL AND body_nonce IS NOT NULL
        AND key_version IS NOT NULL AND body_length IS NOT NULL AND body_sha256 IS NOT NULL)
);

CREATE TABLE IF NOT EXISTS chat_access_log (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id   UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    rental_id  UUID NOT NULL REFERENCES rentals(id) ON DELETE CASCADE,
    message_id UUID REFERENCES rental_messages(id) ON DELETE SET NULL,
    action     TEXT NOT NULL,
    reason     TEXT NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chat_access_log_action_check CHECK (
        action IN ('reveal', 'export_transcript')
    ),
    CONSTRAINT chat_access_log_reason_length CHECK (
        char_length(btrim(reason)) BETWEEN 10 AND 1000
    )
);

CREATE INDEX IF NOT EXISTS chat_access_log_admin_idx
    ON chat_access_log (admin_id, created_at DESC);
CREATE INDEX IF NOT EXISTS chat_access_log_rental_idx
    ON chat_access_log (rental_id, created_at DESC);

CREATE OR REPLACE FUNCTION chat_access_log_append_only()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'chat_access_log bersifat append-only' USING ERRCODE = '23514';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS chat_access_log_append_only ON chat_access_log;
CREATE TRIGGER chat_access_log_append_only
BEFORE UPDATE OR DELETE ON chat_access_log
FOR EACH ROW EXECUTE FUNCTION chat_access_log_append_only();

COMMENT ON TABLE chat_access_log IS 'Audit setiap kali admin membuka/mengekspor isi chat.';
