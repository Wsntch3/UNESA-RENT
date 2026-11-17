-- 0001_identity.up.sql
-- Identitas: program studi Fakultas Teknik, akun, role, dan kontak privat.
--
-- Catatan: file ini sengaja defensif (IF NOT EXISTS / DO block) supaya volume
-- lama yang masih memakai initdb/001_schema.sql tetap bisa dimigrasikan.

CREATE EXTENSION IF NOT EXISTS btree_gist;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ---------------------------------------------------------------------------
-- study_programs
-- Kode prodi 2-3 digit. NIM mahasiswa di-resolve ke kode ini lewat trigger.
-- Seed masih placeholder; ganti dengan daftar resmi FT UNESA di seed/dev_seed.sql.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS study_programs (
    code       VARCHAR(3) PRIMARY KEY,
    name       VARCHAR(120) NOT NULL,
    faculty    VARCHAR(80) NOT NULL DEFAULT 'Fakultas Teknik',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT study_programs_code_format CHECK (code ~ '^[0-9]{2,3}$'),
    CONSTRAINT study_programs_name_length CHECK (char_length(btrim(name)) BETWEEN 2 AND 120)
);

-- ---------------------------------------------------------------------------
-- users
-- role: mahasiswa | admin_dev | admin_bem
-- status: active | suspended
-- NIM hanya wajib untuk mahasiswa; admin tidak punya NIM.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name               VARCHAR(80) NOT NULL,
    email              VARCHAR(254) NOT NULL,
    password_hash      TEXT NOT NULL,
    role               TEXT NOT NULL DEFAULT 'mahasiswa',
    nim                VARCHAR(20),
    study_program_code VARCHAR(3),
    status             TEXT NOT NULL DEFAULT 'active',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Kolom tambahan untuk volume lama yang sudah punya tabel users.
ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'mahasiswa';
ALTER TABLE users ADD COLUMN IF NOT EXISTS nim VARCHAR(20);
ALTER TABLE users ADD COLUMN IF NOT EXISTS study_program_code VARCHAR(3);
ALTER TABLE users ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'users_role_check' AND conrelid = 'users'::regclass) THEN
        ALTER TABLE users ADD CONSTRAINT users_role_check
            CHECK (role IN ('mahasiswa', 'admin_dev', 'admin_bem'));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'users_status_check' AND conrelid = 'users'::regclass) THEN
        ALTER TABLE users ADD CONSTRAINT users_status_check
            CHECK (status IN ('active', 'suspended'));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'users_name_length' AND conrelid = 'users'::regclass) THEN
        ALTER TABLE users ADD CONSTRAINT users_name_length
            CHECK (char_length(btrim(name)) BETWEEN 2 AND 80);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'users_password_hash_not_empty' AND conrelid = 'users'::regclass) THEN
        ALTER TABLE users ADD CONSTRAINT users_password_hash_not_empty
            CHECK (char_length(password_hash) > 0);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'users_nim_format' AND conrelid = 'users'::regclass) THEN
        ALTER TABLE users ADD CONSTRAINT users_nim_format
            CHECK (nim IS NULL OR nim ~ '^[0-9]{6,20}$');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'users_study_program_fk' AND conrelid = 'users'::regclass) THEN
        ALTER TABLE users ADD CONSTRAINT users_study_program_fk
            FOREIGN KEY (study_program_code) REFERENCES study_programs(code)
            ON UPDATE CASCADE ON DELETE RESTRICT;
    END IF;

    -- NOT VALID agar baris lama tanpa NIM tidak memblokir migrasi.
    -- Constraint tetap berlaku untuk INSERT/UPDATE baru.
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'users_mahasiswa_nim_required' AND conrelid = 'users'::regclass) THEN
        ALTER TABLE users ADD CONSTRAINT users_mahasiswa_nim_required
            CHECK (role <> 'mahasiswa' OR nim IS NOT NULL) NOT VALID;
    END IF;
END
$$;

-- Resolusi NIM -> prodi memakai kode terpanjang yang cocok.
-- Contoh: kode '02' dan '021' sama-sama ada, NIM 021... memilih '021'.
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

DROP TRIGGER IF EXISTS users_resolve_study_program ON users;
CREATE TRIGGER users_resolve_study_program
BEFORE INSERT OR UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION users_resolve_study_program();

DROP TRIGGER IF EXISTS study_programs_set_updated_at ON study_programs;
CREATE TRIGGER study_programs_set_updated_at
BEFORE UPDATE ON study_programs
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS users_set_updated_at ON users;
CREATE TRIGGER users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE UNIQUE INDEX IF NOT EXISTS users_email_lower_key ON users (lower(email));
CREATE UNIQUE INDEX IF NOT EXISTS users_nim_key ON users (nim) WHERE nim IS NOT NULL;
CREATE INDEX IF NOT EXISTS users_role_idx ON users (role) WHERE status = 'active';

COMMENT ON TABLE users IS 'Akun mahasiswa FT dan admin (admin_dev/admin_bem).';
COMMENT ON COLUMN users.password_hash IS 'Hash bcrypt; tidak pernah dikirim ke client.';
COMMENT ON COLUMN users.study_program_code IS 'Diisi otomatis oleh trigger dari prefix NIM.';

-- ---------------------------------------------------------------------------
-- user_contacts
-- Tabel terpisah agar nomor WhatsApp tidak ikut terbawa query SELECT users.*.
-- FR-04: hanya boleh dibaca endpoint detail transaksi berstatus approved.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS user_contacts (
    user_id       UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    whatsapp_e164 VARCHAR(20) NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT user_contacts_whatsapp_format CHECK (whatsapp_e164 ~ '^\+[1-9][0-9]{7,14}$')
);

DROP TRIGGER IF EXISTS user_contacts_set_updated_at ON user_contacts;
CREATE TRIGGER user_contacts_set_updated_at
BEFORE UPDATE ON user_contacts
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE user_contacts IS 'Kontak privat. Jangan pernah di-join ke endpoint publik/katalog (FR-04).';
COMMENT ON COLUMN user_contacts.whatsapp_e164 IS 'Format E.164, contoh +6281234567890.';
