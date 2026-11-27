# UNESARENT

Root proyek dengan:

- **Frontend**: React 19 + Vite (JavaScript)
- **Backend**: Go (standard library `net/http`)
- **Auth**: register + login + endpoint `me` menggunakan **JWT (HS256)** dengan claim `role` dan password **bcrypt**

## Fitur

- **1.3.2 Manajemen listing** — penyedia dapat membuat, mengubah, menghapus,
  mengunggah maksimal 3 foto, dan memblokir tanggal listing.
- **1.3.3 Katalog dan pencarian** — pencarian judul, filter kategori/lokasi/harga,
  sort, detail listing, dan moderasi admin.
- **1.3.4 Pengajuan rental dan offer** — ajukan sewa, approve/reject penyedia,
  cancel/complete, availability, dan notifikasi in-app.
- **1.3.5 Chat dan perjanjian** — chat realtime WebSocket dengan riwayat REST,
  tiket sekali pakai, dan PDF perjanjian statis untuk peserta rental.

---

## Struktur

```
.
├── backend/
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── auth/
│   │   │   ├── jwt.go
│   │   │   └── password.go
│   │   ├── catalog/
│   │   │   ├── model.go
│   │   │   ├── service.go
│   │   │   ├── storage.go
│   │   │   ├── store.go
│   │   │   └── store_postgres.go
│   │   ├── config/
│   │   ├── httpapi/
│   │   │   ├── listing_handler.go
│   │   │   ├── rental_handler.go
│   │   │   └── rental_chat_hub.go
│   │   ├── rental/
│   │   │   ├── model.go
│   │   │   ├── service.go
│   │   │   ├── store.go
│   │   │   └── store_postgres.go
│   │   └── user/
│   ├── .env.example
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── auth/
│   │   │   ├── AuthContext.jsx
│   │   │   ├── useAuth.js
│   │   │   ├── RequireAuth.jsx
│   │   │   └── tokenStorage.js
│   │   ├── lib/
│   │   │   ├── http.js
│   │   │   └── authApi.js
│   │   ├── App.jsx
│   │   └── main.jsx
│   └── vite.config.js
├── postgres/
│   ├── initdb/000_extensions.sql
│   ├── migrations/            # 0001 identity ... 0013 admin governance
│   ├── seed/reference.sql     # seed produksi (tanpa data demo)
│   ├── seed/dev_seed.sql      # HANYA untuk lokal
│   ├── docker-compose.yml
│   ├── .env.example
│   └── README.md
├── Makefile
└── README.md
```

---

## Menjalankan

Prasyarat: Go 1.26+, Node.js 20.19+ (atau 22.12+), dan Docker .

### 1. Backend

```bash
cd backend
cp .env.example .env
go run ./cmd/server
```

Ganti `JWT_SECRET` di `.env` dengan nilai acak minimal 32 karakter.

Server berjalan di `http://localhost:8080`. Cek dengan:

```bash
curl http://localhost:8080/api/health
```

Response: `{"status":"ok"}`

### 2. Frontend

Di terminal lain:

```bash
cd frontend
npm install
npm run dev
```

Buka `http://localhost:5173`.

### Jalankan keduanya sekaligus

```bash
make install
make dev
```

---

## PostgreSQL

Setup PostgreSQL ada di folder `postgres/`, berisi Docker Compose, migration
versioned (`golang-migrate`), seed development, dan schema lengkap MVP:

- identitas: `study_programs`, `users`, `user_contacts`, `ktm_verifications`
- katalog: `categories`, `campus_locations`, `listings`, `listing_photos`, `listing_blocks`
- transaksi: `rentals`, `rental_events`, `rental_messages`, `letter_templates`, `security_letters`
- governance: `reports`, `moderation_actions`, `notifications`

```bash
cd postgres
cp .env.example .env
docker compose up -d   # PostgreSQL + migration otomatis
```

Atau dari root proyek:

```bash
make db-up             # jalankan PostgreSQL dan migration
make db-seed           # seed produksi: prodi, kategori, lokasi (reference.sql)
make db-seed-dev       # seed demo lengkap; HANYA untuk lokal
make db-migrate        # jalankan migration yang belum diterapkan
make db-migrate-down   # rollback satu migration
make db-down           # hentikan container, data tetap ada
make db-reset          # hapus volume + buat ulang + migration
```

Schema dijalankan otomatis saat container pertama kali dibuat, lalu service `migrate`
menerapkan seluruh migration. Detail koneksi, ERD, aturan integritas, dan cara
menjalankan tanpa Docker ada di [`postgres/README.md`](postgres/README.md).

> Seed development memakai kode fakultas FT `05` dan kode prodi `120`
> (S1 Teknik Informatika). Tambahkan kode prodi FT lain setelah daftar resmi
> diperoleh; jangan menebak kode karena trigger memakainya untuk validasi NIM.

---

## Endpoint API

| Method | Path                          | Auth                | Deskripsi                              |
| ------ | ----------------------------- | ------------------- | -------------------------------------- |
| GET    | `/api/health`                 | -                   | Health check                           |
| POST   | `/api/auth/register`          | -                   | Daftar mahasiswa, langsung dapat token |
| POST   | `/api/auth/login`             | -                   | Login, dapat token                     |
| GET    | `/api/auth/me`                | Bearer              | Data user yang sedang login            |
| POST   | `/api/ktm`                    | Bearer + mahasiswa  | Ajukan verifikasi KTM (multipart)      |
| GET    | `/api/ktm/me`                 | Bearer + mahasiswa  | Riwayat dan status verifikasi KTM      |
| GET    | `/api/ktm/{id}/file`          | Bearer              | File KTM (pemilik atau `admin_dev`)    |
| GET    | `/api/admin/ktm`              | Bearer + `admin_dev` | Antrean verifikasi (`?status=`)        |
| POST   | `/api/admin/ktm/{id}/verify`  | Bearer + `admin_dev` | Setujui pengajuan                      |
| POST   | `/api/admin/ktm/{id}/reject`  | Bearer + `admin_dev` | Tolak pengajuan                        |
| POST   | `/api/admin/ktm/{id}/revoke`  | Bearer + `admin_dev` | Cabut verifikasi                       |
| GET    | `/api/categories`             | -                   | Daftar kategori aktif                  |
| GET    | `/api/campus-locations`       | -                   | Daftar lokasi serah-terima            |
| GET    | `/api/listings`               | -                   | Katalog: pencarian + filter + sort    |
| GET    | `/api/listings/{id}`          | -                   | Detail listing aktif                  |
| GET    | `/api/listings/{id}/photos/{photoID}/file` | -    | File foto listing                     |
| GET    | `/api/listings/mine`          | Bearer + mahasiswa  | Listing milik sendiri                 |
| POST   | `/api/listings`               | Bearer + mahasiswa  | Buat listing (wajib KTM verified)     |
| PUT    | `/api/listings/{id}`          | Bearer + mahasiswa  | Ubah listing milik sendiri            |
| DELETE | `/api/listings/{id}`          | Bearer + mahasiswa  | Hapus listing milik sendiri           |
| POST   | `/api/listings/{id}/photos`   | Bearer + mahasiswa  | Unggah foto (multipart, maks 3)       |
| DELETE | `/api/listings/{id}/photos/{photoID}` | Bearer + mahasiswa | Hapus foto                    |
| GET    | `/api/listings/{id}/blocks`   | Bearer + mahasiswa  | Periode blokir milik sendiri          |
| POST   | `/api/listings/{id}/blocks`   | Bearer + mahasiswa  | Blokir rentang tanggal                |
| DELETE | `/api/listings/{id}/blocks/{blockID}` | Bearer + mahasiswa | Buka blokir                  |
| GET    | `/api/admin/listings`         | Bearer + `admin_dev` | Semua listing (`?status=`)          |
| POST   | `/api/admin/listings/{id}/takedown` | Bearer + `admin_dev` | Takedown listing              |
| POST   | `/api/admin/listings/{id}/restore`  | Bearer + `admin_dev` | Pulihkan listing               |
| GET    | `/api/listings/{id}/availability` | -                | Tanggal yang sudah terpakai           |
| POST   | `/api/listings/{id}/rentals`  | Bearer + mahasiswa  | Ajukan sewa                           |
| GET    | `/api/rentals/mine`           | Bearer              | Rental sebagai penyewa/penyedia       |
| GET    | `/api/rentals/{id}`           | Bearer              | Detail rental + timeline              |
| POST   | `/api/rentals/{id}/approve`   | Bearer (penyedia)   | Setujui pengajuan                     |
| POST   | `/api/rentals/{id}/reject`    | Bearer (penyedia)   | Tolak pengajuan                       |
| POST   | `/api/rentals/{id}/cancel`    | Bearer (peserta)    | Batalkan rental                       |
| POST   | `/api/rentals/{id}/complete`  | Bearer (penyedia)   | Tandai selesai                        |
| GET    | `/api/rentals/{id}/agreement/file` | Bearer (peserta) | PDF perjanjian (baca saja)         |
| GET    | `/api/rentals/{id}/messages`  | Bearer (peserta)    | Riwayat chat (`?after=`)              |
| POST   | `/api/rentals/{id}/messages`  | Bearer (peserta)    | Kirim chat (fallback REST)            |
| POST   | `/api/rentals/{id}/ws-ticket` | Bearer (peserta)    | Tiket WebSocket sekali pakai          |
| GET    | `/api/rentals/{id}/ws`        | Tiket WebSocket     | Koneksi chat realtime                 |
| GET    | `/api/notifications`          | Bearer              | Notifikasi in-app (`?unread=1`)       |
| POST   | `/api/notifications/{id}/read`| Bearer              | Tandai notifikasi sudah dibaca        |

### Register

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"Budi Santoso","email":"budi@example.com","password":"rahasia123","nim":"24051204999"}'
```

`201 Created`

```json
{
  "token": "eyJhbGciOi...",
  "token_type": "Bearer",
  "expires_at": "2026-11-15T11:57:02+07:00",
  "user": {
    "id": "9399caea-c9e8-483a-8a48-5104f65c4491",
    "name": "Budi Santoso",
    "email": "budi@example.com",
    "role": "mahasiswa",
    "nim": "24051204999",
    "study_program_code": "120",
    "status": "active",
    "created_at": "2026-11-15T15:35:07+07:00"
  }
}
```

`nim` wajib, 11 digit dengan format `AA FF PPP J UUU`. `FF` adalah kode fakultas
(Fakultas Teknik = `05`) dan `PPP` adalah kode prodi (contoh S1 Teknik Informatika =
`120`). Kombinasi fakultas/prodi harus ada di tabel `study_programs`.
`study_program_code` diisi otomatis oleh database.

### Login

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"budi@example.com","password":"rahasia123"}'
```

`200 OK` dengan bentuk response yang sama seperti register.

### Me

```bash
curl http://localhost:8080/api/auth/me \
  -H 'Authorization: Bearer <token>'
```

```json
{
  "user": {
    "id": "9399caea-c9e8-483a-8a48-5104f65c4491",
    "name": "Budi Santoso",
    "email": "budi@example.com",
    "role": "mahasiswa",
    "nim": "24051204999",
    "study_program_code": "120",
    "status": "active",
    "created_at": "2026-11-15T15:35:07+07:00"
  }
}
```

### Verifikasi KTM

Akun mahasiswa boleh aktif tanpa KTM, tetapi belum boleh membuat listing atau
menyewa. Pengajuan KTM adalah tahap terpisah dan wajib direview `admin_dev`.

Format file: **JPEG atau PNG**, maksimal **5 MB**, field multipart bernama `file`.
Di development file disimpan di `KTM_STORAGE_DIR` (default `backend/data/ktm`);
database hanya menyimpan `storage_key`.

```bash
# Ajukan KTM (mahasiswa)
curl -X POST http://localhost:8080/api/ktm \
  -H 'Authorization: Bearer <token>' \
  -F 'file=@ktm.png;type=image/png'

# Riwayat + status verifikasi sendiri
curl http://localhost:8080/api/ktm/me \
  -H 'Authorization: Bearer <token>'

# Antrean admin (default status=pending)
curl 'http://localhost:8080/api/admin/ktm?status=pending' \
  -H 'Authorization: Bearer <token-admin>'

# Setujui: admin wajib mengonfirmasi NIM pada kartu
curl -X POST http://localhost:8080/api/admin/ktm/<id>/verify \
  -H 'Authorization: Bearer <token-admin>' \
  -H 'Content-Type: application/json' \
  -d '{"confirm_nim":true}'

# Tolak / cabut: alasan wajib
curl -X POST http://localhost:8080/api/admin/ktm/<id>/reject \
  -H 'Authorization: Bearer <token-admin>' \
  -H 'Content-Type: application/json' \
  -d '{"reason":"Foto KTM buram"}'

curl -X POST http://localhost:8080/api/admin/ktm/<id>/revoke \
  -H 'Authorization: Bearer <token-admin>' \
  -H 'Content-Type: application/json' \
  -d '{"reason":"KTM bukan milik pemilik akun"}'
```

Aturan status: hanya satu pengajuan `pending` per user; `verified` tidak bisa
diverifikasi ulang; `rejected`/`revoked` wajib punya alasan; pengajuan baru boleh
dibuat setelah status sebelumnya `rejected`/`revoked`. Akses file memakai
`GET /api/ktm/{id}/file` dan hanya diizinkan untuk pemilik akun atau `admin_dev`.
Setiap verify/reject/revoke dan akses file dicatat di `moderation_actions`.

### Katalog dan listing

Penyedia wajib sudah **terverifikasi KTM** untuk membuat listing. File foto
disimpan di `LISTING_STORAGE_DIR` (default `backend/data/listings`), database hanya
menyimpan `storage_key`.

```bash
# Katalog publik: cari + filter
curl 'http://localhost:8080/api/listings?q=kamera&category=elektronik&sort=price_asc&limit=20'

# Buat listing
curl -X POST http://localhost:8080/api/listings \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{
    "title":"Kamera Canon EOS 700D",
    "description":"Termasuk tas dan charger.",
    "category_id":"<uuid-kategori>",
    "campus_location_id":"<uuid-lokasi>",
    "price_per_day":75000,
    "condition_percent":90
  }'

# Unggah foto (maks 3, JPEG/PNG, 5 MB)
curl -X POST http://localhost:8080/api/listings/<id>/photos \
  -H 'Authorization: Bearer <token>' \
  -F 'file=@kamera.jpg;type=image/jpeg'

# Blokir tanggal (inklusif)
curl -X POST http://localhost:8080/api/listings/<id>/blocks \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{"start_date":"2026-12-01","end_date":"2026-12-03","reason":"Dipakai sendiri"}'

# Admin takedown / restore
curl -X POST http://localhost:8080/api/admin/listings/<id>/takedown \
  -H 'Authorization: Bearer <token-admin>' \
  -H 'Content-Type: application/json' \
  -d '{"reason":"Barang tidak sesuai deskripsi"}'
```

Aturan:

- List publik hanya menampilkan listing `is_active = true` milik penyedia yang KTM-nya
  masih verified. Data owner dibatasi ke `{id, name}`; email, NIM, dan WhatsApp tidak
  pernah dikirim.
- Filter katalog: `q` (judul), `category` (slug), `location` (UUID), `min_price`,
  `max_price`, `sort` (`newest`, `price_asc`, `price_desc`), `limit` (default 20,
  maks 60), `offset`.
- Hapus listing bersifat permanen; jika sudah dipakai transaksi ditolak `409
  listing_in_use`.
- Takedown wajib alasan dan dicatat di `moderation_actions`; listing takedown tidak
  muncul di katalog sampai dipulihkan admin.

### Rental, chat, dan perjanjian

Penyewa wajib sudah terverifikasi KTM. Harga dan `owner_id` selalu diambil dari
listing oleh trigger database, bukan dari body request. `end_date` inklusif dan
durasi maksimal 30 hari.

```bash
# Cek tanggal yang sudah terpakai (blokir + rental approved)
curl 'http://localhost:8080/api/listings/<listing-id>/availability?from=2026-12-01&to=2026-12-31'

# Ajukan sewa
curl -X POST http://localhost:8080/api/listings/<listing-id>/rentals \
  -H 'Authorization: Bearer <token-penyewa>' \
  -H 'Content-Type: application/json' \
  -d '{"start_date":"2026-12-01","end_date":"2026-12-03"}'

# Penyedia menyetujui / menolak
curl -X POST http://localhost:8080/api/rentals/<rental-id>/approve \
  -H 'Authorization: Bearer <token-penyedia>'

curl -X POST http://localhost:8080/api/rentals/<rental-id>/reject \
  -H 'Authorization: Bearer <token-penyedia>' \
  -H 'Content-Type: application/json' \
  -d '{"reason":"Barang sedang dipakai"}'
```

Status: `pending -> approved/rejected/cancelled`, `approved -> completed/cancelled`.
Setiap transisi dicatat di `rental_events` dan membuat notifikasi in-app untuk pihak
lain. Approval kedua pada tanggal yang sama ditolak `409 date_unavailable`.

Chat memakai **REST untuk riwayat** dan **WebSocket untuk pesan baru**. Ambil tiket
sekali pakai lebih dulu, lalu buka koneksi (tiket berlaku 60 detik):

```bash
curl -X POST http://localhost:8080/api/rentals/<rental-id>/ws-ticket \
  -H 'Authorization: Bearer <token>'
```

```
GET ws://localhost:8080/api/rentals/<rental-id>/ws?ticket=<tiket>
```

Frame yang dikirim client:

```json
{ "type": "message", "body": "Bisa diambil jam 4 sore?" }
```

Frame dari server:

```json
{
  "type": "message",
  "message": {
    "id": "<uuid>",
    "rental_id": "<uuid>",
    "sender": { "id": "<uuid>", "name": "Penyewa Demo" },
    "body": "Bisa diambil jam 4 sore?",
    "created_at": "2026-11-25T14:13:24+07:00"
  }
}
```

Setelah reconnect, ambil pesan yang terlewat lewat
`GET /api/rentals/<id>/messages?after=<created_at terakhir>`. Kirim pesan lewat
`POST /api/rentals/<id>/messages` juga tetap didukung dan ikut dibroadcast ke
koneksi WebSocket. Chat hanya bisa dikirim saat rental `pending`/`approved`;
status final bersifat baca saja.

PDF perjanjian bersifat **statis** dan sudah disiapkan di
`AGREEMENT_PDF_PATH` (default `backend/assets/rental-agreement.pdf`). Tidak ada
generasi PDF per rental. Peserta rental membuka:

```bash
curl -X GET http://localhost:8080/api/rentals/<rental-id>/agreement/file \
  -H 'Authorization: Bearer <token-peserta>' \
  -o perjanjian.pdf
```

Server WebSocket memakai satu instance (hub in-memory). Untuk multi-instance,
tambahkan pub/sub Redis atau `LISTEN/NOTIFY` PostgreSQL.

### Format error

```json
{
  "error": {
    "code": "validation_error",
    "message": "Data yang dikirim tidak valid",
    "fields": {
      "email": "Format email tidak valid",
      "password": "Password minimal 8 karakter"
    }
  }
}
```

Kode error yang mungkin: `validation_error` (422), `email_taken` / `nim_taken` (409),
`invalid_credentials` (401), `missing_token` / `invalid_token` / `token_expired` (401),
`forbidden` (403), `invalid_body` (400), `internal_error` (500).

> Catatan: body register/login memakai `DisallowUnknownFields`, jadi kirim hanya field
> yang terdaftar (`name`, `email`, `password`, `nim`). Jangan kirim `confirmPassword`
> ke API.

---

## Menghubungkan ke tampilan

### Login

```jsx
import { useState } from 'react'
import { useAuth } from './auth/useAuth'

export default function LoginPage() {
  const { login, pending, error } = useAuth()
  const [form, setForm] = useState({ email: '', password: '' })

  async function handleSubmit(e) {
    e.preventDefault()
    try {
      await login(form)
    } catch (err) {
      console.error(err)
    }
  }

  return (
    <form onSubmit={handleSubmit}>
      <input
        type="email"
        value={form.email}
        onChange={(e) => setForm({ ...form, email: e.target.value })}
        placeholder="Email"
        required
      />
      <input
        type="password"
        value={form.password}
        onChange={(e) => setForm({ ...form, password: e.target.value })}
        placeholder="Password"
        required
      />
      {error && <p>{error.message}</p>}
      {error?.fields?.email && <p>{error.fields.email}</p>}
      {error?.fields?.password && <p>{error.fields.password}</p>}
      {error?.fields?.nim && <p>{error.fields.nim}</p>}
      <button disabled={pending}>{pending ? 'Memproses...' : 'Login'}</button>
    </form>
  )
}
```

### Register

```jsx
const { register, pending, error } = useAuth()

await register({
  name: 'Budi',
  email: 'budi@example.com',
  password: 'rahasia123',
  nim: '24051204999',
})
```

Register otomatis login (token langsung disimpan oleh context). `nim` wajib diisi
dengan NIM yang prefix-nya cocok dengan kode prodi Fakultas Teknik.

### Cek status & logout

```jsx
const { status, user, isAuthenticated, logout } = useAuth()
```

Nilai `status`: `loading`, `anonymous`, atau `authenticated`.

### Guard halaman

```jsx
import { RequireAuth } from './auth/RequireAuth'

<RequireAuth fallback={<LoginPage />}>
  <Dashboard />
</RequireAuth>
```

Kalau memakai `react-router-dom`:

```jsx
if (status === 'loading') return <p>Memuat...</p>
if (!isAuthenticated) return <Navigate to="/login" replace />
```

### Request ke endpoint lain

```js
import { request } from './lib/http'

const data = await request('/api/sesuatu')
```

`request` otomatis menambahkan `Authorization: Bearer <token>` dan melempar `ApiError`
dengan properti `status`, `code`, dan `fields`.
Kalau backend membalas 401 karena token kedaluwarsa, panggil `logout()`.

---

## Environment variable backend

| Variable       | Default                  | Keterangan                              |
| -------------- | ------------------------ | --------------------------------------- |
| `APP_ADDR`     | `:8080`                  | Alamat listen                           |
| `JWT_SECRET`   | secret dev (dengan warning) | **Wajib diganti**, minimal 32 karakter |
| `JWT_TTL`      | `24h`                    | Masa berlaku access token               |
| `CORS_ORIGINS` | `http://localhost:5173,...` | Origin frontend, pisahkan koma       |
| `DATABASE_URL` | kosong                   | DSN PostgreSQL; kosong = file JSON      |
| `STORE_PATH`   | `data/users.json`        | File penyimpanan user saat tanpa database |
| `LISTING_STORAGE_DIR` | `data/listings`    | Direktori foto listing (storage lokal) |
| `AGREEMENT_PDF_PATH` | `assets/rental-agreement.pdf` | PDF perjanjian statis yang ditampilkan |

Frontend memakai `VITE_API_URL` (opsional). Saat development biarkan kosong karena
Vite mem-proxy `/api` ke `http://localhost:8080`.

---

## Catatan keamanan & production

> Rencana produksi lengkap (OTP email, pembatasan domain, KTM, chat terenkripsi,
> COD, admin, dan deploy free tier) ada di
> [`docs/RUNBOOK_PRODUKSI.md`](docs/RUNBOOK_PRODUKSI.md). Dokumen itu adalah
> instruksi eksekusi final untuk sesi AI berikutnya.
>
> Langkah deploy manual (Supabase, Render, Vercel/Cloudflare, Brevo, backup,
> monitoring, troubleshooting) ada di
> [`docs/TUTORIAL_DEPLOY.md`](docs/TUTORIAL_DEPLOY.md).

- Password di-hash bcrypt; hash tidak pernah dikirim ke client.
- Login memakai pesan generik agar tidak membocorkan email mana yang terdaftar.
- Token di client disimpan di `localStorage` (praktis, tetapi rentan XSS). Untuk
  production yang lebih ketat, gunakan access token di memori + refresh token httpOnly.
- Logout bersifat client-side karena JWT stateless. Jika butuh invalidasi token,
  tambahkan refresh token + tabel sesi/denylist.
- Penyimpanan user memakai PostgreSQL (`user.PostgresStore`) jika `DATABASE_URL` diisi.
  Tanpa `DATABASE_URL`, backend memakai file JSON dengan lock (cukup untuk dev).
  PostgreSQL menegakkan validasi NIM/prodi, keunikan email/NIM, dan role.
- JWT membawa claim `role`, tetapi role/status terbaru selalu dibaca dari database;
  middleware `RequireRole` membatasi endpoint admin ke `admin_dev`.
- Sebelum deploy: set `JWT_SECRET` dari secret manager, aktifkan HTTPS, batasi
  `CORS_ORIGINS`, dan tambahkan rate limiting pada endpoint login/register.
