# TUTORIAL DEPLOY UNESARENT

> **Untuk siapa:** pemilik proyek yang akan deploy **manual**.
> **Sifat:** langkah operasional. Tidak ada perintah commit/push otomatis.
> **Prasyarat:** kode sudah selesai diedit sesuai `docs/RUNBOOK_PRODUKSI.md` dan build lokal sudah lulus.
> **Versi:** 1.0 — 27 November 2026.

---

## 0. Gambaran arsitektur

```text
Browser
  │ HTTPS (hash router: /#/...)
  ▼
Vercel / Cloudflare Pages  ← React build (VITE_API_URL)
  │ HTTPS + WSS
  ▼
Render Free (Go API)
  │ DATABASE_URL (Supabase pooler :6543)
  │ SUPABASE_URL + service role key (Storage REST)
  ▼
Supabase Free
  ├── PostgreSQL 17
  └── Storage privat `unesarent`
```

Yang dibutuhkan:

- Akun GitHub (repo private).
- Akun Supabase (free).
- Akun Render (free).
- Akun Vercel (Hobby) atau Cloudflare Pages.
- Akun Brevo (free, untuk OTP email).
- Domain (opsional tetapi disarankan).
- Akun UptimeRobot (free).

---

## 1. Persiapkan lokal

Install tool yang dipakai:

```bash
# Go (sesuai go.mod)
go version

# Node.js 20+
node -v
npm -v

# PostgreSQL client (untuk seed/backup)
psql --version

# migrate CLI
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
export PATH="$PATH:$(go env GOPATH)/bin"
migrate -version
```

Pastikan build lokal lulus sebelum deploy:

```bash
cd backend
go build ./...
go test ./...

cd ../frontend
npm ci
npm run build
```

Jika salah satu gagal, jangan deploy dulu.

---

## 2. Supabase

### 2.1 Buat project

1. Buka <https://supabase.com/dashboard>, login.
2. **New project**.
3. Isi:
   - Name: `unesarent-prod`
   - Database password: simpan di password manager.
   - Region: **Southeast Asia (Singapore)**.
   - Plan: Free.
4. Tunggu provisioning selesai.

### 2.2 Simpan kredensial

Buka **Project Settings → API**:

- `Project URL`: `https://<ref>.supabase.co`
- `anon public key`: tidak dipakai backend.
- `service_role key`: **rahasia**, hanya untuk backend. Jangan pernah dimasukkan ke frontend.

Buka **Project Settings → Database → Connection string**:

- **Direct connection** (port `5432`): untuk migrasi, seed, backup.
- **Transaction pooler** (port `6543`): untuk `DATABASE_URL` backend.
- Jika direct connection IPv6-only dan jaringanmu tidak mendukung IPv6, pakai **Session pooler** (port `5432`) untuk migrasi.

Jangan menebak host pooler. Copy persis dari dashboard.

Contoh bentuk:

```text
# app (transaction pooler)
postgres://postgres.<ref>:<password>@aws-0-<region>.pooler.supabase.com:6543/postgres?sslmode=require&default_query_exec_mode=simple_protocol&statement_cache_capacity=0

# migrasi/backup (direct atau session pooler)
postgres://postgres.<ref>:<password>@db.<ref>.supabase.co:5432/postgres?sslmode=require
```

### 2.3 Buat bucket Storage

1. Buka **Storage → New bucket**.
2. Name: `unesarent`
3. Public: **OFF** (private).
4. File size limit: `10 MB`.
5. Allowed MIME types: `image/jpeg,image/png,application/pdf`.
6. Create.

Tidak perlu membuat RLS policy untuk client. Backend memakai `service_role` yang melewati RLS.

### 2.4 Jalankan migrasi

Dari root repo:

```bash
export DATABASE_URL_DIRECT='postgres://...:5432/postgres?sslmode=require'
migrate -path postgres/migrations -database "$DATABASE_URL_DIRECT" up
```

Cek status:

```bash
migrate -path postgres/migrations -database "$DATABASE_URL_DIRECT" version
```

Jika muncul `dirty`, baca bagian Troubleshooting.

**Alternatif tanpa CLI:** buka Supabase **SQL Editor**, lalu paste isi file migrasi satu per satu, urut nama file `0001` → terakhir. Cara ini rawan salah urutan; CLI lebih disarankan.

### 2.5 Jalankan reference seed

```bash
psql "$DATABASE_URL_DIRECT" -v ON_ERROR_STOP=1 -f postgres/seed/reference.sql
```

Isi yang benar:

```sql
SELECT count(*) FROM users;            -- 0
SELECT count(*) FROM study_programs;   -- 1
SELECT count(*) FROM categories;       -- 6
SELECT count(*) FROM campus_locations; -- 5
```

Jangan menjalankan `postgres/seed/dev_seed.sql` di produksi. File itu hanya untuk lokal.

### 2.6 Buat akun admin

Jika `backend/cmd/createadmin` sudah ada (Fase 2):

```bash
cd backend
DATABASE_URL="$DATABASE_URL_DIRECT" \
APP_ENV=development \
JWT_SECRET="$(openssl rand -hex 32)" \
go run ./cmd/createadmin \
  -email 'admin@domainmu.com' \
  -password 'PasswordAdminYangKuat123!'
```

Jika belum ada, buat lewat SQL Editor Supabase (password di-hash bcrypt oleh `pgcrypto`):

```sql
INSERT INTO users (name, email, password_hash, role, status)
VALUES (
  'Administrator UNESA Rent',
  lower('admin@domainmu.com'),
  crypt('PasswordAdminYangKuat123!', gen_salt('bf', 12)),
  'admin_dev',
  'active'
)
ON CONFLICT (lower(email)) DO UPDATE
SET name = EXCLUDED.name,
    password_hash = EXCLUDED.password_hash,
    role = 'admin_dev',
    status = 'active';
```

Jika kolom `email_verified_at` sudah ada (setelah migrasi auth), tambahkan:

```sql
UPDATE users SET email_verified_at = now()
WHERE lower(email) = lower('admin@domainmu.com');
```

Jangan gunakan email admin yang sama dengan email mahasiswa jika ingin memisahkan peran.

---

## 3. Brevo (email OTP)

1. Buat akun <https://www.brevo.com>.
2. **Senders, Domains & Dedicated IPs → Domains** → tambah domain pengirim.
3. Tambahkan record SPF dan DKIM yang diberikan Brevo ke DNS domainmu. Tunggu verifikasi.
4. **SMTP & API → SMTP**:
   - Host: `smtp-relay.brevo.com`
   - Port: `587`
   - Login: email akun Brevo
   - SMTP key: generate dan simpan.
5. `SMTP_FROM` harus memakai domain yang sudah terverifikasi, misalnya:
   `UNESA Rent <no-reply@domainmu.com>`

Tes setelah backend deploy:

```bash
curl -i -X POST https://api.domainmu.com/api/auth/otp/request \
  -H 'Content-Type: application/json' \
  -d '{"email":"emailkamu@mhs.unesa.ac.id","purpose":"register"}'
```

Cek inbox dan folder spam.

---

## 4. Backend di Render

### 4.1 Buat Web Service

1. Buka <https://dashboard.render.com>, login dengan GitHub.
2. **New → Web Service**.
3. Connect repo UNESARENT (private).
4. Isi:
   - Name: `unesarent-api`
   - Root Directory: `backend`
   - Runtime: Go
   - Build Command: `go build -o server ./cmd/server`
   - Start Command: `./server`
   - Health Check Path: `/api/health/ready` (atau `/api/health` jika endpoint ready belum ada)
   - Region: Singapore
   - Plan: Free
5. Create Web Service.

### 4.2 Environment variables

Buka **Environment → Add Environment Variable**. Isi minimal:

```text
APP_ENV=production
DATABASE_URL=<transaction pooler :6543>
JWT_SECRET=<openssl rand -hex 32>
CORS_ORIGINS=https://app.domainmu.com
PUBLIC_BASE_URL=https://api.domainmu.com
TRUSTED_PROXY=true
```

Jika Fase 2 (OTP) sudah dikerjakan, tambahkan:

```text
ALLOWED_STUDENT_EMAIL_DOMAINS=mhs.unesa.ac.id
ADMIN_LOGIN_OTP=true
SMTP_HOST=smtp-relay.brevo.com
SMTP_PORT=587
SMTP_USER=<brevo login>
SMTP_PASS=<brevo smtp key>
SMTP_FROM=UNESA Rent <no-reply@domainmu.com>
OTP_PEPPER=<openssl rand -hex 32>
```

Jika Fase 3 (Supabase Storage) sudah dikerjakan, tambahkan:

```text
STORAGE_DRIVER=supabase
SUPABASE_URL=https://<ref>.supabase.co
SUPABASE_SERVICE_ROLE_KEY=<service role key>
SUPABASE_BUCKET=unesarent
```

Jika Fase 4 (chat encryption) sudah dikerjakan, tambahkan:

```text
CHAT_ENC_KEYS={"1":"<openssl rand -base64 32>"}
CHAT_ENC_ACTIVE_VERSION=1
```

Jika Fase 5 (COD) sudah dikerjakan, tambahkan:

```text
COD_REQUIRE_RETURN=true
```

> Untuk Render, Koyeb, dan Railway, **jangan isi `APP_ADDR`**. Platform menyuntik `PORT`, dan kode otomatis memakainya. `APP_ADDR` hanya dipakai untuk development lokal bila ingin port selain default.

**Penting:** Render Free memakai filesystem ephemeral. Jika `STORAGE_DRIVER` masih `local`, file KTM/foto/dokumen akan hilang setiap restart/deploy. Jangan pakai local storage di produksi.

### 4.3 Deploy

1. Klik **Manual Deploy → Deploy latest commit**.
2. Tunggu build selesai.
3. Buka log; pastikan tidak ada error konfigurasi.
4. Cek:

```bash
curl -i https://unesarent-api.onrender.com/api/health
```

Jika 200, backend hidup. Jika 502, tunggu cold start 30–60 detik.

### 4.4 Custom domain

1. Render → **Settings → Custom Domains → Add Custom Domain**.
2. Isi `api.domainmu.com`.
3. Tambahkan CNAME di DNS:
   ```text
   api.domainmu.com  CNAME  <nama-service>.onrender.com
   ```
4. Tunggu SSL otomatis (bisa beberapa menit).
5. Cek `https://api.domainmu.com/api/health`.

### 4.5 Alternatif backend: Koyeb (tidak disarankan untuk akun baru)

Koyeb masih bisa menjalankan Dockerfile ini, tetapi sekarang **akun baru wajib memakai plan berbayar dan kartu**. Jadi untuk akun baru, Koyeb tidak lagi menjadi jalur tanpa kartu.

Konfigurasi lama tetap disimpan jika suatu saat akun lama masih bisa dipakai:

1. Build context: `backend`
2. Dockerfile: `backend/Dockerfile`
3. Port: `8080`
4. Health check path: `/api/health/ready`
5. Environment variable sama seperti bagian 4.2, tanpa `APP_ADDR`.

### 4.6 Alternatif backend tanpa kartu: Railway Free

Railway punya plan Free **tanpa kartu** dengan domain `.railway.app` permanen. Service akan tidur setelah 5–10 menit tanpa trafik dan bangun saat ada request, sehingga cocok untuk demo/UAS.

1. Pastikan perubahan sudah di-commit dan di-push ke GitHub.
2. Buka <https://railway.com>, login dengan GitHub.
3. **New Project → Empty Project**.
4. Tambahkan service: **Empty Service**, lalu rename misalnya `unesarent-api`.
5. Settings service:
   - **Source → Connect Repo**: pilih repo UNESARENT.
   - **Root Directory**: `/backend` (atau `backend` jika UI tidak menerima slash)
   - **Health Check Path**: `/api/health/ready`
6. Railway akan mendeteksi `backend/Dockerfile` otomatis.
7. Isi environment variable **sama seperti bagian 4.2**. Jangan isi `APP_ADDR`; biarkan kode membaca `PORT` yang disuntik Railway. Untuk `PUBLIC_BASE_URL`, isi dulu sementara `https://example.com` agar service bisa deploy.
8. Deploy dulu. Tombol **Generate Domain** baru biasanya muncul setelah service berhasil deploy dan terdeteksi listen di `PORT`.
9. Setelah deploy sukses, buka **Settings → Networking → Public Networking → Generate Domain**. Akan muncul domain seperti `https://xxxx.up.railway.app`.
10. Kembali ke Variables, ganti `PUBLIC_BASE_URL` dengan domain Railway tersebut, lalu deploy ulang.
11. Cek:

```bash
curl -i https://xxxx.up.railway.app/api/health
curl -i https://xxxx.up.railway.app/api/health/ready
```

Kemudian di Vercel gunakan:

```text
VITE_API_URL=https://xxxx.up.railway.app
```

Dan di Railway perbarui:

```text
CORS_ORIGINS=https://<domain-vercel-kamu>
```

Jika `curl` mengembalikan:

```text
404 {"status":"error","code":404,"message":"Application not found"}
x-railway-fallback: true
```

maka belum ada deployment aktif yang melayani domain, atau target port domain salah. Cek:

1. Tab **Deployments** → buka deployment terakhir → lihat **Deploy Logs**. Pastikan ada baris:

   ```text
   INFO server berjalan addr=:... 
   ```

2. Di **Settings → Networking → Public Networking**, klik edit pada domain dan pastikan target port sama dengan angka pada log tersebut. Jika tidak ada log runtime karena deployment gagal, set `PORT=8080` di Variables dan arahkan target port domain ke `8080`.
3. Kalau deployment gagal, perbaiki error build/deploy dulu sebelum menguji domain.
4. Pastikan domain tersebut terpasang di service backend yang benar, bukan di service/project lain.

Jika log/build mengatakan Railpack tidak menemukan apa pun untuk dibangun dan Dockerfile tidak dipakai, berarti Root Directory masih di root repo. Set Root Directory ke `backend`, atau tambahkan variable `RAILWAY_DOCKERFILE_PATH=Dockerfile` setelah Root Directory di-set.

Jika status service **offline/sleeping**, cek apakah ada perubahan yang belum di-deploy. Di canvas Railway klik **Deploy** pada banner staged changes, lalu tunggu hingga status menjadi **Running/Healthy**. Service Free juga tidur setelah 5–10 menit tanpa trafik dan bangun saat ada request; ini normal, tetapi curl pertama bisa gagal/lama.

Catatan:

- Plan Free memberi $1/bulan credit dan tidak meminta kartu.
- Karena storage memakai Supabase, file KTM/foto/dokumen tidak bergantung pada disk Railway yang ephemeral.
- Cold start beberapa detik masih normal setelah service tidur.

---

## 5. Frontend di Vercel / Cloudflare Pages

### 5.1 Vercel

1. Buka <https://vercel.com>, login dengan GitHub.
2. **Add New → Project → Import** repo UNESARENT.
3. Setting:
   - Root Directory: `frontend`
   - Framework Preset: Vite
   - Build Command: `npm run build`
   - Output Directory: `dist`
4. Environment Variable:
   ```text
   VITE_API_URL=https://api.domainmu.com
   ```
5. Deploy.
6. Buka domain Vercel, cek aplikasi.

Routing memakai hash (`/#/jelajahi`), jadi tidak perlu rewrite SPA. Jika nanti pindah ke path router, tambahkan rewrite ke `index.html`.

### 5.2 Cloudflare Pages (alternatif)

1. Cloudflare Dashboard → **Workers & Pages → Create → Pages → Connect to Git**.
2. Pilih repo.
3. Build command: `npm run build`
4. Build output directory: `dist`
5. Env: `VITE_API_URL=https://api.domainmu.com`
6. Deploy.

### 5.3 Custom domain frontend

- Vercel: **Settings → Domains → Add** `app.domainmu.com`; ikuti record DNS yang diberikan.
- Cloudflare Pages: **Custom domains → Set up**; ikuti record DNS.

`VITE_API_URL` dibaca saat build. Jika diubah, redeploy frontend.

---

## 6. CORS & WebSocket

Di Render, `CORS_ORIGINS` harus **persis** origin frontend:

```text
CORS_ORIGINS=https://app.domainmu.com
```

Jangan pakai trailing slash, jangan `*`.

WebSocket memakai origin yang sama. Jika chat tidak connect:

1. Pastikan `CORS_ORIGINS` benar.
2. Pastikan frontend diakses via `https://`, bukan `http://`.
3. Cek log Render saat handshake `/api/rentals/{id}/ws`.
4. Pastikan tidak ada proxy yang memblokir `Upgrade`.

---

## 7. Operasional

### 7.1 Backup harian

Buat `.github/workflows/backup.yml` di repo:

```yaml
name: backup-db
on:
  schedule:
    - cron: '0 18 * * *' # 01:00 WIB
  workflow_dispatch:

jobs:
  dump:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install PostgreSQL client
        run: |
          sudo apt-get update
          sudo apt-get install -y postgresql-client-17
      - name: Dump
        env:
          DATABASE_URL_DIRECT: ${{ secrets.DATABASE_URL_DIRECT }}
        run: |
          pg_dump "$DATABASE_URL_DIRECT" --no-owner --no-privileges -Fc -f unesarent.dump
      - uses: actions/upload-artifact@v4
        with:
          name: unesarent-db-${{ github.run_id }}
          path: unesarent.dump
          retention-days: 30
```

Tambahkan secret `DATABASE_URL_DIRECT` di **GitHub → Settings → Secrets and variables → Actions**.

Tes manual: **Actions → backup-db → Run workflow**.

### 7.2 Keep-alive

Supabase Free pause setelah 7 hari tanpa aktivitas. Buat `.github/workflows/keepalive.yml`:

```yaml
name: keepalive
on:
  schedule:
    - cron: '0 */6 * * *'
  workflow_dispatch:
jobs:
  ping:
    runs-on: ubuntu-latest
    steps:
      - run: curl -fsS https://api.domainmu.com/api/health
```

### 7.3 Cleanup `pg_cron`

Supabase Dashboard → **Database → Extensions** → aktifkan `pg_cron`. Lalu SQL Editor:

```sql
CREATE OR REPLACE FUNCTION unesarent_cleanup()
RETURNS void LANGUAGE sql AS $$
    DELETE FROM auth_otps WHERE created_at < now() - interval '3 days';
    DELETE FROM notifications
     WHERE read_at IS NOT NULL AND created_at < now() - interval '90 days';
$$;

SELECT cron.schedule('unesarent-cleanup', '0 3 * * *', $$SELECT unesarent_cleanup();$$);
```

Cek:

```sql
SELECT jobid, schedule, command FROM cron.job;
```

Jangan hapus chat/dokumen otomatis; retensi 2/5 tahun untuk bukti.

### 7.4 Monitoring

1. UptimeRobot → **Add New Monitor**:
   - Type: HTTPS
   - URL: `https://api.domainmu.com/api/health`
   - Interval: 5 menit
   - Alert: email
2. Render → **Settings → Notifications**: aktifkan deploy/crash alert.
3. Cek kuota mingguan:
   ```sql
   SELECT pg_size_pretty(pg_database_size(current_database())) AS db_size;
   SELECT count(*) AS dokumen, pg_size_pretty(sum(file_size_bytes)) AS total
   FROM rental_documents;
   ```
4. Supabase Dashboard → Storage → usage.

---

## 8. Smoke test setelah deploy

### 8.1 Dasar

```bash
curl -i https://api.domainmu.com/api/health
```

Harus `200`.

### 8.2 Auth

- [ ] Buka `https://app.domainmu.com`.
- [ ] Daftar dengan email `@mhs.unesa.ac.id` → terima OTP.
- [ ] Kode salah 5× → terkunci.
- [ ] Daftar sukses → login.
- [ ] Lupa sandi → OTP → password baru.
- [ ] Login admin → OTP → dashboard admin.

### 8.3 KTM & listing

- [ ] Upload KTM JPG/PNG ≤5 MB.
- [ ] Admin buka file KTM dari dashboard.
- [ ] Admin verify dengan konfirmasi NIM.
- [ ] User bisa membuat listing + foto.
- [ ] Listing muncul di katalog.

### 8.4 Rental & COD

- [ ] Ajukan rental → owner approve.
- [ ] Chat real-time.
- [ ] Unduh surat perjanjian.
- [ ] Upload bukti serah terima → konfirmasi.
- [ ] Upload bukti pengembalian → konfirmasi.
- [ ] Catat pembayaran → kuitansi.
- [ ] `Selesai` berhasil setelah COD lengkap.

### 8.5 Admin & laporan

- [ ] Statistik admin.
- [ ] Laporan user → antrean.
- [ ] Resolve/reject.
- [ ] Suspend user → logout paksa.
- [ ] Audit log.
- [ ] Ekspor CSV.

### 8.6 Keamanan

```bash
grep -R "service_role\|SUPABASE_SERVICE_ROLE_KEY\|SMTP_PASS\|JWT_SECRET" frontend/dist
```

Tidak boleh ada hasil.

---

## 9. Rollback & restore

### 9.1 Kode

Karena tidak ada commit otomatis, simpan salinan file sebelum mengedit. Jika deploy bermasalah:

1. Kembalikan file dari salinan/backup kerja.
2. Build ulang.
3. Deploy ulang dari Render/Vercel.

Jangan memakai `git reset --hard` atau `git clean -fd` tanpa backup.

### 9.2 Database

Restore dari artifact backup:

```bash
pg_restore --clean --no-owner --no-privileges \
  -d "$DATABASE_URL_DIRECT" unesarent.dump
```

### 9.3 Migrasi

Turun satu versi:

```bash
migrate -path postgres/migrations -database "$DATABASE_URL_DIRECT" down 1
```

Hanya lakukan jika yakin tidak ada data baru yang akan hilang.

### 9.4 Storage

Object Storage tidak ikut terhapus saat rollback DB. Jika rollback besar, catat object yatim lalu hapus manual dari dashboard.

---

## 10. Referensi environment variable

### Backend

| Variable | Wajib | Dipakai sejak | Contoh |
|---|---|---|---|
| `APP_ENV` | ya | sekarang | `production` |
| `APP_ADDR` | tidak | sekarang | `:8080`; diisi otomatis dari `PORT` bila kosong |
| `PORT` | platform | Render | `10000` (disuntik Render, jangan diisi manual) |
| `DATABASE_URL` | ya | sekarang | pooler 6543 + `default_query_exec_mode=simple_protocol` |
| `DATABASE_URL_DIRECT` | CI/migrasi | sekarang | direct 5432 |
| `JWT_SECRET` | ya | sekarang | 32+ byte random |
| `JWT_TTL` | tidak | sekarang | `24h` |
| `CORS_ORIGINS` | ya | sekarang | `https://app.domainmu.com` |
| `PUBLIC_BASE_URL` | ya | sekarang | `https://api.domainmu.com` |
| `TRUSTED_PROXY` | ya | sekarang | `true` |
| `ALLOWED_STUDENT_EMAIL_DOMAINS` | ya | Fase 2 | `mhs.unesa.ac.id` |
| `ADMIN_EMAILS` | tidak | Fase 2 | `admin@domainmu.com` |
| `ADMIN_EMAIL` | bootstrap | Fase 2 | email admin pertama |
| `ADMIN_PASSWORD` | bootstrap | Fase 2 | kata sandi admin pertama |
| `ADMIN_LOGIN_OTP` | ya | Fase 2 | `true` |
| `SMTP_HOST` | ya | Fase 2 | `smtp-relay.brevo.com` |
| `SMTP_PORT` | ya | Fase 2 | `587` |
| `SMTP_USER` | ya | Fase 2 | Brevo login |
| `SMTP_PASS` | ya | Fase 2 | Brevo SMTP key |
| `SMTP_FROM` | ya | Fase 2 | `UNESA Rent <no-reply@domainmu.com>` |
| `OTP_PEPPER` | ya | Fase 2 | 32+ byte random |
| `OTP_TTL` | tidak | Fase 2 | `10m` |
| `OTP_MAX_ATTEMPTS` | tidak | Fase 2 | `5` |
| `REGISTRATION_TICKET_TTL` | tidak | Fase 2 | `15m` |
| `STORAGE_DRIVER` | ya | Fase 3 | `supabase` |
| `SUPABASE_URL` | ya | Fase 3 | `https://<ref>.supabase.co` |
| `SUPABASE_SERVICE_ROLE_KEY` | ya | Fase 3 | service role |
| `SUPABASE_BUCKET` | ya | Fase 3 | `unesarent` |
| `CHAT_ENC_KEYS` | ya | Fase 4 | `{"1":"<base64 32 byte>"}` |
| `CHAT_ENC_ACTIVE_VERSION` | ya | Fase 4 | `1` |
| `COD_REQUIRE_RETURN` | tidak | Fase 5 | `true` |
| `KTM_STORAGE_DIR` | dev | sekarang | `data/ktm` |
| `LISTING_STORAGE_DIR` | dev | sekarang | `data/listings` |
| `RENTAL_DOCUMENTS_DIR` | dev | Fase 5 | `data/rental-documents` |
| `AGREEMENT_PDF_PATH` | dev | Fase 5 | `assets/rental-agreement.pdf` |

`ADMIN_EMAIL` dan `ADMIN_PASSWORD` hanya dipakai sekali oleh `go run ./cmd/createadmin`; jangan disimpan sebagai env server. `RENTAL_DOCUMENTS_DIR` dan `AGREEMENT_PDF_PATH` hanya dipakai saat `STORAGE_DRIVER=local`; produksi memakai Supabase Storage.

Generate secret:

```bash
openssl rand -hex 32       # JWT_SECRET, OTP_PEPPER
openssl rand -base64 32    # nilai CHAT_ENC_KEYS
```

### Frontend

| Variable | Contoh |
|---|---|
| `VITE_API_URL` | `https://api.domainmu.com` |
| `VITE_PROXY_TARGET` | `http://localhost:8080` (dev only) |

Karena `VITE_API_URL` dibaca saat build, CSP frontend dibangkitkan otomatis oleh `frontend/vite.config.js` dari nilai tersebut. Jangan menaruh secret di frontend. Jangan pernah menaruh service role key di `VITE_*`.

---

## 11. Troubleshooting

### `migrate` bilang `dirty`

Artinya migrasi sebelumnya gagal di tengah. Cek:

```bash
migrate -path postgres/migrations -database "$DATABASE_URL_DIRECT" version
```

Perbaiki penyebabnya, lalu:

```bash
migrate -path postgres/migrations -database "$DATABASE_URL_DIRECT" force <versi-terakhir-yang-sukses>
```

Hati-hati: `force` tidak memperbaiki data. Untuk produksi kosong, lebih aman drop schema dan ulangi.

### `prepared statement already exists` / error pooler

Tambahkan ke `DATABASE_URL`:

```text
default_query_exec_mode=simple_protocol&statement_cache_capacity=0
```

Jangan jalankan migrasi lewat port 6543.

### CORS error

- `CORS_ORIGINS` harus persis `https://app.domainmu.com`.
- Tanpa trailing slash.
- Jangan `*`.
- Restart/redeploy setelah mengubah env.

### WebSocket gagal

- Pastikan frontend `https`.
- Cek `CORS_ORIGINS`.
- Cek log Render pada path `/ws`.
- Pastikan tidak ada firewall/proxy yang memblokir `Upgrade`.

### Email OTP tidak masuk

- Pastikan domain pengirim Brevo sudah verified (SPF + DKIM).
- Cek folder spam.
- Pastikan `SMTP_FROM` memakai domain verified.
- Cek log Render untuk error SMTP.
- Brevo free: 300 email/hari.

### Render 502 / lambat

Render Free spin down setelah 15 menit idle. Request pertama bisa 30–60 detik. Pasang UptimeRobot tiap 10 menit.

### File upload hilang setelah restart

Render Free ephemeral. Set `STORAGE_DRIVER=supabase` dan pastikan bucket/kredensial benar. Jangan pakai local storage di produksi.

### Supabase pause

Free tier pause setelah 7 hari tanpa aktivitas. Keep-alive GitHub Actions tiap 6 jam.

### Storage 403

- Pastikan memakai `service_role` key, bukan `anon`.
- Pastikan bucket `unesarent` ada dan private.
- Pastikan `SUPABASE_URL` tanpa trailing slash.

### `pg_dump` version mismatch

Install `postgresql-client-17` (sesuai Supabase). Di GitHub Actions sudah disediakan.

### `relation ... does not exist`

Migrasi belum dijalankan atau berhenti di tengah. Cek `migrate version` dan jalankan `up`.

### Render build gagal karena Go version

Set env `GO_VERSION` di Render sesuai versi di `go.mod`, atau sesuaikan `go.mod` ke versi Go yang tersedia di Render. Build ulang.

---

## 12. Batas free tier (ringkasan)

| Layanan | Batas | Dampak |
|---|---|---|
| Supabase DB | 500 MB | Cukup untuk data teks; file tidak masuk DB. |
| Supabase Storage | 1 GB | KTM 1–3 MB/foto; kompres foto listing. |
| Supabase egress | 5 GB/bulan | Jangan streaming file besar. |
| Supabase | Pause 7 hari idle | Keep-alive tiap 6 jam. |
| Supabase | Tanpa backup otomatis | Backup harian `pg_dump`. |
| Render Free | Spin down 15 menit, cold start 30–60 detik | UptimeRobot. |
| Render Free | 750 jam/bulan | Cukup 1 service 24/7. |
| Render Free | 512 MB RAM | Jangan PDF generation bersamaan. |
| Vercel Hobby | Non-komersial | Boleh untuk UAS; komersial pindah Cloudflare. |
| Brevo | 300 email/hari | Hanya OTP. |

Jika proyek dipakai resmi fakultas, upgrade Supabase/Render dan aktifkan backup/PITR.

---

## 13. Checklist akhir

- [ ] Migrasi `up` sukses, `version` bersih.
- [ ] Reference seed masuk, tidak ada user demo.
- [ ] Admin bisa login.
- [ ] OTP email masuk.
- [ ] KTM upload + verify.
- [ ] Listing + rental + COD + dokumen.
- [ ] Chat real-time.
- [ ] Laporan + audit.
- [ ] Backup artifact muncul.
- [ ] Keep-alive jalan.
- [ ] UptimeRobot hijau.
- [ ] Tidak ada secret di frontend build.
- [ ] Semua acceptance test runbook lulus.
