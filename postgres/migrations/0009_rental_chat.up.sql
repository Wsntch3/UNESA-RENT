-- 0009_rental_chat.up.sql
-- Chat internal per rental. Pesan append-only; file/lampiran tidak didukung.
-- Hanya peserta rental yang boleh mengakses; aturan itu ditegakkan backend.

CREATE TABLE rental_messages (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rental_id  UUID NOT NULL REFERENCES rentals(id) ON DELETE CASCADE,
    sender_id  UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    body       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT rental_messages_body_length CHECK (char_length(btrim(body)) BETWEEN 1 AND 2000)
);

CREATE INDEX rental_messages_rental_idx ON rental_messages (rental_id, created_at, id);

COMMENT ON TABLE rental_messages IS 'Pesan chat internal per rental; append-only, tanpa lampiran.';
