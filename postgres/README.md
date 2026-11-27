<div align="center">

# UNESARENT PostgreSQL

Schema dan migration PostgreSQL 17 untuk UNESARENT.

[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-17-4169E1?style=flat-square&logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=flat-square&logo=docker&logoColor=white)](https://docs.docker.com/compose/)
[![migrate](https://img.shields.io/badge/migration-golang--migrate-00ADD8?style=flat-square)](https://github.com/golang-migrate/migrate)

[Dokumentasi utama](../README.md) · [Schema](#schema) · [Migration](#migration) · [Integrasi backend](#integrasi-backend)

</div>

---

> **Status:** schema database lengkap untuk MVP Listing-First, termasuk verifikasi KTM
> (`0005`), hardening (`0006`), role admin tunggal (`0007`), governance template
> surat (`0008`), dan chat rental (`0009`). Backend sudah memakai PostgreSQL (`pgx`) saat `DATABASE_URL` diisi
> dan otomatis fallback ke file JSON jika tidak.

---

## Ringkasan

Folder `postgres/` berisi:

- PostgreSQL 17 lewat Docker Compose dengan healthcheck dan named volume
- Extension `btree_gist`, `pg_trgm`, dan `pgcrypto`
- Migration versioned (`golang-migrate`) dengan up/down
- Schema untuk identitas, katalog, transaksi sewa, surat pernyataan, dan governance
- Seed referensi produksi (`seed/reference.sql`) dan seed demo lokal (`seed/dev_seed.sql`)
- Constraint database untuk aturan bisnis yang tidak boleh dilanggar aplikasi

Schema dijalankan otomatis saat container dibuat pertama kali, lalu migration
dijalankan oleh service `migrate`.

---

## Struktur

```text
postgres/
├── initdb/
│   └── 000_extensions.sql       # hanya extension; dijalankan saat volume kosong
├── migrations/
│   ├── 0001_identity.up.sql
│   ├── 0001_identity.down.sql
│   ├── 0002_catalog.up.sql
│   ├── 0002_catalog.down.sql
│   ├── 0003_rentals.up.sql
│   ├── 0003_rentals.down.sql
│   ├── 0004_governance.up.sql
│   ├── 0004_governance.down.sql
│   ├── 0005_ktm_verification.up.sql
│   ├── 0005_ktm_verification.down.sql
│   ├── 0006_hardening.up.sql
│   ├── 0006_hardening.down.sql
│   ├── 0007_single_admin.up.sql
│   ├── 0007_single_admin.down.sql
│   ├── 0008_letter_governance.up.sql
│   ├── 0008_letter_governance.down.sql
│   ├── 0009_rental_chat.up.sql
│   └── 0009_rental_chat.down.sql
├── seed/
│   ├── reference.sql            # produksi: prodi, kategori, lokasi
│   └── dev_seed.sql             # HANYA lokal: akun/listing/rental demo
├── .env.example
├── docker-compose.yml
└── README.md
```

| File | Deskripsi |
| --- | --- |
| `initdb/000_extensions.sql` | Extension yang dibutuhkan schema |
| `migrations/*.up.sql` | Perubahan schema ke depan |
| `migrations/*.down.sql` | Rollback migration |
| `seed/reference.sql` | Data referensi produksi (idempotent) |
| `seed/dev_seed.sql` | Data development lengkap; hanya untuk lokal (idempotent) |
| `docker-compose.yml` | PostgreSQL, healthcheck, volume, dan service migrate |

---

## Menjalankan dengan Docker

Prasyarat: Docker dan Docker Compose.

```bash
cd postgres
cp .env.example .env
docker compose up -d
```

Urutan yang terjadi:

1. Container `postgres` dibuat dan `initdb/000_extensions.sql` dijalankan.
2. Healthcheck menunggu PostgreSQL siap.
3. Service `migrate` menjalankan seluruh migration `up`.
4. Container `migrate` selesai dengan exit code `0`.

Dari root proyek:

```bash
make db-up       # docker compose up -d (sekaligus migration)
make db-seed         # seed referensi produksi (reference.sql)
make db-seed-dev     # seed data demo; hanya untuk lokal
make db-down     # hentikan container, data tetap ada
make db-reset    # hapus volume + buat ulang + migration
```

Cek status:

```bash
docker compose ps -a
docker compose logs migrate
```

Seed development:

```bash
make db-seed
```

Atau manual:

```bash
cd postgres
docker compose exec -T postgres sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1' < seed/dev_seed.sql
```

---

## Menjalankan dengan PostgreSQL Lokal

Jika memakai PostgreSQL lokal (bukan Docker):

```bash
createdb unesarent
psql -d unesarent -f initdb/000_extensions.sql

migrate -path migrations \
  -database "postgres://unesarent:unesarent_dev@localhost:5432/unesarent?sslmode=disable" up

psql -d unesarent -v ON_ERROR_STOP=1 -f seed/dev_seed.sql
```

Pastikan user database punya izin membuat extension, tabel, index, function, dan trigger.
Jika tidak ingin memasang CLI `migrate`, gunakan service Compose `migrate` dengan
mengarahkan `DATABASE_URL` ke PostgreSQL lokal, atau jalankan migration dari container
migrate sesuai kebutuhan environment.

---

## Migration

Migration memakai [golang-migrate](https://github.com/golang-migrate/migrate).
File diberi nomor urut dan wajib punya pasangan `up`/`down`.

```bash
# dari root proyek
make db-migrate        # jalankan semua migration yang belum diterapkan
make db-migrate-down   # rollback satu migration
```

Manual dari folder `postgres/`:

```bash
docker compose run --rm migrate
docker compose run --rm migrate -path=/migrations -database="postgres://unesarent:unesarent_dev@postgres:5432/unesarent?sslmode=disable" down 1
```

Menambah migration baru:

```bash
cd postgres
docker run --rm -v "${PWD}/migrations:/migrations" migrate/migrate \
  create -ext sql -dir /migrations -seq -digits 4 nama_perubahan
```

> Volume `migrations/` di service Compose sengaja di-mount read-only. Gunakan
> perintah `docker run` di atas (mount read-write) hanya saat membuat file migration baru.

Aturan:

- Jangan mengubah migration yang sudah pernah diterapkan di environment bersama.
- Tambahkan migration baru untuk setiap perubahan schema.
- Selalu isi file `down` agar rollback tetap berfungsi.
- `initdb/` hanya untuk bootstrap extension, bukan tempat schema.

### Volume lama

Volume yang dibuat sebelum migration dipakai tetap bisa dimigrasikan: `0001_identity`
memakai `IF NOT EXISTS` dan `DO` block untuk menambahkan kolom/constraint yang belum ada.
Jika volume lama bermasalah, reset sekali:

```bash
make db-reset
```

---

## Schema

### Identitas

| Tabel | Kolom penting | Keterangan |
| --- | --- | --- |
| `study_programs` | `code` (3 digit), `faculty_code` (2 digit), `name`, `faculty` | Daftar prodi FT; sumber validasi NIM |
| `users` | `role`, `nim`, `study_program_code`, `status` | `mahasiswa`, `admin_dev` |
| `user_contacts` | `user_id`, `whatsapp_e164` | Kontak privat, terpisah dari `users` (FR-04) |
| `ktm_verifications` | `user_id`, `storage_key`, `status`, `reviewed_by` | Riwayat verifikasi KTM; satu `pending` per user |

Format NIM UNESA adalah `AA FF PPP J UUU` (11 digit): `AA` angkatan, `FF` kode
fakultas (Fakultas Teknik = `05`), `PPP` kode prodi (contoh S1 Teknik Informatika =
`120`), `J` jenjang, dan `UUU` nomor urut. `users.study_program_code` diisi otomatis
oleh trigger `users_resolve_study_program` dari `FF` dan `PPP`; NIM di luar
fakultas/prodi yang terdaftar ditolak dengan error database.

`ktm_verifications` menyimpan riwayat submission, bukan kolom status di `users`:

- Status: `pending`, `verified`, `rejected`, `revoked`.
- Hanya satu baris `pending` per user (unique index parsial).
- `verified`/`rejected`/`revoked` wajib punya `reviewed_by` dan `reviewed_at`.
- `reviewed_by`/`revoked_by` harus user dengan role `admin_dev` yang aktif (trigger).
- `rejected` wajib punya `rejection_reason`; `revoked` wajib punya `revoke_reason`.
- `verified` wajib `reviewer_confirmed_nim = true`.
- File KTM hanya disimpan sebagai `storage_key`; tidak ada BLOB di database.
- Di development file berada di `KTM_STORAGE_DIR` (`backend/data/ktm`) dan hanya boleh
  diakses pemilik atau `admin_dev` lewat `GET /api/ktm/{id}/file`. Untuk produksi,
  implementasi `ktm.Storage` dapat diganti ke Supabase Storage tanpa mengubah skema.
- Menghapus baris `ktm_verifications`/user tidak menghapus file fisik; pembersihan file
  yatim masih manual sampai ada job housekeeping.
- Status efektif dihitung oleh `is_ktm_verified()`: KTM `verified`, `nim_snapshot`
  masih sama dengan `users.nim`, dan akun masih `active`.

### Katalog

| Tabel | Kolom penting | Keterangan |
| --- | --- | --- |
| `categories` | `slug`, `name` | Kategori barang |
| `campus_locations` | `name` | Titik serah-terima area FT |
| `listings` | `price_per_day`, `condition_percent`, `is_active` | Harga integer, tanpa diskon |
| `listing_photos` | `listing_id`, `storage_key`, `position` | Maksimal 3 foto per listing |
| `listing_blocks` | `listing_id`, `period` | Tanggal tidak tersedia dari penyedia |

`listings.is_active = false` adalah mekanisme takedown (FR-06). Takedown wajib
menyertakan `takedown_by`, `takedown_reason`, dan `takedown_at`.

### Transaksi

| Tabel | Kolom penting | Keterangan |
| --- | --- | --- |
| `letter_templates` | `version`, `body`, `is_active`, `approval_reference` | Template surat; maksimal satu aktif |
| `rentals` | `start_date`, `end_date`, `total_days`, `total_price`, `status` | Pengajuan/transaksi sewa |
| `rental_events` | `from_status`, `to_status`, `actor_id` | Timeline untuk sengketa |
| `rental_messages` | `rental_id`, `sender_id`, `body`, `created_at` | Chat internal per rental, append-only |
| `security_letters` | `rental_id`, `template_id`, `pdf_storage_key` | Metadata PDF Surat Pernyataan (belum dipakai backend) |

Status rental: `pending`, `approved`, `rejected`, `cancelled`, `completed`.
Pesan chat dibatasi 1-2000 karakter dan hanya dapat dikirim saat rental
`pending`/`approved`; aturan status ditegakkan backend, bukan trigger.

- `end_date` **inklusif**. `total_days = end_date - start_date + 1`.
- `total_price = total_days * price_per_day_snapshot` dihitung database.
- Harga disimpan sebagai snapshot saat pengajuan, sehingga perubahan harga listing
  tidak mengubah transaksi yang sudah berjalan.
- Transisi status dibatasi trigger: `pending -> approved/rejected/cancelled`,
  `approved -> completed/cancelled`.

### Governance

| Tabel | Kolom penting | Keterangan |
| --- | --- | --- |
| `reports` | `type`, `status`, `assigned_to` | Pengaduan `technical` / `safety`; keduanya ditangani `admin_dev` sampai role BEM tersedia |
| `moderation_actions` | `actor_id`, `action`, `target_type`, `target_id` | Audit trail admin, append-only |
| `notifications` | `user_id`, `type`, `read_at` | Notifikasi in-app |

---

## ERD

```mermaid
erDiagram
    study_programs ||--o{ users : "kode prodi"
    users ||--o| user_contacts : "kontak privat"
    users ||--o{ ktm_verifications : "submission KTM"
    users ||--o{ ktm_verifications : "reviewer admin_dev"
    users ||--o{ listings : "owner"
    categories ||--o{ listings : "kategori"
    campus_locations ||--o{ listings : "lokasi"
    listings ||--o{ listing_photos : "foto"
    listings ||--o{ listing_blocks : "blokir"
    listings ||--o{ rentals : "disewa"
    users ||--o{ rentals : "penyewa"
    users ||--o{ rentals : "penyedia"
    rentals ||--o{ rental_events : "timeline"
    rentals ||--o| security_letters : "surat"
    letter_templates ||--o{ security_letters : "template"
    users ||--o{ reports : "pengadu"
    users ||--o{ moderation_actions : "aktor"
    users ||--o{ notifications : "penerima"
```

---

## Aturan integritas di database

| Kebutuhan | Implementasi |
| --- | --- |
| FR-01 NIM FT | `study_programs` (`faculty_code` + `code`) + trigger NIM + `CHECK` format 11 digit |
| Verifikasi KTM | `ktm_verifications`: satu `pending` per user, reviewer wajib `admin_dev` aktif, `rejected` wajib alasan, `is_ktm_verified()` sebagai sumber tunggal |
| FR-02 harga pas | `price_per_day INTEGER CHECK > 0`; tidak ada kolom diskon/cicilan |
| FR-03 double-booking | `EXCLUDE USING gist` pada `rentals` untuk status `approved` |
| FR-03 tanggal blokir | `listing_blocks` dengan exclusion constraint + trigger `rentals_before_write` |
| FR-04 privasi WA | tabel terpisah `user_contacts`; tidak pernah di-join ke katalog |
| FR-05 surat PDF | `letter_templates` (satu aktif, wajib `approval_reference`, immutable setelah aktif) + `security_letters` (satu per rental, hanya dari template aktif) |
| FR-06 multi-admin | `users.role` + `listings.is_active` + `moderation_actions` |
| NFR-01 pencarian | partial index katalog + GIN trigram pada `listings.title` |
| NFR-03 file | database hanya menyimpan `storage_key` dan metadata |
| BR-03 tanpa payment | **tidak ada tabel payment/escrow** (lihat marker di `0003_rentals.up.sql`) |

## Seed development

`seed/dev_seed.sql` mengisi:

- Kode fakultas FT `05` dan prodi `120` (S1 Teknik Informatika)
- Kategori: elektronik, alat tulis, olahraga, fotografi, perkakas, lainnya
- Lokasi serah-terima area FT
- Dua akun admin development (keduanya `admin_dev`):
  - `admin.dev@unesarent.local` / `AdminDev123!`
  - `admin.ops@unesarent.local` / `AdminOps123!`
- Akun mahasiswa demo:
  - `mahasiswa@unesarent.local` / `Mahasiswa123!` (NIM sintetis `24051204999`)
  - `mahasiswa2@unesarent.local` / `Mahasiswa123!` (NIM sintetis `24051204998`)
- Verifikasi KTM untuk kedua mahasiswa demo, sehingga langsung bisa membuat listing
  dan mengajukan rental (baris `ktm_verifications` tanpa file fisik; endpoint file
  KTM akan 404)
- Empat listing contoh milik mahasiswa demo (tanpa foto) dengan UUID tetap agar seed
  idempotent
- Dua rental milik `mahasiswa2`: satu `approved` (kamera) dan satu `pending` (tenda)
- Dua pesan chat contoh pada rental `approved`
- Template Surat Pernyataan Keamanan `v1` dengan teks "Template resmi UNESA Rent"

## Koneksi

Nilai default development:

```text
postgres://unesarent:unesarent_dev@localhost:5432/unesarent?sslmode=disable
```

| Parameter | Default |
| --- | --- |
| Host | `localhost` |
| Port | `5432` (lihat catatan port di bawah) |
| Database | `unesarent` |
| User | `unesarent` |
| Password | `unesarent_dev` |
| SSL mode | `disable` |

Ganti password sebelum dipakai di luar development.

> **Port sudah dipakai?** Jika `5432` sudah dipakai PostgreSQL lain di mesin kamu,
> set `POSTGRES_PORT=5433` di `postgres/.env` (file ini tidak di-commit), jalankan
> `docker compose up -d`, lalu sesuaikan port pada `DATABASE_URL` backend.
> Migration di dalam container tetap memakai port `5432`.


## Troubleshooting

| Masalah | Solusi |
| --- | --- |
| Port 5432 sudah dipakai | Set `POSTGRES_PORT=5433` di `postgres/.env`, jalankan `docker compose up -d`, sesuaikan `DATABASE_URL` |
| Migration gagal di volume lama | Cek `docker compose logs migrate`; reset dengan `make db-reset` |
| Registrasi mahasiswa selalu ditolak | Seed prodi belum dijalankan atau kode prodi masih placeholder |
| `schema_migrations` tidak ada | Migration belum pernah dijalankan; `make db-migrate` |
| Perubahan SQL tidak diterapkan | Init script hanya jalan pada volume kosong; gunakan migration |
| Data hilang setelah restart | Jangan pakai `down -v`; named volume menyimpan data |
| Password authentication failed | Samakan kredensial `.env` dengan `DATABASE_URL` |