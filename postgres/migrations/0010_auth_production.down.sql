DROP TRIGGER IF EXISTS users_block_verified_nim_change ON users;
DROP FUNCTION IF EXISTS users_block_verified_nim_change();
DROP TABLE IF EXISTS auth_otps;
ALTER TABLE users DROP COLUMN IF EXISTS last_login_at;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified_at;
