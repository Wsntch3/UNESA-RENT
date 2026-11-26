<div align="center">

# UNESARENT Backend

API autentikasi berbasis Go dengan JWT, bcrypt, dan penyimpanan user yang dapat diganti.

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![JWT](https://img.shields.io/badge/Auth-JWT%20HS256-000000?style=flat-square&logo=jsonwebtokens&logoColor=white)](https://jwt.io/)
[![bcrypt](https://img.shields.io/badge/Password-bcrypt-4B8BBE?style=flat-square)](https://pkg.go.dev/golang.org/x/crypto/bcrypt)

[Dokumentasi utama](../README.md) · [API](#api) · [Konfigurasi](#konfigurasi)

</div>

---

## Ringkasan

Backend UNESARENT menyediakan register, login, dan endpoint profil menggunakan access
token JWT. Seluruh routing memakai standard library `net/http` tanpa framework tambahan.
Password disimpan sebagai hash bcrypt. User disimpan di PostgreSQL (`pgx`) jika
`DATABASE_URL` diisi, atau di file JSON untuk development tanpa database.

Repositori ini adalah bagian dari monorepo [UNESARENT](../README.md).

---

## Fitur

- Register dan login dengan validasi input
- Registrasi mahasiswa dengan NIM; prodi diturunkan otomatis dari digit fakultas/prodi NIM
- Access token JWT HS256 dengan claim `role` (`mahasiswa`, `admin_dev`); role/status
  terbaru selalu dibaca ulang dari database
- Middleware `RequireAuth` dan `RequireRole`
- Verifikasi signature, issuer, dan masa berlaku token
- Password bcrypt
- Middleware Bearer token
- CORS untuk origin frontend yang dikonfigurasi
- Request logging dan panic recovery
- Body limit 1 MB
- Graceful shutdown
- Penyimpanan PostgreSQL (`pgx`) atau JSON atomik dengan permission `0600`
- Interface `user.Store` untuk memisahkan business logic dari penyimpanan
- Katalog publik: kategori, lokasi, pencarian judul, filter kategori/harga/lokasi, sort
- Pengelolaan listing penyedia: create/update/delete, maks 3 foto, blokir tanggal
- Moderasi admin: takedown/restore listing dengan audit `moderation_actions`
- Pengajuan sewa, approve/reject/cancel/complete, availability, dan notifikasi in-app
- Chat rental via WebSocket (riwayat lewat REST) dengan tiket sekali pakai
- PDF Surat Pernyataan Keamanan statis untuk dibaca peserta rental

---

## Teknologi

| Komponen    | Implementasi                          |
| ----------- | ------------------------------------- |
| Bahasa      | Go 1.26+                              |
| HTTP        | `net/http`                            |
| JWT         | `github.com/golang-jwt/jwt/v5`        |
| Password    | `golang.org/x/crypto/bcrypt`          |
| Database    | PostgreSQL 17 + `github.com/jackc/pgx/v5` |
| WebSocket   | `github.com/coder/websocket`          |
| Logging     | `log/slog`                            |
| Storage     | PostgreSQL, fallback file JSON        |

---

## Struktur

```text
backend/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── auth/
│   │   ├── jwt.go
│   │   └── password.go
│   ├── config/
│   │   ├── config.go
│   │   └── env.go
│   ├── db/
│   │   └── db.go
│   ├── httpapi/
│   │   ├── auth_handler.go
│   │   ├── ktm_handler.go
│   │   ├── listing_handler.go
│   │   ├── rental_handler.go
│   │   ├── rental_chat_hub.go
│   │   ├── middleware.go
│   │   ├── respond.go
│   │   └── router.go
│   ├── catalog/
│   │   ├── model.go
│   │   ├── service.go
│   │   ├── storage.go
│   │   ├── store.go
│   │   └── store_postgres.go
│   ├── rental/
│   │   ├── model.go
│   │   ├── service.go
│   │   ├── store.go
│   │   └── store_postgres.go
│   ├── ktm/
│   │   ├── model.go
│   │   ├── service.go
│   │   ├── storage.go
│   │   ├── store.go
│   │   └── store_postgres.go
│   └── user/
│       ├── model.go
│       ├── service.go
│       ├── store.go
│       └── store_postgres.go
├── .env.example
└── go.mod
```

| Package              | Tanggung jawab                                        |
| -------------------- | ----------------------------------------------------- |
| `cmd/server`         | Bootstrap aplikasi, pilihan storage, HTTP, shutdown   |
| `internal/config`    | Pembacaan environment dan file `.env`                 |
| `internal/db`        | Connection pool PostgreSQL (`pgxpool`)                |
| `internal/auth`      | Penerbitan/verifikasi JWT dan hashing password        |
| `internal/httpapi`   | Router, handler, middleware, dan format response      |
| `internal/ktm`       | Aturan verifikasi KTM, storage file, dan akses file   |
| `internal/catalog`   | Listing, foto, blokir tanggal, katalog, dan moderasi  |
| `internal/rental`    | Rental, transisi status, notifikasi, chat, perjanjian |
| `internal/user`      | Model, validasi, business logic, dan penyimpanan user |

---

## Menjalankan

Prasyarat: Go 1.26+ dan, jika memakai database, PostgreSQL 17 yang sudah dimigrasikan
(lihat [postgres/README.md](../postgres/README.md)).

```bash
cd backend
cp .env.example .env
go run ./cmd/server
```

Ganti `JWT_SECRET` di `.env` dengan string acak minimal 32 karakter:

```bash
openssl rand -base64 48
```

Server berjalan di `http://localhost:8080`.

### Dengan PostgreSQL

1. Jalankan PostgreSQL dan migration: `make db-up` dari root proyek (atau
   `docker compose up -d` dari `postgres/`).
2. Isi `DATABASE_URL` di `backend/.env`, contoh:

   ```text
   DATABASE_URL=postgres://unesarent:unesarent_dev@localhost:5432/unesarent?sslmode=disable
   ```

3. Jalankan server. Log akan menampilkan `penyimpanan user: PostgreSQL`.

Jika `DATABASE_URL` kosong, backend otomatis memakai `STORE_PATH` (file JSON). Mode ini
cukup untuk mencoba auth tanpa database, tetapi tidak punya validasi NIM/prodi dan role
dari database.

> Jika port `5432` sudah dipakai PostgreSQL lain di mesin kamu, set
> `POSTGRES_PORT=5433` di `postgres/.env` lalu sesuaikan port pada `DATABASE_URL`.

Health check:

```bash
curl http://localhost:8080/api/health
```

Response:

```json
{
  "status": "ok"
}
```

---

## Konfigurasi

Backend membaca environment variable dan file `.env` di direktori kerja. Environment
variable yang sudah di-set tidak ditimpa oleh `.env`.

| Variable          | Default                     | Wajib | Deskripsi                            |
| ----------------- | --------------------------- | ----- | ------------------------------------ |
| `APP_ADDR`        | `:8080`                     | Tidak | Alamat listen HTTP server            |
| `JWT_SECRET`      | secret development          | Ya    | Secret JWT, minimal 32 karakter      |
| `JWT_TTL`         | `24h`                       | Tidak | Masa berlaku access token            |
| `CORS_ORIGINS`    | `http://localhost:5173,...` | Tidak | Origin frontend, dipisahkan koma     |
| `DATABASE_URL`    | kosong                      | Tidak | DSN PostgreSQL; kosong = file JSON   |
| `STORE_PATH`      | `data/users.json`           | Tidak | File JSON saat `DATABASE_URL` kosong |
| `KTM_STORAGE_DIR` | `data/ktm`                  | Tidak | Direktori file KTM (storage lokal)   |
| `LISTING_STORAGE_DIR` | `data/listings`         | Tidak | Direktori foto listing (storage lokal) |
| `AGREEMENT_PDF_PATH` | `assets/rental-agreement.pdf` | Tidak | PDF perjanjian statis yang ditampilkan |

`JWT_TTL` memakai format durasi Go, misalnya `30m`, `12h`, atau `24h`. Format seperti
`7d` tidak didukung.

Jika `JWT_SECRET` kosong, server memakai secret development dan menampilkan peringatan.
Konfigurasi tersebut tidak boleh dipakai di production.

`KTM_STORAGE_DIR` hanya dipakai implementasi `ktm.LocalStorage`. File disimpan dengan
permission `0600` di luar database; tabel `ktm_verifications` hanya menyimpan
`storage_key`. Untuk pindah ke Supabase, buat implementasi `ktm.Storage` baru
(`Save`/`Open`/`Delete`) lalu ganti wiring di `cmd/server/main.go`; service dan handler
tidak perlu berubah.

---

## API

Dokumentasi request dan response lengkap tersedia di [README utama](../README.md#api-reference).

| Method | Endpoint                      | Auth   | Role         | Deskripsi                        |
| ------ | ----------------------------- | ------ | ------------ | -------------------------------- |
| GET    | `/api/health`                 | Tidak  | -            | Health check                     |
| POST   | `/api/auth/register`          | Tidak  | -            | Registrasi mahasiswa             |
| POST   | `/api/auth/login`             | Tidak  | -            | Login user                       |
| GET    | `/api/auth/me`                | Bearer | semua        | Data user yang sedang login      |
| POST   | `/api/ktm`                    | Bearer | `mahasiswa`  | Ajukan verifikasi KTM (multipart) |
| GET    | `/api/ktm/me`                 | Bearer | `mahasiswa`  | Riwayat verifikasi KTM sendiri   |
| GET    | `/api/ktm/{id}/file`          | Bearer | pemilik/admin| File KTM (JPEG/PNG)              |
| GET    | `/api/admin/ktm`              | Bearer | `admin_dev`  | Antrean verifikasi KTM           |
| POST   | `/api/admin/ktm/{id}/verify`  | Bearer | `admin_dev`  | Setujui pengajuan                |
| POST   | `/api/admin/ktm/{id}/reject`  | Bearer | `admin_dev`  | Tolak pengajuan                  |
| POST   | `/api/admin/ktm/{id}/revoke`  | Bearer | `admin_dev`  | Cabut verifikasi                 |
| GET    | `/api/categories`             | Tidak  | -            | Kategori aktif                   |
| GET    | `/api/campus-locations`       | Tidak  | -            | Lokasi serah-terima              |
| GET    | `/api/listings`               | Tidak  | -            | Katalog + pencarian + filter     |
| GET    | `/api/listings/{id}`          | Tidak  | -            | Detail listing aktif             |
| GET    | `/api/listings/{id}/photos/{photoID}/file` | Tidak | -   | File foto listing                |
| GET    | `/api/listings/mine`          | Bearer | `mahasiswa`  | Listing milik sendiri            |
| POST   | `/api/listings`               | Bearer | `mahasiswa`  | Buat listing (wajib KTM verified)|
| PUT    | `/api/listings/{id}`          | Bearer | `mahasiswa`  | Ubah listing                     |
| DELETE | `/api/listings/{id}`          | Bearer | `mahasiswa`  | Hapus listing                    |
| POST   | `/api/listings/{id}/photos`   | Bearer | `mahasiswa`  | Unggah foto (maks 3)             |
| DELETE | `/api/listings/{id}/photos/{photoID}` | Bearer | `mahasiswa` | Hapus foto                  |
| GET    | `/api/listings/{id}/blocks`   | Bearer | `mahasiswa`  | Daftar blokir tanggal            |
| POST   | `/api/listings/{id}/blocks`   | Bearer | `mahasiswa`  | Tambah blokir tanggal            |
| DELETE | `/api/listings/{id}/blocks/{blockID}` | Bearer | `mahasiswa` | Hapus blokir              |
| GET    | `/api/admin/listings`         | Bearer | `admin_dev`  | Semua listing (`?status=`)       |
| POST   | `/api/admin/listings/{id}/takedown` | Bearer | `admin_dev` | Takedown listing            |
| POST   | `/api/admin/listings/{id}/restore`  | Bearer | `admin_dev` | Pulihkan listing             |
| GET    | `/api/listings/{id}/availability` | Tidak | -         | Tanggal terpakai             |
| POST   | `/api/listings/{id}/rentals`  | Bearer | `mahasiswa`  | Ajukan sewa                  |
| GET    | `/api/rentals/mine`           | Bearer | semua        | Rental milik user            |
| GET    | `/api/rentals/{id}`           | Bearer | peserta      | Detail rental + events       |
| POST   | `/api/rentals/{id}/approve`   | Bearer | penyedia     | Setujui pengajuan            |
| POST   | `/api/rentals/{id}/reject`    | Bearer | penyedia     | Tolak pengajuan              |
| POST   | `/api/rentals/{id}/cancel`    | Bearer | peserta      | Batalkan rental              |
| POST   | `/api/rentals/{id}/complete`  | Bearer | penyedia     | Tandai selesai               |
| GET    | `/api/rentals/{id}/agreement/file` | Bearer | peserta | PDF perjanjian (baca saja)   |
| GET    | `/api/rentals/{id}/messages`  | Bearer | peserta      | Riwayat chat                 |
| POST   | `/api/rentals/{id}/messages`  | Bearer | peserta      | Kirim chat (fallback REST)   |
| POST   | `/api/rentals/{id}/ws-ticket` | Bearer | peserta      | Tiket WebSocket sekali pakai |
| GET    | `/api/rentals/{id}/ws`        | Tiket  | peserta      | Koneksi chat realtime        |
| GET    | `/api/notifications`          | Bearer | semua        | Notifikasi in-app            |
| POST   | `/api/notifications/{id}/read`| Bearer | pemilik      | Tandai sudah dibaca          |

Contoh register (semua field wajib; `nim` 11 digit format `AA FF PPP J UUU`, dengan
`FF` = kode fakultas FT `05` dan `PPP` = kode prodi):

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"Budi","email":"budi@example.com","password":"rahasia123","nim":"24051204999"}'
```

Contoh login:

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"budi@example.com","password":"rahasia123"}'
```

Contoh endpoint terproteksi:

```bash
curl http://localhost:8080/api/auth/me \
  -H 'Authorization: Bearer <token>'
```

### Verifikasi KTM

Upload memakai `multipart/form-data`, field `file`, hanya `image/jpeg` atau
`image/png`, maksimal 5 MB. Service menyimpan file lebih dulu, lalu membuat baris
`pending`; jika insert gagal, file dihapus kembali. Metadata dan status dibaca dari
`ktm_verifications`, bukan kolom di `users`.

```bash
curl -X POST http://localhost:8080/api/ktm \
  -H 'Authorization: Bearer <token>' \
  -F 'file=@ktm.png;type=image/png'
```

Admin mengambil antrean dan memberi keputusan:

```bash
curl 'http://localhost:8080/api/admin/ktm?status=pending' \
  -H 'Authorization: Bearer <token-admin>'

curl -X POST http://localhost:8080/api/admin/ktm/<id>/verify \
  -H 'Authorization: Bearer <token-admin>' \
  -H 'Content-Type: application/json' \
  -d '{"confirm_nim":true}'

curl -X POST http://localhost:8080/api/admin/ktm/<id>/reject \
  -H 'Authorization: Bearer <token-admin>' \
  -H 'Content-Type: application/json' \
  -d '{"reason":"Foto KTM buram"}'
```

Aturan penting:

- Satu user hanya boleh punya satu pengajuan `pending`.
- `verify` wajib `confirm_nim: true`; `reject`/`revoke` wajib `reason`.
- `revoke` hanya untuk pengajuan `verified`.
- File diakses lewat `GET /api/ktm/{id}/file`; hanya pemilik atau `admin_dev`.
- Akses file dan keputusan admin dicatat di `moderation_actions`.
- Kode error: `ktm_pending_exists`/`ktm_already_verified` (409),
  `ktm_invalid_transition` (409), `ktm_not_eligible`/`forbidden` (403),
  `file_too_large` (413), `ktm_file_purged` (410).

### Katalog dan listing

Implementasi ada di `internal/catalog`. Aturan utama:

- Create listing mensyaratkan `is_ktm_verified(owner_id)`; trigger database tetap
  menjadi penjaga terakhir dan error-nya dipetakan ke `403 ktm_not_verified`.
- Validasi service mencerminkan constraint DB: judul 3-120 karakter, deskripsi
  maks 2000, harga 1-10.000.000, kondisi 1-100. Pelanggaran dibalas `422
  validation_error` dengan `fields`.
- Foto maksimal 3 per listing, `image/jpeg` atau `image/png`, maks 5 MB. Posisi 1-3
  diisi slot kosong terkecil. File disimpan lewat `catalog.Storage`; jika insert
  database gagal, file dihapus kembali.
- Blokir tanggal disimpan sebagai `DATERANGE` inklusif. Overlap dengan blokir lain
  atau rental `approved` ditolak database dan dipetakan ke `409 block_overlap`.
- Katalog publik hanya menampilkan listing aktif milik penyedia yang masih verified.
  Email, NIM, dan nomor WhatsApp tidak pernah ikut response.
- Takedown/restore hanya untuk `admin_dev`, wajib alasan saat takedown, dan dicatat
  di `moderation_actions`.

Kode error tambahan: `listing_not_found` (404), `ktm_not_verified`/`forbidden`
(403), `photo_limit`/`listing_in_use`/`block_overlap`/`listing_invalid_transition`
(409), `file_too_large` (413).

### Rental, chat, dan perjanjian

Implementasi ada di `internal/rental` dan `internal/httpapi/rental_handler.go`.
Migration chat ada di `0009_rental_chat`; tanpa migration ini endpoint chat gagal.

- Submit hanya untuk mahasiswa KTM verified, bukan pemilik listing, tanggal tidak
  lampau, dan durasi maksimal 30 hari. Harga/owner diisi trigger database.
- Approve menolak overlap rental `approved` (`409 date_unavailable`) dan tanggal
  yang diblokir penyedia (`409 dates_blocked`). Transisi status final tidak bisa
  diubah lagi.
- Setiap transisi menulis `rental_events` dan `notifications` untuk pihak lain.
- Chat: `POST /ws-ticket` menerbitkan tiket 32 byte sekali pakai selama 60 detik;
  hub WebSocket in-memory per rental. Riwayat dan catch-up memakai
  `GET /messages?after=<RFC3339>`. Pesan 1-2000 karakter, rate limit 30/menit per
  koneksi. Status `rejected`/`cancelled`/`completed` hanya boleh dibaca.
- Frame client `{"type":"message","body":"..."}`; frame server
  `{"type":"message","message":{...}}` atau `{"type":"error","code":"...","message":"..."}`.
- `GET /agreement/file` menyajikan PDF statis dari `AGREEMENT_PDF_PATH` hanya untuk
  peserta rental. Tidak ada pembuatan PDF dan `security_letters` tidak diisi.
- Server mematikan `ReadTimeout`/`WriteTimeout` karena koneksi WebSocket panjang;
  `ReadHeaderTimeout` tetap aktif.
- Hub hanya cocok untuk **satu instance**. Multi-instance perlu Redis pub/sub atau
  PostgreSQL `LISTEN/NOTIFY`.

---

## Alur Autentikasi

### Register

1. Service memvalidasi nama, email, password, dan format NIM (11 digit).
2. Email dinormalisasi menjadi lowercase dan NIM di-trim.
3. Password di-hash dengan bcrypt.
4. `user.Store.Create` menyimpan user. Dengan PostgreSQL, trigger
   `users_resolve_study_program` mengisi `study_program_code` dari digit fakultas/prodi
   NIM dan menolak NIM yang tidak cocok dengan prodi Fakultas Teknik. Index unik
   menolak email atau NIM yang sudah dipakai.
5. JWT diterbitkan dengan claim `role` dan dikembalikan bersama data user.

### Login

1. Email dicari di storage.
2. Password dibandingkan dengan hash bcrypt.
3. Jika email tidak ditemukan, service tetap menjalankan bcrypt comparison terhadap
   dummy hash untuk mengurangi perbedaan waktu respons.
4. JWT diterbitkan jika kredensial valid.

### Request terproteksi

1. Middleware membaca header `Authorization: Bearer <token>`.
2. Token diverifikasi menggunakan secret, issuer, algoritma HS256, dan expiry.
3. User ID dari claim `sub` disimpan di context; role dan status terbaru dibaca dari
   database, bukan dari claim `role`.
4. `RequireRole` dapat dipasang setelah `RequireAuth` untuk endpoint admin
   (`RequireRole("admin_dev")`).
5. Handler mengambil data user terbaru dari storage.

---

## Penyimpanan

Ada dua implementasi `user.Store`:

### PostgreSQL (`user.PostgresStore`)

Dipakai jika `DATABASE_URL` diisi. Query memakai `pgx` dan memetakan error constraint
database menjadi error domain:

| Kondisi database                        | Error Go         | HTTP |
| --------------------------------------- | ---------------- | ---- |
| `users_email_lower_key` dilanggar       | `ErrEmailTaken`  | 409 `email_taken` |
| `users_nim_key` dilanggar               | `ErrNIMTaken`    | 409 `nim_taken` |
| NIM tidak cocok prodi / format salah    | `ErrInvalidNIM`  | 422 field `nim` |
| User tidak ditemukan / ID bukan UUID    | `ErrNotFound`    | 401 / 404 |

`study_program_code` tidak pernah dikirim client; nilainya diisi trigger database.

### File JSON (`user.MemoryStore`)

Fallback saat `DATABASE_URL` kosong:

- Data disimpan di memori dengan `sync.RWMutex`.
- Data dipersist ke file JSON jika `STORE_PATH` diisi.
- Penulisan file dilakukan secara atomik menggunakan file sementara dan `rename`.
- File dibuat dengan permission `0600`.
- Email disimpan lowercase dan dicek unik.
- NIM dicek unik, tetapi validasi fakultas/prodi hanya ada di PostgreSQL.

Interface yang tersedia:

```go
type Store interface {
    Create(ctx context.Context, u User) (User, error)
    FindByEmail(ctx context.Context, email string) (User, error)
    FindByID(ctx context.Context, id string) (User, error)
}
```

`Create` mengembalikan user hasil penyimpanan karena PostgreSQL mengisi `id`,
`study_program_code`, `created_at`, dan `updated_at`.

---

## Build dan Verifikasi

```bash
go build ./...
go vet ./...
gofmt -l .
```

Menjalankan server dengan konfigurasi eksplisit:

```bash
JWT_SECRET=$(openssl rand -base64 48) go run ./cmd/server
```

