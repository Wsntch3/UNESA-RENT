# UNESARENT

Root proyek dengan:

- **Frontend**: React 19 + Vite (JavaScript)
- **Backend**: Go (standard library `net/http`)
- **Auth**: register + login + endpoint `me` menggunakan **JWT (HS256)** dengan claim `role` dan password **bcrypt**
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
│   │   ├── config/
│   │   ├── httpapi/
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
│   ├── migrations/            # 0001 identity ... 0008 letter governance
│   ├── seed/dev_seed.sql
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
- transaksi: `rentals`, `rental_events`, `letter_templates`, `security_letters`
- governance: `reports`, `moderation_actions`, `notifications`

```bash
cd postgres
cp .env.example .env
docker compose up -d   # PostgreSQL + migration otomatis
```

Atau dari root proyek:

```bash
make db-up             # jalankan PostgreSQL dan migration
make db-seed           # seed prodi, kategori, lokasi, admin, template surat
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

Frontend memakai `VITE_API_URL` (opsional). Saat development biarkan kosong karena
Vite mem-proxy `/api` ke `http://localhost:8080`.

---

## Catatan keamanan & production

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
