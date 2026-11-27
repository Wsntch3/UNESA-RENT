-- seed/reference.sql
-- Seed referensi produksi. HANYA data acuan, tanpa akun/listing/rental demo.
-- Aman dijalankan berulang (idempotent).

-- ---------------------------------------------------------------------------
-- Program studi FT
-- Hanya kode yang sudah diketahui. Jangan menebak kode prodi lain karena
-- trigger NIM memakainya untuk validasi.
-- ---------------------------------------------------------------------------
INSERT INTO study_programs (code, name, faculty, faculty_code) VALUES
    ('120', 'S1 Teknik Informatika', 'Fakultas Teknik', '05')
ON CONFLICT (code) DO UPDATE
    SET name = EXCLUDED.name,
        faculty = EXCLUDED.faculty,
        faculty_code = EXCLUDED.faculty_code;

-- ---------------------------------------------------------------------------
-- Kategori barang
-- ---------------------------------------------------------------------------
INSERT INTO categories (slug, name) VALUES
    ('elektronik',  'Elektronik'),
    ('alat-tulis',  'Alat Tulis'),
    ('olahraga',    'Olahraga'),
    ('fotografi',   'Fotografi'),
    ('perkakas',    'Perkakas'),
    ('lainnya',     'Lainnya')
ON CONFLICT (slug) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Lokasi serah-terima di area Fakultas Teknik
-- ---------------------------------------------------------------------------
INSERT INTO campus_locations (name, description) VALUES
    ('Gedung A FT',    'Lobby Gedung A Fakultas Teknik'),
    ('Gedung B FT',    'Lobby Gedung B Fakultas Teknik'),
    ('Gedung C FT',    'Lobby Gedung C Fakultas Teknik'),
    ('Lab Terpadu FT', 'Area Lab Terpadu Fakultas Teknik'),
    ('Area Parkir FT', 'Area parkir Fakultas Teknik')
ON CONFLICT (name) DO NOTHING;
