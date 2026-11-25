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
    ('Mahasiswa Demo', 'mahasiswa@unesarent.local', crypt('Mahasiswa123!', gen_salt('bf', 12)), 'mahasiswa', '24051204999', 'active'),
    ('Penyewa Demo', 'mahasiswa2@unesarent.local', crypt('Mahasiswa123!', gen_salt('bf', 12)), 'mahasiswa', '24051204998', 'active')
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- Verifikasi KTM mahasiswa demo (dev only)
-- Trigger mewajibkan INSERT berstatus pending, jadi baris diverifikasi lewat
-- UPDATE. Idempotent: dilewati jika sudah ada pending/verified.
-- File KTM tidak ikut di-seed; endpoint file akan 404 untuk baris ini.
-- ---------------------------------------------------------------------------
INSERT INTO ktm_verifications (
    user_id, storage_key, mime_type, file_size_bytes, checksum_sha256, status, submitted_at
)
SELECT u.id,
       'ktm/' || u.id::text || '/00000000000000000000000000000000.jpg',
       'image/jpeg',
       1,
       repeat('0', 64),
       'pending',
       now() - interval '2 days'
FROM users u
WHERE lower(u.email) = 'mahasiswa@unesarent.local'
  AND NOT EXISTS (
      SELECT 1 FROM ktm_verifications k
       WHERE k.user_id = u.id AND k.status IN ('pending', 'verified')
  );

UPDATE ktm_verifications k
   SET status = 'verified',
       reviewed_by = (
           SELECT a.id FROM users a
            WHERE lower(a.email) = 'admin.dev@unesarent.local'
              AND a.role = 'admin_dev' AND a.status = 'active'
            LIMIT 1
       ),
       reviewed_at = now() - interval '1 day',
       reviewer_confirmed_nim = true
  FROM users u
 WHERE u.id = k.user_id
   AND lower(u.email) = 'mahasiswa@unesarent.local'
   AND k.status = 'pending';

-- Verifikasi KTM akun penyewa demo (dev only)
INSERT INTO ktm_verifications (
    user_id, storage_key, mime_type, file_size_bytes, checksum_sha256, status, submitted_at
)
SELECT u.id,
       'ktm/' || u.id::text || '/00000000000000000000000000000000.jpg',
       'image/jpeg',
       1,
       repeat('0', 64),
       'pending',
       now() - interval '2 days'
FROM users u
WHERE lower(u.email) = 'mahasiswa2@unesarent.local'
  AND NOT EXISTS (
      SELECT 1 FROM ktm_verifications k
       WHERE k.user_id = u.id AND k.status IN ('pending', 'verified')
  );

UPDATE ktm_verifications k
   SET status = 'verified',
       reviewed_by = (
           SELECT a.id FROM users a
            WHERE lower(a.email) = 'admin.dev@unesarent.local'
              AND a.role = 'admin_dev' AND a.status = 'active'
            LIMIT 1
       ),
       reviewed_at = now() - interval '1 day',
       reviewer_confirmed_nim = true
  FROM users u
 WHERE u.id = k.user_id
   AND lower(u.email) = 'mahasiswa2@unesarent.local'
   AND k.status = 'pending';

-- ---------------------------------------------------------------------------
-- Listing contoh milik mahasiswa demo (dev only, tanpa foto)
-- UUID tetap agar idempotent. Foto tidak di-seed karena butuh file fisik.
-- ---------------------------------------------------------------------------
INSERT INTO listings (id, owner_id, category_id, campus_location_id, title, description, price_per_day, condition_percent)
SELECT l.id, u.id, c.id, cl.id, l.title, l.description, l.price_per_day, l.condition_percent
FROM (VALUES
    ('11111111-1111-4111-8111-111111111111'::uuid, 'elektronik', 'Gedung A FT',
     'Kamera Canon EOS 700D',
     'Kamera DSLR dengan lensa kit 18-55mm. Termasuk tas dan charger. Kondisi terawat, siap dipakai untuk tugas atau dokumentasi acara.',
     75000, 90),
    ('22222222-2222-4222-8222-222222222222'::uuid, 'elektronik', 'Gedung B FT',
     'Proyektor Mini 1080p',
     'Proyektor portabel 1080p dengan HDMI dan USB. Cocok untuk presentasi kelompok atau nobar.',
     50000, 85),
    ('33333333-3333-4333-8333-333333333333'::uuid, 'olahraga', 'Area Parkir FT',
     'Tenda Camping 4 Orang',
     'Tenda kapasitas 4 orang, sudah termasuk pasak dan flysheet. Bersih dan tidak bocor.',
     40000, 80),
    ('44444444-4444-4444-8444-444444444444'::uuid, 'perkakas', 'Lab Terpadu FT',
     'Bor Listrik 650W',
     'Bor listrik 650W dengan set mata bor kayu dan besi. Hanya untuk pemakaian wajar.',
     25000, 95)
) AS l(id, category_slug, location_name, title, description, price_per_day, condition_percent)
JOIN users u ON lower(u.email) = 'mahasiswa@unesarent.local'
JOIN categories c ON c.slug = l.category_slug
JOIN campus_locations cl ON cl.name = l.location_name
ON CONFLICT (id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Rental contoh (dev only)
-- Penyewa demo menyewa kamera milik mahasiswa demo.
-- Tanggal memakai current_date agar selalu valid saat seed dijalankan ulang.
-- ---------------------------------------------------------------------------
INSERT INTO rentals (id, listing_id, renter_id, start_date, end_date)
SELECT r.id, r.listing_id, u.id, current_date + r.start_offset, current_date + r.end_offset
FROM (VALUES
    ('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid,
     '11111111-1111-4111-8111-111111111111'::uuid, 2, 4),
    ('bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'::uuid,
     '33333333-3333-4333-8333-333333333333'::uuid, 7, 8)
) AS r(id, listing_id, start_offset, end_offset)
JOIN users u ON lower(u.email) = 'mahasiswa2@unesarent.local'
ON CONFLICT (id) DO NOTHING;

-- Satu rental sudah disetujui agar chat dan PDF perjanjian bisa diuji.
UPDATE rentals
   SET status = 'approved'
 WHERE id = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'
   AND status = 'pending';

-- Pesan chat contoh pada rental approved (butuh migration 0009).
INSERT INTO rental_messages (id, rental_id, sender_id, body)
SELECT m.id, m.rental_id, u.id, m.body
FROM (VALUES
    ('cccccccc-cccc-4ccc-8ccc-cccccccccccc'::uuid,
     'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid,
     'mahasiswa@unesarent.local',
     'Halo, kamera bisa diambil setelah jam 4 sore di Gedung A.'),
    ('dddddddd-dddd-4ddd-8ddd-dddddddddddd'::uuid,
     'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'::uuid,
     'mahasiswa2@unesarent.local',
     'Siap, saya ke Gedung A jam 4 sore. Terima kasih.')
) AS m(id, rental_id, sender_email, body)
JOIN users u ON lower(u.email) = m.sender_email
JOIN rentals r ON r.id = m.rental_id
ON CONFLICT (id) DO NOTHING;

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
