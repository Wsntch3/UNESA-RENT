# Backend UNESARENT

API Go untuk autentikasi, katalog, rental, chat, verifikasi KTM, dan panel admin. Routing memakai `net/http` tanpa framework. Password bcrypt, token JWT HS256, dan role/status selalu dibaca ulang dari database.

## Menjalankan

```bash
cp .env.example .env
go run ./cmd/server
```

Cek:

```bash
curl http://localhost:8080/api/health
curl http://localhost:8080/api/health/ready
```

`APP_ENV=development` boleh tanpa database dan memakai file JSON. Produksi wajib PostgreSQL.

## Database

Aplikasi memakai `DATABASE_URL` pooler Supabase port `6543` dengan query:

```text
?sslmode=require&default_query_exec_mode=simple_protocol&statement_cache_capacity=0
```

Migrasi dan `createadmin` memakai `DATABASE_URL_DIRECT` port `5432`:

```bash
migrate -path ../postgres/migrations -database "$DATABASE_URL_DIRECT" up
DATABASE_URL="$DATABASE_URL_DIRECT" go run ./cmd/createadmin \
  -email 'admin@domain.com' -password 'passwordminimal12'
```

Seed produksi ada di `postgres/seed/reference.sql`. Jangan pakai `dev_seed.sql` di produksi.

## Env utama

```text
APP_ENV=production
PORT=8080
DATABASE_URL=<pooler :6543>
JWT_SECRET=<acak>
JWT_TTL=24h
CORS_ORIGINS=https://unesa-rent.vercel.app
PUBLIC_BASE_URL=https://<domain-backend>
TRUSTED_PROXY=true
ALLOWED_STUDENT_EMAIL_DOMAINS=mhs.unesa.ac.id
ADMIN_LOGIN_OTP=true
OTP_PEPPER=<acak 32+ karakter>
BREVO_APIKEY=<API key Brevo>
SMTP_FROM=UNESA Rent <email-terverifikasi>
CHAT_ENC_KEYS={"1":"<base64 32 byte>"}
STORAGE_DRIVER=supabase
SUPABASE_URL=...
SUPABASE_SERVICE_ROLE_KEY=...
SUPABASE_BUCKET=unesarent
```

`SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, dan `SMTP_PASS` dipakai hanya kalau `BREVO_APIKEY` kosong. Port SMTP `587` sering diblokir platform; `2525` lebih aman.

Jangan isi `APP_ADDR` di Railway. Biarkan aplikasi membaca `PORT` dari platform.

## Endpoint

```text
/api/auth/*          register, login, login/otp, otp, me
/api/ktm/*           pengajuan, verifikasi, file KTM admin
/api/listings/*      katalog, listing sendiri, moderasi admin
/api/rentals/*       pengajuan, status, chat, dokumen
/api/reports/*       laporan
/api/admin/*         user, audit, statistik
/api/health          health check
/api/health/ready    health check + ping database
```

## Build dan test

```bash
gofmt -w ./cmd ./internal
go build ./...
go test ./...
```

Dockerfile ada di `backend/Dockerfile`. Di Railway, set Root Directory ke `backend` dan target port `8080`.
