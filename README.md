# UNESARENT

Marketplace sewa barang untuk mahasiswa FT UNESA. Frontend React + Vite, backend Go dengan `net/http`, database PostgreSQL, email OTP lewat Brevo.

## Yang bisa dilakukan

- Daftar dengan email `@mhs.unesa.ac.id`, verifikasi KTM, lalu menyewa atau menyewakan barang.
- Cari barang, filter kategori/lokasi/harga, lihat detail, dan blokir tanggal tertentu.
- Ajukan sewa, approve/reject, chat realtime, lalu selesaikan transaksi dengan COD dan dokumen.
- Admin bisa memoderasi listing, verifikasi KTM, mengelola user, dan melihat audit.

## Struktur

```text
backend/    API Go, migrasi Dependensi, Dockerfile
frontend/   React + Vite
postgres/   schema, migration, seed
```

## Jalankan lokal

Butuh Go 1.26+, Node 20.19+, dan Docker.

```bash
cd postgres && docker compose up -d
cd backend && cp .env.example .env && go run ./cmd/server
cd frontend && npm install && npm run dev
```

Atau dari root:

```bash
make install
make dev
```

Backend: `http://localhost:8080`, frontend: `http://localhost:5173`.

Cek backend:

```bash
curl http://localhost:8080/api/health
```

## Database

Migrasi memakai `DATABASE_URL_DIRECT` port `5432`. Aplikasi memakai pooler port `6543`.

```bash
migrate -path postgres/migrations -database "$DATABASE_URL_DIRECT" up
migrate -path postgres/migrations -database "$DATABASE_URL_DIRECT" version
```

Versi terakhir harus `13`. Seed produksi:

```bash
psql "$DATABASE_URL_DIRECT" -v ON_ERROR_STOP=1 -f postgres/seed/reference.sql
```

`postgres/seed/dev_seed.sql` hanya untuk lokal. Jangan dipakai di produksi.

## Env yang penting

Backend:

```text
APP_ENV=production
DATABASE_URL=<pooler :6543 + default_query_exec_mode=simple_protocol&statement_cache_capacity=0>
JWT_SECRET=<acak>
CORS_ORIGINS=https://unesa-rent.vercel.app
PUBLIC_BASE_URL=https://<domain-backend>
ADMIN_LOGIN_OTP=true
BREVO_APIKEY=<API key Brevo>
SMTP_FROM=UNESA Rent <email-terverifikasi>
OTP_PEPPER=<acak 32+ karakter>
CHAT_ENC_KEYS={"1":"<base64 32 byte>"}
STORAGE_DRIVER=supabase
SUPABASE_URL=...
SUPABASE_SERVICE_ROLE_KEY=...
SUPABASE_BUCKET=unesarent
```

Frontend:

```text
VITE_API_URL=https://<domain-backend>
```

`VITE_API_URL` dibaca saat build. Kalau diubah, frontend wajib di-build ulang.

## Deploy

- Database dan Storage: Supabase.
- Backend: Railway, Root Directory `backend`, port `8080`, health `/api/health/ready`.
- Frontend: Vercel, Root Directory `frontend`.

Commit saja tidak memicu deploy. Yang memicu auto deploy adalah `git push`.

## Alur login admin

Password benar menghasilkan `otp_required: true` dan `challenge`. Kode OTP dikirim ke email admin. Token JWT baru diterbitkan setelah kode 6 digit diverifikasi, lalu pengguna diarahkan ke dashboard admin.

## Catatan

Jangan simpan `.env`, API key, service role key, atau file KTM di repository. Semua secret cukup ada di platform deploy.
