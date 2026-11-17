-- 0007_single_admin.up.sql
-- Menyatukan admin_dev dan admin_bem menjadi satu role admin_dev.
--
-- Tim tidak memiliki operator BEM, sehingga role admin_bem tidak lagi dipakai.
-- Semua akun admin_bem yang sudah ada dikonversi agar tidak kehilangan akses.
-- token_version dinaikkan supaya sesi lama (JWT) harus login ulang.

UPDATE users
   SET role = 'admin_dev',
       token_version = token_version + 1
 WHERE role = 'admin_bem';

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('mahasiswa', 'admin_dev'));

COMMENT ON TABLE users IS 'Akun mahasiswa FT dan admin (admin_dev).';
