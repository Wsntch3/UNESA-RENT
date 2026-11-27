-- 0011_chat_encryption.down.sql

DROP TRIGGER IF EXISTS chat_access_log_append_only ON chat_access_log;
DROP FUNCTION IF EXISTS chat_access_log_append_only();
DROP TABLE IF EXISTS chat_access_log;

ALTER TABLE rental_messages DROP CONSTRAINT IF EXISTS rental_messages_payload_check;
ALTER TABLE rental_messages DROP CONSTRAINT IF EXISTS rental_messages_body_length;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM rental_messages WHERE body_ciphertext IS NOT NULL) THEN
        ALTER TABLE rental_messages ALTER COLUMN body SET NOT NULL;
    END IF;
END $$;

ALTER TABLE rental_messages DROP COLUMN IF EXISTS body_sha256;
ALTER TABLE rental_messages DROP COLUMN IF EXISTS body_length;
ALTER TABLE rental_messages DROP COLUMN IF EXISTS key_version;
ALTER TABLE rental_messages DROP COLUMN IF EXISTS body_nonce;
ALTER TABLE rental_messages DROP COLUMN IF EXISTS body_ciphertext;
