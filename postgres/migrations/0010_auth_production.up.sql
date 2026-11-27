-- 0010_auth_production.up.sql
-- OTP email, verifikasi email, dan penguncian NIM setelah KTM verified.

ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;

-- Akun lama (dev) dianggap sudah verifikasi email agar tidak terkunci.
UPDATE users SET email_verified_at = created_at WHERE email_verified_at IS NULL;

CREATE TABLE IF NOT EXISTS auth_otps (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email          VARCHAR(254) NOT NULL,
    user_id        UUID REFERENCES users(id) ON DELETE CASCADE,
    purpose        TEXT NOT NULL,
    code_hash      CHAR(64) NOT NULL,
    challenge_hash CHAR(64),
    attempts       SMALLINT NOT NULL DEFAULT 0,
    max_attempts   SMALLINT NOT NULL DEFAULT 5,
    expires_at     TIMESTAMPTZ NOT NULL,
    verified_at    TIMESTAMPTZ,
    consumed_at    TIMESTAMPTZ,
    request_ip     VARCHAR(45),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT auth_otps_purpose_check CHECK (
        purpose IN ('register', 'reset_password', 'admin_login')
    ),
    CONSTRAINT auth_otps_attempts_check CHECK (
        attempts >= 0 AND max_attempts BETWEEN 1 AND 10
    ),
    CONSTRAINT auth_otps_email_format CHECK (
        email ~* '^[^@[:space:]]+@[^@[:space:]]+\.[^@[:space:]]+$'
    )
);

CREATE INDEX IF NOT EXISTS auth_otps_lookup_idx
    ON auth_otps (lower(email), purpose, created_at DESC);
CREATE INDEX IF NOT EXISTS auth_otps_expiry_idx
    ON auth_otps (expires_at) WHERE consumed_at IS NULL;

-- NIM tidak boleh diubah selama masih ada KTM verified.
CREATE OR REPLACE FUNCTION users_block_verified_nim_change()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.nim IS DISTINCT FROM NEW.nim AND EXISTS (
        SELECT 1 FROM ktm_verifications k
        WHERE k.user_id = OLD.id AND k.status = 'verified'
    ) THEN
        RAISE EXCEPTION 'NIM tidak boleh diubah setelah KTM terverifikasi; cabut verifikasi terlebih dahulu'
            USING ERRCODE = '23514', CONSTRAINT = 'users_nim_locked_after_verification';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS users_block_verified_nim_change ON users;
CREATE TRIGGER users_block_verified_nim_change
BEFORE UPDATE OF nim ON users
FOR EACH ROW EXECUTE FUNCTION users_block_verified_nim_change();

COMMENT ON TABLE auth_otps IS 'OTP email sekali pakai untuk register, reset password, dan login admin.';
COMMENT ON COLUMN auth_otps.code_hash IS 'HMAC-SHA256(kode, OTP_PEPPER); bukan plaintext.';
COMMENT ON COLUMN auth_otps.challenge_hash IS 'SHA-256 token lanjutan; dikirim ke client sekali saja.';
