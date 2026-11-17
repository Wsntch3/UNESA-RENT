-- seed/dev_seed.sql
-- Seed development. Aman dijalankan berulang (idempotent).
--
-- Format NIM UNESA: AA FF PPP J UUU (11 digit).
--   AA  = angkatan
--   FF  = kode fakultas (Fakultas Teknik = 05)
--   PPP = kode program studi (contoh S1 Teknik Informatika = 120)
--   J   = jenjang
--   UUU = nomor urut
--
-- Contoh: 24051204100 -> 24 | 05 | 120 | 4 | 100

-- ---------------------------------------------------------------------------
-- Program studi FT
-- Baru memuat kode yang diketahui. Tambahkan prodi FT lain setelah daftar resmi
-- diperoleh; jangan menebak kode karena trigger memakainya untuk validasi NIM.
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

-- ---------------------------------------------------------------------------
-- Akun admin development
-- Dua akun admin_dev agar tidak lockout. Tidak ada role admin_bem.
-- Password: AdminDev123! / AdminOps123! (ganti sebelum dipakai di luar dev)
-- ---------------------------------------------------------------------------
INSERT INTO users (name, email, password_hash, role, status) VALUES
    ('Developer UNESA Rent', 'admin.dev@unesarent.local', crypt('AdminDev123!', gen_salt('bf', 12)), 'admin_dev', 'active'),
    ('Operator UNESA Rent',  'admin.ops@unesarent.local', crypt('AdminOps123!', gen_salt('bf', 12)), 'admin_dev', 'active')
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- Akun mahasiswa development
-- NIM demo sintetis: 24 | 05 | 120 | 4 | 999.
-- NIM asli dengan format yang sama (mis. 24051204100) juga akan diterima.
-- Password: Mahasiswa123!
-- ---------------------------------------------------------------------------
INSERT INTO users (name, email, password_hash, role, nim, status) VALUES
    ('Mahasiswa Demo', 'mahasiswa@unesarent.local', crypt('Mahasiswa123!', gen_salt('bf', 12)), 'mahasiswa', '24051204999', 'active')
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- Template Surat Pernyataan Keamanan
--
-- Naskah default TIDAK mengklaim pengesahan BEM karena belum ada referensi
-- persetujuan offline. Setelah approval diperoleh:
--   1. Buat versi baru (jangan ubah baris yang sudah aktif).
--   2. Isi approval_reference, approved_by_name, approved_offline_at.
--   3. Sesuaikan body dengan otoritas yang benar-benar menyetujui.
-- ---------------------------------------------------------------------------
INSERT INTO letter_templates (
    version, title, body, is_active, created_by,
    approval_reference, approved_by_name, approved_offline_at
)
SELECT
    'v1',
    'Surat Pernyataan Keamanan',
    E'SURAT PERNYATAAN KEAMANAN\n'
    'Nomor: {{nomor_surat}}\n\n'
    'Pengelola UNESA Rent menerangkan bahwa transaksi sewa berikut telah dicatat oleh sistem UNESA Rent:\n\n'
    'Penyedia : {{nama_penyedia}}\n'
    'Penyewa  : {{nama_penyewa}}\n'
    'Barang   : {{nama_barang}}\n'
    'Rentang  : {{tanggal_mulai}} s.d. {{tanggal_selesai}}\n'
    'Durasi   : {{total_hari}} hari\n'
    'Nilai    : Rp{{total_harga}}\n\n'
    'Surat ini dihasilkan otomatis dari template resmi UNESA Rent dan bukan merupakan jaminan asuransi atau ganti rugi. Sengketa kehilangan/kerusakan diselesaikan secara langsung oleh para pihak sesuai ketentuan kampus.\n\n'
    'Diterbitkan: {{tanggal_terbit}}',
    true,
    u.id,
    'DEV - belum ada referensi persetujuan offline',
    NULL,
    NULL
FROM users u
WHERE lower(u.email) = 'admin.dev@unesarent.local'
ON CONFLICT (version) DO NOTHING;
