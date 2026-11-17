-- 0007_single_admin.down.sql
-- Mengembalikan role admin_bem ke constraint.
-- Catatan: akun yang sudah dikonversi ke admin_dev tidak bisa dibedakan lagi,
-- jadi rollback ini tidak mengembalikan role data.

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('mahasiswa', 'admin_dev', 'admin_bem'));

COMMENT ON TABLE users IS 'Akun mahasiswa FT dan admin (admin_dev/admin_bem).';
