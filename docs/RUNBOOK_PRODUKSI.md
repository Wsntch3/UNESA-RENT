# RUNBOOK PRODUKSI UNESARENT

> **Untuk siapa:** sesi AI berikutnya yang mengeksekusi perubahan.
> **Sifat:** final. Semua keputusan desain sudah dikunci di dokumen ini. Jangan bertanya lagi; ikuti urutan fase.
> **Bahasa UI:** semua copy yang dilihat pengguna tetap Bahasa Indonesia dan mengikuti design language yang sudah ada.
> **Versi:** 1.0 — 27 November 2026.

---

## 0. Aturan main untuk AI pelaksana

1. Baca seluruh dokumen ini sebelum menyentuh kode.
2. Kerjakan **fase 0 → 10 berurutan**. Jangan lompat fase.
3. Di akhir setiap fase: jalankan acceptance test fase itu. **Cukup edit file; jangan commit, push, atau membuat branch otomatis kecuali diminta.**
4. Jangan mengubah desain visual di luar yang disebutkan. Jangan menambah dependency frontend baru kecuali disebutkan.
5. Jangan menghapus file yang tidak disebutkan. Jangan memakai `git reset --hard` atau `git clean -fd` tanpa izin.
6. Semua perintah dijalankan dari root repo: `/home/Wsntch3/Downloads/UNESARENT`.
7. Jika ada error, perbaiki dulu sampai acceptance test fase itu lulus, baru lanjut.
8. Produksi **mulai dari database kosong** + reference seed. Data dev tidak dimigrasikan.
9. Jangan pernah menaruh `.env`, service role key, password, atau file KTM di dalam repo.
10. Semua endpoint admin wajib `requireAuth` + `requireAdmin`. Semua akses admin ke data sensitif wajib tercatat di audit log.

---

## 1. Keputusan final (tidak perlu ditanyakan lagi)

| Topik | Keputusan |
|---|---|
| Database produksi | Proyek Supabase baru, kosong, hanya reference seed. Data dev tidak dimigrasikan. |
| Demo | Dihapus dari build produksi. Seed demo hanya untuk lokal (`make db-seed-dev`). |
| Login mahasiswa | Email + password + OTP email. Domain wajib `@mhs.unesa.ac.id`. |
| Login admin | Email + password + OTP email wajib di produksi. Email admin bebas domain. |
| SSO kampus | Tidak dipakai. |
| Identitas | 1 email = 1 akun, 1 NIM = 1 akun. NIM yang sudah terverifikasi tidak bisa dipakai akun lain. NIM dikunci setelah KTM verified. |
| Chat | Isi pesan dienkripsi AES-256-GCM at rest. Admin hanya melihat metadata. Reveal/export isi chat butuh alasan + audit log. |
| COD | Perjanjian PDF per rental, bukti serah terima, bukti pengembalian, kuitansi COD, arsip dokumen. |
| Storage | Supabase Storage, satu bucket privat `unesarent`. Lokal hanya untuk dev. |
| Backend | Go API di Render Free (alternatif: Oracle Always Free). WebSocket tetap didukung. |
| Frontend | Vercel Hobby (boleh untuk UAS non-komersial) atau Cloudflare Pages. |
| Email | Brevo SMTP gratis (300 email/hari). |
| Cron/backup | GitHub Actions + `pg_cron` Supabase. |
| Monitoring | UptimeRobot gratis + endpoint health yang mengecek DB. |
| Role admin | Hanya `admin_dev`. Tidak ada `admin_bem`. |
| Retensi | KTM: selama akun aktif + 1 tahun. Chat: 2 tahun. Dokumen rental: 5 tahun. Audit: 5 tahun. |

### 1.1 Tiga prioritas UAS — sudah masuk semua

| Prioritas yang diminta | Fase | Cakupan di runbook |
|---|---|---|
| **COD Transaction and Documentation** | **Fase 5** (bagian 9) | Perjanjian PDF per rental, state COD, bukti serah terima, bukti pengembalian, kuitansi, arsip dokumen, akses admin + audit. |
| **Rental Status and History** | **Fase 6** (bagian 10) | Timeline status, filter riwayat per peran/status/tanggal, ringkasan tahunan, deteksi overdue, monitoring rental admin, ekspor CSV. |
| **Admin and Reporting** | **Fase 7** (bagian 11) | Laporan pengguna, antrean moderasi, resolve/reject, suspend/reactivate, hapus akun belum verified, audit log read-only, ekspor data, statistik admin. |

Ketergantungan yang perlu diketahui:

- Fase 5 butuh **Fase 3 (Storage)** untuk produksi. Untuk pengembangan lokal, storage lokal sudah cukup; adapter Supabase bisa dipasang belakangan.
- Fase 6 memakai skema rental yang sudah ada (`0003_rentals`) dan tidak menunggu Fase 2/4.
- Fase 7 memakai tabel `reports`, `moderation_actions`, `notifications` yang sudah ada (`0004_governance`), ditambah migrasi `0013`.
- Fase 2 (auth/OTP) dan Fase 4 (chat encryption) tetap wajib untuk produksi, tetapi bukan penghalang untuk mulai mengerjakan 5–7.

### 1.2 Jika eksekusi dimulai dari Fase 5 (bukan Fase 0)

Bacaan wajib untuk AI lain yang masuk di tengah jalan:

**Kondisi repo saat ini**
- Migrasi `0001`–`0009` sudah ada.
- Auth masih email + password/JWT, belum ada OTP/domain allowlist.
- KTM, katalog, rental, chat plaintext, dan role `admin_dev` sudah berjalan.
- Belum ada Supabase Storage; file masih lokal.
- Belum ada enkripsi chat.
- Tabel `reports`, `moderation_actions`, `notifications` sudah ada dari `0004_governance`.

**Yang boleh dan tidak boleh dikerjakan di Fase 5**
- Boleh: tabel dokumen/COD, generator PDF, endpoint COD, UI RentalDetail, akses dokumen admin.
- Boleh: memakai storage lokal untuk dev.
- Jangan: mengubah auth/OTP, domain email, atau enkripsi chat di fase ini.
- Jangan: deploy ke Render dengan storage lokal. Filesystem Render Free bersifat ephemeral; file KTM/foto/dokumen akan hilang saat restart. Untuk produksi, Fase 3 wajib.

**Penomoran migrasi (penting)**
`migrate` menerapkan versi berurutan. Jangan membuat `0012`/`0013` lebih dulu lalu menambahkan `0010`/`0011`; migrasi bernomor lebih kecil akan dilewati.

| Skenario | Urutan file |
|---|---|
| Mulai dari Fase 5 | `0010_cod_documents`, `0011_rental_history`, `0012_admin_governance`, lalu `0013_auth_production`, `0014_chat_encryption` |
| Urutan penuh Fase 0–10 | `0010_auth_production`, `0011_chat_encryption`, `0012_cod_documents`, `0013_admin_governance` |

Isi SQL tidak berubah; hanya nama file dan judul bagian yang disesuaikan. Jika memilih jalur “mulai dari Fase 5”, ganti nama file pada bagian 6.2, 8.1, 9.1, dan 11.1 sesuai tabel di atas.

**Dependency**
- Fase 5: butuh `rental` yang sudah ada + storage (lokal untuk dev, Supabase untuk produksi).
- Fase 6: hanya query/tabel rental yang sudah ada.
- Fase 7: tabel governance yang sudah ada + migrasi admin.
- Fase 2/3/4 tidak menghalangi 5–7, tetapi wajib sebelum produksi resmi.

**Jika sesi sedang limit**
- Cukup edit file. Jangan jalankan `go build`, `npm run build`, atau test berat.
- Laporkan file yang diubah dan perintah verifikasi yang harus dijalankan pemilik secara manual.
- Jangan commit/push.

---

## 2. Arsitektur target & batas free tier

```text
Browser
  │  HTTPS (hash router, tidak butuh SPA rewrite)
  ▼
Vercel / Cloudflare Pages  ── static React build (VITE_API_URL)
  │  HTTPS + WSS
  ▼
Render Free (Go API, region Singapore)
  │  DATABASE_URL (Supabase pooler :6543)
  │  SUPABASE_URL + service role key (Storage REST)
  ▼
Supabase Free
  ├── PostgreSQL 17 (500 MB)
  └── Storage privat `unesarent` (1 GB)
```

Batas yang harus ditulis di laporan UAS:

| Layanan | Batas free tier | Dampak |
|---|---|---|
| Supabase DB | 500 MB | Cukup untuk ratusan user + ribuan transaksi teks. File tidak masuk DB. |
| Supabase Storage | 1 GB | KTM 1–3 MB/foto. Kompres foto listing < 300 KB. |
| Supabase egress | 5 GB/bulan | PDF dan foto di-cache; jangan streaming video. |
| Supabase | Pause setelah 7 hari tanpa aktivitas | Ping health tiap 6 jam dari GitHub Actions. |
| Supabase | Tanpa backup otomatis | Backup harian `pg_dump` ke GitHub Actions artifact. |
| Render Free | Spin down setelah 15 menit idle, cold start 30–60 detik | UptimeRobot ping tiap 10 menit. 750 jam/bulan cukup untuk 1 service 24/7. |
| Render Free | 512 MB RAM, 0.1 CPU | Jangan jalankan PDF generation berat bersamaan. |
| Vercel Hobby | Non-komersial, 100 GB transfer | Boleh untuk UAS. Jika komersial, pindah ke Cloudflare Pages. |
| Brevo | 300 email/hari | OTP saja, bukan newsletter. |

---

## 3. Aturan identitas & registrasi (jawaban untuk bagian Register)

### 3.1 Audit kondisi sekarang

| Yang sudah ada | Bukti |
|---|---|
| Email unik (case-insensitive) | `users_email_lower_key` di `0001_identity.up.sql` |
| NIM unik | `users_nim_key` di `0001_identity.up.sql` |
| NIM 11 digit + validasi FT/prodi | trigger `users_resolve_study_program` di `0006_hardening.up.sql` |
| Satu definisi "terverifikasi" | fungsi `is_ktm_verified(user_id)` di `0006_hardening.up.sql` |
| KTM terikat snapshot NIM | `ktm_verifications.nim_snapshot` + `is_ktm_verified` |
| Error email/NIM sudah terdaftar | `mapPostgresError` + `auth_handler.handleError` |

**Artinya:** aturan "email/NIM yang sudah terverifikasi tidak bisa didaftarkan lagi" **sudah 90% ditegakkan** oleh unique index. Yang belum ada:

1. Domain email belum dibatasi (masih `@gmail.com` boleh).
2. Belum ada bukti kepemilikan email (belum ada OTP).
3. Belum ada pesan/error yang membedakan "NIM sudah terverifikasi" vs "NIM terdaftar tapi belum verifikasi".
4. NIM masih bisa diubah lewat SQL setelah verified; `is_ktm_verified` akan langsung false, dan NIM lama bisa dipakai akun baru.
5. Belum ada prosedur admin untuk membebaskan NIM yang "dijajah" akun belum terverifikasi.

### 3.2 Aturan final

- **Email** = kanal login + OTP. Tidak membuktikan identitas. Unik, case-insensitive.
- **NIM** = anchor identitas. Unik, permanen setelah KTM verified.
- **Satu NIM hanya boleh punya satu akun terverifikasi, selamanya.**
- Registrasi publik hanya untuk `mahasiswa` dengan email `@mhs.unesa.ac.id`.
- Admin dibuat lewat CLI, bukan lewat registrasi publik. Email admin bebas.
- Alur: isi form → OTP email → akun dibuat → upload KTM → admin verifikasi.
- Jika NIM sudah dipakai akun **terverifikasi** → tolak `409 identity_already_verified`.
- Jika NIM sudah dipakai akun **belum terverifikasi** → tolak `409 nim_taken`, arahkan login ke akun itu atau hubungi admin.
- Admin boleh menghapus akun **belum terverifikasi** yang **tidak punya aktivitas** (listing/rental/pesan/laporan). Ini membebaskan NIM + email. Jika sudah ada aktivitas, admin hanya bisa suspend; penyelesaian manual.
- NIM tidak bisa diubah setelah ada KTM `verified`. Perubahan NIM hanya boleh setelah KTM dicabut (`revoked`), dan itu pun lewat admin + audit.
- Jangan buat tabel `verified_identity` terpisah. Sumber kebenaran tetap `users.nim` + `is_ktm_verified()`. Tabel baru hanya menambah risiko data drift.

### 3.3 Alur OTP final

```text
POST /api/auth/otp/request   { email, purpose: "register" }
  → 202 selalu (jangan bocorkan apakah email terdaftar)
  → kirim 6 digit, berlaku 10 menit, maks 5 percobaan

POST /api/auth/otp/verify    { email, purpose, code }
  → 200 { verification_token }   (berlaku 15 menit, sekali pakai)

POST /api/auth/register      { name, email, nim, password, verification_token }
  → 201 { token, user }          (langsung login, lanjut upload KTM)
```

- `purpose`: `register`, `reset_password`, `admin_login`.
- Kode disimpan sebagai `HMAC-SHA256(code, OTP_PEPPER)`, bukan plaintext.
- `verification_token` disimpan sebagai `SHA-256(token)` di `auth_otps.challenge_hash`.
- OTP request rate limit: 3 per 10 menit per email, 10 per jam per IP.
- OTP verify rate limit: 10 per 10 menit per IP.
- Kode salah 5× → baris di-consume, harus minta kode baru.
- Respons error tidak boleh membocorkan apakah email ada. Untuk `reset_password`, selalu 202.

### 3.4 File yang disentuh aturan ini

- `backend/internal/config/config.go` — domain allowlist, OTP, SMTP.
- `backend/internal/user/service.go` — validasi domain, pre-check NIM, konsumsi ticket.
- `backend/internal/user/store.go` + `store_postgres.go` — `FindNIMStatus`.
- `backend/internal/httpapi/auth_handler.go` — endpoint OTP + mapping error baru.
- `backend/internal/httpapi/router.go` — route OTP.
- `frontend/src/pages/Register.jsx` — 2 langkah (form → OTP).
- `frontend/src/pages/Login.jsx` — hapus demo, tambah lupa sandi + OTP admin.
- `frontend/src/auth/AuthContext.jsx` + `frontend/src/lib/authApi.js` — method OTP.
- `frontend/src/App.jsx` — route `/lupa-sandi`.

### 3.5 Acceptance test aturan identitas

1. Daftar `nama@gmail.com` → 422, field `email`: "Gunakan email resmi mahasiswa @mhs.unesa.ac.id."
2. Daftar `nama@mhs.unesa.ac.id` tanpa `verification_token` → 422.
3. Minta OTP, masukkan kode salah 5× → kode tidak berlaku; minta kode baru berhasil.
4. Daftar dengan NIM milik akun **verified** + email baru → 409 `identity_already_verified`.
5. Daftar dengan NIM milik akun **belum verified** → 409 `nim_taken`.
6. Daftar dengan email yang sudah ada → 409 `email_taken`.
7. Setelah KTM verified, `UPDATE users SET nim='...' WHERE id=...` → trigger menolak.
8. Admin hapus akun belum verified tanpa aktivitas → 200; NIM + email bisa dipakai daftar lagi.
9. Admin coba hapus akun yang punya rental → 409, akun tetap ada.
10. Login akun `suspended` → 403 `account_suspended`.

---

## 4. Fase 0 — Baseline

**Tujuan:** memastikan kondisi awal hijau sebelum mengedit.

```bash
cd /home/Wsntch3/Downloads/UNESARENT
cd backend && go build ./... && go test ./...
cd ../frontend && npm ci && npm run build
cd ..
```

**Acceptance:** build Go lulus, semua test Go lulus, `npm run build` lulus.

Tidak perlu membuat branch, commit, atau push. Cukup pastikan baseline hijau, lalu mulai mengedit file sesuai fase berikutnya.

---

## 5. Fase 1 — Hapus demo & pisahkan seed

### 5.1 Hapus demo dari frontend

**`frontend/src/pages/Login.jsx`**
- Hapus array `DEMO_ACCOUNTS`.
- Hapus seluruh blok `<details className="auth-demo"> ... </details>`.
- Hapus blok testimoni palsu di `<aside>`: `Dimas Arya`, `S1 Teknik Elektro '22 · 14 transaksi`, badge `Top Renter`.
- Ganti metrik palsu `420+ Alat terdaftar` menjadi copy netral, misalnya:
  - `KTM` / `Verifikasi mahasiswa`
  - `FT` / `Khusus Fakultas Teknik`
  - `0%` / `Biaya admin`
- Hapus CSS `.auth-demo`, `.demo-grid`, `.demo-btn` dari `frontend/src/index.css` jika tidak dipakai di tempat lain.
- Ganti label "Email Mahasiswa" menjadi "Email Mahasiswa UNESA".

**`frontend/src/pages/Register.jsx`**
- Ganti label `Email Mahasiswa / Pribadi` → `Email Mahasiswa UNESA`.
- Ganti hint menjadi: `Wajib berakhiran @mhs.unesa.ac.id. Kode verifikasi dikirim ke email ini.`
- Hapus teks `kode prodi yang tersedia di development` pada catatan NIM. Ganti: `Pastikan NIM dan program studi sesuai KTM.`
- Ubah indikator langkah menjadi: `1 Buat Akun`, `2 Verifikasi Email`, `3 Verifikasi KTM`.

Cek sisa kata demo/development:

```bash
rg -n "demo|Demo|development|admin\.dev@|admin\.ops@|Mahasiswa123|AdminDev123|AdminOps123" \
  frontend backend postgres README.md
```

Hanya boleh tersisa di `postgres/seed/dev_seed.sql` dan dokumentasi dev.

### 5.2 Pisahkan seed

Buat `postgres/seed/reference.sql` berisi **hanya**:
- `study_programs` (kode 120, S1 Teknik Informatika, FT, faculty_code 05).
- `categories` (6 kategori yang ada).
- `campus_locations` (5 lokasi yang ada).

Tidak boleh ada `INSERT INTO users`, `listings`, `rentals`, atau KTM.

Ubah `postgres/seed/dev_seed.sql`:
- Tetap seperti sekarang untuk lokal.
- Tambahkan komentar besar di baris pertama: `-- HANYA UNTUK LOKAL. JANGAN DIJALANKAN DI PRODUKSI.`

Ubah `Makefile`:

```make
db-seed: db-seed-reference

db-seed-reference:
	cd postgres && docker compose exec -T postgres sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -v ON_ERROR_STOP=1' < seed/reference.sql

db-seed-dev:
	cd postgres && docker compose exec -T postgres sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -v ON_ERROR_STOP=1' < seed/dev_seed.sql
```

Perbarui `postgres/README.md` dan `README.md`: seed produksi = `reference.sql`; demo = `dev_seed.sql`.

### 5.3 Acceptance Fase 1

```bash
make db-reset
make db-migrate
make db-seed
```

Lalu cek:

```sql
SELECT count(*) FROM users;            -- 0
SELECT count(*) FROM listings;         -- 0
SELECT count(*) FROM rentals;          -- 0
SELECT count(*) FROM study_programs;   -- 1
SELECT count(*) FROM categories;       -- 6
SELECT count(*) FROM campus_locations; -- 5
```

`npm run build` tetap lulus. Login page tidak lagi menampilkan akun demo.

---

## 6. Fase 2 — Auth produksi: domain, OTP, reset, admin bootstrap

### 6.1 Tambah konfigurasi

**`backend/internal/config/config.go`** — tambahkan field:

```go
type Config struct {
    // ... field lama ...
    PublicBaseURL          string
    AllowedStudentDomains  []string
    AdminEmails            []string
    AdminLoginOTP          bool

    SMTPHost string
    SMTPPort int
    SMTPUser string
    SMTPPass string
    SMTPFrom string

    OTPPepper             []byte
    OTPTTL                time.Duration
    OTPMaxAttempts        int
    RegistrationTicketTTL time.Duration

    TrustedProxy bool
}
```

Validasi di `Load()`:
- `APP_ENV=production` → `SMTP_HOST`, `SMTP_FROM`, `OTP_PEPPER` (min 32 karakter), `PUBLIC_BASE_URL` wajib.
- `ALLOWED_STUDENT_EMAIL_DOMAINS` default `mhs.unesa.ac.id`.
- `ADMIN_LOGIN_OTP` default `true` jika `APP_ENV=production`.
- `TRUSTED_PROXY` default `true` jika `APP_ENV=production` (Render/Cloudflare).
- Tolak `CORS_ORIGINS` yang mengandung `localhost` saat production.

Contoh `.env` produksi ada di **`docs/TUTORIAL_DEPLOY.md`** bagian “Referensi environment variable”.

### 6.2 Migrasi `0010_auth_production`

Buat `postgres/migrations/0010_auth_production.up.sql`:

```sql
-- 0010_auth_production.up.sql
-- OTP email, verifikasi email, dan penguncian NIM setelah KTM verified.

ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;

-- Akun lama (dev) dianggap sudah verifikasi email agar tidak terkunci.
UPDATE users SET email_verified_at = created_at WHERE email_verified_at IS NULL;

CREATE TABLE IF NOT EXISTS auth_otps (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email          VARCHAR(254) NOT NULL,
    user_id        UUID REFERENCES users(id) ON DELETE CASCADE,
    purpose        TEXT NOT NULL,
    code_hash      CHAR(64) NOT NULL,
    challenge_hash CHAR(64),
    attempts       SMALLINT NOT NULL DEFAULT 0,
    max_attempts   SMALLINT NOT NULL DEFAULT 5,
    expires_at     TIMESTAMPTZ NOT NULL,
    verified_at    TIMESTAMPTZ,
    consumed_at    TIMESTAMPTZ,
    request_ip     VARCHAR(45),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT auth_otps_purpose_check CHECK (
        purpose IN ('register', 'reset_password', 'admin_login')
    ),
    CONSTRAINT auth_otps_attempts_check CHECK (
        attempts >= 0 AND max_attempts BETWEEN 1 AND 10
    ),
    CONSTRAINT auth_otps_email_format CHECK (
        email ~* '^[^@[:space:]]+@[^@[:space:]]+\.[^@[:space:]]+$'
    )
);

CREATE INDEX IF NOT EXISTS auth_otps_lookup_idx
    ON auth_otps (lower(email), purpose, created_at DESC);
CREATE INDEX IF NOT EXISTS auth_otps_expiry_idx
    ON auth_otps (expires_at) WHERE consumed_at IS NULL;

-- NIM tidak boleh diubah selama masih ada KTM verified.
CREATE OR REPLACE FUNCTION users_block_verified_nim_change()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.nim IS DISTINCT FROM NEW.nim AND EXISTS (
        SELECT 1 FROM ktm_verifications k
        WHERE k.user_id = OLD.id AND k.status = 'verified'
    ) THEN
        RAISE EXCEPTION 'NIM tidak boleh diubah setelah KTM terverifikasi; cabut verifikasi terlebih dahulu'
            USING ERRCODE = '23514', CONSTRAINT = 'users_nim_locked_after_verification';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS users_block_verified_nim_change ON users;
CREATE TRIGGER users_block_verified_nim_change
BEFORE UPDATE OF nim ON users
FOR EACH ROW EXECUTE FUNCTION users_block_verified_nim_change();

COMMENT ON TABLE auth_otps IS 'OTP email sekali pakai untuk register, reset password, dan login admin.';
COMMENT ON COLUMN auth_otps.code_hash IS 'HMAC-SHA256(kode, OTP_PEPPER); bukan plaintext.';
COMMENT ON COLUMN auth_otps.challenge_hash IS 'SHA-256 token lanjutan; dikirim ke client sekali saja.';
```

Buat `0010_auth_production.down.sql`:

```sql
DROP TRIGGER IF EXISTS users_block_verified_nim_change ON users;
DROP FUNCTION IF EXISTS users_block_verified_nim_change();
DROP TABLE IF EXISTS auth_otps;
ALTER TABLE users DROP COLUMN IF EXISTS last_login_at;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified_at;
```

### 6.3 Package email

Buat `backend/internal/email/sender.go`:

```go
package email

import (
    "context"
    "fmt"
    "log/slog"
    "net/smtp"
    "strings"
)

type Sender interface {
    Send(ctx context.Context, to, subject, text string) error
}

type SMTPSender struct {
    addr string
    auth smtp.Auth
    from string
}

func NewSMTPSender(host string, port int, user, pass, from string) *SMTPSender {
    addr := fmt.Sprintf("%s:%d", host, port)
    var auth smtp.Auth
    if user != "" {
        auth = smtp.PlainAuth("", user, pass, host)
    }
    return &SMTPSender{addr: addr, auth: auth, from: from}
}

func (s *SMTPSender) Send(ctx context.Context, to, subject, text string) error {
    if err := ctx.Err(); err != nil {
        return err
    }
    msg := strings.Join([]string{
        "From: " + s.from,
        "To: " + to,
        "Subject: " + subject,
        "MIME-Version: 1.0",
        "Content-Type: text/plain; charset=UTF-8",
        "",
        text,
    }, "\r\n")
    return smtp.SendMail(s.addr, s.auth, s.from, []string{to}, []byte(msg))
}

// DevSender menulis OTP ke log. Hanya boleh dipakai saat APP_ENV=development.
type DevSender struct{}

func (DevSender) Send(_ context.Context, to, subject, text string) error {
    slog.Info("email development", "to", to, "subject", subject, "body", text)
    return nil
}
```

Buat `backend/internal/email/templates.go`:

```go
package email

import "fmt"

func OTPSubject() string { return "Kode verifikasi UNESA Rent" }

func OTPBody(code string, minutes int) string {
    return fmt.Sprintf(
        "Kode verifikasi kamu: %s\n\nKode ini berlaku %d menit dan hanya bisa dipakai sekali.\n"+
            "Jangan bagikan kode ini kepada siapa pun, termasuk admin UNESA Rent.\n\n"+
            "Jika kamu tidak meminta kode ini, abaikan email ini.",
        code, minutes,
    )
}
```

### 6.4 Package `authotp`

Buat `backend/internal/authotp/service.go` dengan kontrak:

```go
package authotp

type Purpose string

const (
    PurposeRegister      Purpose = "register"
    PurposeResetPassword Purpose = "reset_password"
    PurposeAdminLogin    Purpose = "admin_login"
)

type Service struct {
    store       Store
    mail        email.Sender
    pepper      []byte
    ttl         time.Duration
    maxAttempts int
    now         func() time.Time
}

func NewService(store Store, mail email.Sender, pepper []byte, ttl time.Duration, maxAttempts int) *Service

// Request membuat kode baru, menonaktifkan kode lama, lalu mengirim email.
func (s *Service) Request(ctx context.Context, email string, purpose Purpose, ip string) error

// Verify memeriksa kode. Jika benar, mengembalikan ticket mentah sekali pakai.
func (s *Service) Verify(ctx context.Context, email string, purpose Purpose, code string) (ticket string, err error)

// ConsumeTicket dipakai Register/ResetPassword untuk menukar ticket menjadi aksi.
func (s *Service) ConsumeTicket(ctx context.Context, email string, purpose Purpose, ticket string) error

// CreateChallenge dipakai login admin: membuat kode + challenge sebelum OTP dikirim.
func (s *Service) CreateChallenge(ctx context.Context, email, userID string, ip string) (challenge string, err error)

// VerifyChallenge dipakai login admin: challenge + kode -> sukses.
func (s *Service) VerifyChallenge(ctx context.Context, challenge, code string) (email string, err error)
```

Aturan implementasi:
- Kode 6 digit dari `crypto/rand`, format `%06d`.
- `code_hash = hex(HMAC-SHA256(pepper, code))`.
- `ticket`/`challenge` = 32 byte random, base64url. Simpan `hex(SHA-256(raw))`.
- `Request`: dalam satu transaksi, `UPDATE auth_otps SET consumed_at=now() WHERE lower(email)=lower($1) AND purpose=$2 AND consumed_at IS NULL`, lalu `INSERT`.
- Cek rate limit per email: `SELECT count(*) FROM auth_otps WHERE lower(email)=lower($1) AND purpose=$2 AND created_at > now() - interval '10 minutes'`; jika ≥3, return `ErrTooManyRequests`.
- `Verify`: `SELECT ... FOR UPDATE` baris terbaru yang belum consumed dan belum expired; bandingkan hash dengan `hmac.Equal`; jika salah, `attempts+1`; jika `attempts >= max_attempts`, set `consumed_at`.
- Jika benar: set `verified_at=now()`, `challenge_hash=sha256(ticket)`, kembalikan ticket.
- `ConsumeTicket`: `UPDATE ... SET consumed_at=now() WHERE lower(email)=lower($1) AND purpose=$2 AND challenge_hash=$3 AND verified_at IS NOT NULL AND consumed_at IS NULL AND expires_at > now() RETURNING id`; jika 0 baris → `ErrInvalidTicket`.
- `CreateChallenge`: insert baris `admin_login` dengan `user_id`, `code_hash`, `challenge_hash`, kirim email, return challenge mentah.
- `VerifyChallenge`: cari baris berdasarkan `challenge_hash`, cek kode, set `consumed_at`, return email.

Error yang diekspor: `ErrTooManyRequests`, `ErrInvalidCode`, `ErrExpired`, `ErrInvalidTicket`.

### 6.5 Ubah user service & store

**`backend/internal/user/model.go`**

```go
type User struct {
    // ... field lama ...
    EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
    LastLoginAt     *time.Time `json:"last_login_at,omitempty"`
}
```

**`backend/internal/user/store.go`**

```go
type NIMStatus struct {
    Exists   bool
    Verified bool
    UserID   string
}

type Store interface {
    // ... method lama ...
    FindNIMStatus(ctx context.Context, nim string) (NIMStatus, error)
    MarkLogin(ctx context.Context, userID string) error
    UpdatePassword(ctx context.Context, userID, passwordHash string) error
}
```

- `PostgresStore.FindNIMStatus`:
  ```sql
  SELECT u.id::text, is_ktm_verified(u.id)
  FROM users u WHERE u.nim = $1
  ```
  Jika `pgx.ErrNoRows` → `NIMStatus{Exists:false}`.
- `MemoryStore.FindNIMStatus`: cari di `byID`; `Verified=false` (mode JSON tidak punya KTM).
- `MarkLogin`: `UPDATE users SET last_login_at=now() WHERE id=$1::uuid`.
- `UpdatePassword`: `UPDATE users SET password_hash=$2, token_version=token_version+1, updated_at=now() WHERE id=$1::uuid`.

**`backend/internal/user/store_postgres.go`**
- Tambahkan `email_verified_at, last_login_at` ke `userColumns` (setelah `updated_at`), dan scan ke `u.EmailVerifiedAt`, `u.LastLoginAt` di `scanUser`.
- `Create`:
  ```go
  row := s.pool.QueryRow(ctx, `
      INSERT INTO users (name, email, password_hash, role, nim, status, email_verified_at)
      VALUES ($1,$2,$3,$4,$5,$6,$7)
      RETURNING `+userColumns, ..., u.EmailVerifiedAt)
  ```
  Jika `u.EmailVerifiedAt == nil`, kirim `nil`.

**`backend/internal/user/service.go`**
- Tambah:
  ```go
  var ErrIdentityVerified = errors.New("identitas sudah terverifikasi")
  ```
- `RegisterInput` tambah `VerificationToken string \`json:"verification_token"\``.
- `Register`:
  1. Trim + normalize.
  2. `validateRegister` + cek domain:
     ```go
     if !s.isAllowedStudentEmail(in.Email) {
         fields["email"] = "Gunakan email resmi mahasiswa @mhs.unesa.ac.id"
     }
     ```
  3. `s.otp.ConsumeTicket(ctx, in.Email, authotp.PurposeRegister, in.VerificationToken)`; jika gagal → `ValidationError{"verification_token": "Kode verifikasi tidak valid atau sudah kedaluwarsa. Minta kode baru."}`.
  4. `status, _ := s.store.FindNIMStatus(ctx, in.NIM)`:
     - `status.Exists && status.Verified` → `ErrIdentityVerified`
     - `status.Exists` → `ErrNIMTaken`
  5. `s.store.FindByEmail`; jika ketemu → `ErrEmailTaken`.
  6. Set `newUser.EmailVerifiedAt = ptr(time.Now())`.
  7. `Create`.
- `Service` struct tambah `otp *authotp.Service` dan `allowedDomains []string`; ubah `NewService(store, tokens, otp, allowedDomains)`.
- `Login`: setelah password benar dan status active, `s.store.MarkLogin(ctx, found.ID)`. Jika `found.Role == RoleAdminDev && adminLoginOTP`, return `AuthResult{OTPRequired:true, Challenge:...}` (lihat 6.6).
- `ResetPassword(ctx, email, ticket, newPassword)`:
  - validasi password,
  - `otp.ConsumeTicket(email, reset_password, ticket)`,
  - `FindByEmail`,
  - hash + `UpdatePassword`.

**Catatan kompatibilitas:** `NewService` dipanggil di `cmd/server/main.go`. Sesuaikan.

### 6.6 Handler & route

**`backend/internal/httpapi/auth_handler.go`**

Tambah request/response:

```go
type otpRequestBody struct {
    Email   string `json:"email"`
    Purpose string `json:"purpose"`
}
type otpVerifyBody struct {
    Email   string `json:"email"`
    Purpose string `json:"purpose"`
    Code    string `json:"code"`
}
type loginOTPBody struct {
    Challenge string `json:"challenge"`
    Code      string `json:"code"`
}
type resetPasswordBody struct {
    Email       string `json:"email"`
    Ticket      string `json:"ticket"`
    NewPassword string `json:"new_password"`
}
```

Endpoint:
- `POST /api/auth/otp/request` → selalu `202 {"status":"sent"}`; jangan bocorkan email tidak ada.
- `POST /api/auth/otp/verify` → `200 {"verification_token":"..."}`.
- `POST /api/auth/login` → jika admin OTP: `200 {"otp_required":true,"challenge":"...","expires_at":"..."}`; selain itu JWT seperti biasa.
- `POST /api/auth/login/otp` → `200` JWT.
- `POST /api/auth/password/forgot` → selalu `202`.
- `POST /api/auth/password/reset` → `200 {"status":"ok"}`.

Mapping error baru di `handleError`:

```go
case errors.Is(err, user.ErrIdentityVerified):
    writeError(w, http.StatusConflict, "identity_already_verified",
        "NIM ini sudah terverifikasi pada akun lain. Masuk dengan akun tersebut atau hubungi admin.")
case errors.Is(err, user.ErrNIMTaken):
    writeError(w, http.StatusConflict, "nim_taken",
        "NIM sudah terdaftar tetapi belum terverifikasi. Jika ini NIM kamu, masuk dan selesaikan verifikasi KTM atau hubungi admin.")
case errors.Is(err, authotp.ErrTooManyRequests):
    writeError(w, http.StatusTooManyRequests, "too_many_requests",
        "Terlalu banyak permintaan kode. Coba lagi dalam 10 menit.")
case errors.Is(err, authotp.ErrInvalidCode):
    writeError(w, http.StatusUnauthorized, "invalid_otp", "Kode verifikasi salah atau sudah kedaluwarsa.")
```

**`backend/internal/httpapi/router.go`**

```go
otpLimit := RateLimit(10, time.Minute)
mux.Handle("POST /api/auth/otp/request", otpLimit(http.HandlerFunc(authHandler.RequestOTP)))
mux.Handle("POST /api/auth/otp/verify", otpLimit(http.HandlerFunc(authHandler.VerifyOTP)))
mux.Handle("POST /api/auth/login/otp", otpLimit(http.HandlerFunc(authHandler.LoginOTP)))
mux.Handle("POST /api/auth/password/forgot", otpLimit(http.HandlerFunc(authHandler.ForgotPassword)))
mux.Handle("POST /api/auth/password/reset", otpLimit(http.HandlerFunc(authHandler.ResetPassword)))
```

**`backend/cmd/server/main.go`**
- Buat `mailer`:
  ```go
  var mailer email.Sender
  if cfg.SMTPHost == "" {
      if cfg.Env != "development" { log.Fatal(...) }
      mailer = email.DevSender{}
  } else {
      mailer = email.NewSMTPSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)
  }
  ```
- Buat `otpSvc := authotp.NewService(authotp.NewPostgresStore(pool), mailer, cfg.OTPPepper, cfg.OTPTTL, cfg.OTPMaxAttempts)`.
- `user.NewService(store, tokens, otpSvc, cfg.AllowedStudentDomains, cfg.AdminLoginOTP)`.

### 6.7 Admin bootstrap

Buat `backend/cmd/createadmin/main.go`:

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "os"
    "strings"
    "time"

    "unesarent/backend/internal/auth"
    "unesarent/backend/internal/config"
    "unesarent/backend/internal/db"
)

func main() {
    email := flag.String("email", os.Getenv("ADMIN_EMAIL"), "email admin")
    password := flag.String("password", os.Getenv("ADMIN_PASSWORD"), "password admin (min 12 karakter)")
    name := flag.String("name", "Administrator UNESA Rent", "nama admin")
    flag.Parse()

    if !strings.Contains(*email, "@") || len(*password) < 12 {
        log.Fatal("email valid dan password minimal 12 karakter wajib diisi")
    }

    cfg, err := config.Load()
    if err != nil { log.Fatal(err) }
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    pool, err := db.New(ctx, cfg.DatabaseURL)
    if err != nil { log.Fatal(err) }
    defer pool.Close()

    hash, err := auth.HashPassword(*password)
    if err != nil { log.Fatal(err) }

    _, err = pool.Exec(ctx, `
        INSERT INTO users (name, email, password_hash, role, status, email_verified_at)
        VALUES ($1, lower($2), $3, 'admin_dev', 'active', now())
        ON CONFLICT (lower(email)) DO UPDATE
           SET name = EXCLUDED.name,
               password_hash = EXCLUDED.password_hash,
               role = 'admin_dev',
               status = 'active',
               email_verified_at = now(),
               token_version = users.token_version + 1,
               updated_at = now()
    `, *name, *email, hash)
    if err != nil { log.Fatal(err) }

    fmt.Println("admin siap:", strings.ToLower(*email))
}
```

Jalankan di lokal yang mengarah ke Supabase:

```bash
cd backend
DATABASE_URL='postgres://...:5432/postgres?sslmode=require' \
APP_ENV=development \
JWT_SECRET='...' \
go run ./cmd/createadmin -email 'admin@domainmu.com' -password 'PasswordAdminYangKuat123!'
```

Catatan: `APP_ENV=development` hanya untuk melewati validasi env produksi saat bootstrap. Jangan pakai untuk server.

### 6.8 Frontend

**`frontend/src/lib/authApi.js`** tambah:

```js
requestOtp: (payload) => request('/api/auth/otp/request', { method: 'POST', body: payload, auth: false }),
verifyOtp: (payload) => request('/api/auth/otp/verify', { method: 'POST', body: payload, auth: false }),
loginOtp: (payload) => request('/api/auth/login/otp', { method: 'POST', body: payload, auth: false }),
forgotPassword: (payload) => request('/api/auth/password/forgot', { method: 'POST', body: payload, auth: false }),
resetPassword: (payload) => request('/api/auth/password/reset', { method: 'POST', body: payload, auth: false }),
```

**`frontend/src/auth/AuthContext.jsx`** tambah `requestOtp`, `verifyOtp`, `loginOtp`, `forgotPassword`, `resetPassword` dengan pola `pending`/`error` yang sama.

**`frontend/src/pages/Register.jsx`** — alur 2 langkah:
- State `step: 'form' | 'otp'`.
- Validasi email: `/^[^\s@]+@mhs\.unesa\.ac\.id$/i`. Pesan: `Gunakan email resmi mahasiswa @mhs.unesa.ac.id.`
- Submit form → `requestOtp({ email, purpose: 'register' })` → `step='otp'`.
- Layar OTP: input 6 digit (`inputMode="numeric"`, `maxLength=6`), tombol `Verifikasi & Buat Akun`, tombol `Kirim ulang kode` dengan cooldown 60 detik.
- Verifikasi → `verifyOtp` → `register({ ...form, verification_token })` → `navigate('/profil')`.
- Jika `register` gagal karena ticket kedaluwarsa, tampilkan tombol kembali ke langkah OTP dan minta kode baru.
- Copy penting:
  - Judul OTP: `Verifikasi Email Mahasiswa`
  - Subtitle: `Kami mengirim 6 digit kode ke {email}.`
  - Sukses: `Email terverifikasi. Lanjutkan verifikasi KTM di halaman profil.`

**`frontend/src/pages/Login.jsx`**
- Hapus demo (Fase 1).
- Tambah link `Lupa kata sandi?` ke `/lupa-sandi`.
- Jika respons login `otp_required`, tampilkan layar OTP admin: input kode, tombol `Masuk sebagai Admin`, lalu `loginOtp({ challenge, code })`.
- Copy: `Verifikasi Admin` / `Masukkan kode yang dikirim ke email admin.`

**`frontend/src/pages/ForgotPassword.jsx`** (baru)
- Langkah 1: input email → `forgotPassword`.
- Langkah 2: input OTP → `verifyOtp({ purpose: 'reset_password' })` → dapat ticket.
- Langkah 3: password baru + konfirmasi → `resetPassword`.
- Copy selalu netral: `Jika email terdaftar, kode verifikasi telah dikirim.`

**`frontend/src/App.jsx`**
- Import `ForgotPassword`.
- Tambah `if (pathname === '/lupa-sandi') return <ForgotPassword />`.

### 6.9 Acceptance Fase 2

Jalankan migrasi + server lokal:

```bash
make db-migrate
cd backend && go test ./... && go run ./cmd/server
```

Uji manual/curl:
1. `POST /api/auth/otp/request` email `@gmail.com` → 202 tetapi tidak mengirim (domain ditolak saat register). Untuk UX, frontend sudah menolak lebih awal.
2. `POST /api/auth/otp/request` email `@mhs.unesa.ac.id` → cek log `DevSender`, kode muncul.
3. `POST /api/auth/otp/verify` kode benar → dapat `verification_token`.
4. `POST /api/auth/register` dengan token → 201 + JWT.
5. Ulangi dengan NIM sama → 409 `nim_taken`.
6. Verifikasi KTM lewat admin, lalu coba daftar NIM sama dengan email baru → 409 `identity_already_verified`.
7. `UPDATE users SET nim='...'` untuk user verified → error trigger.
8. Login admin → `otp_required`, verifikasi → JWT.
9. Lupa sandi end-to-end → password baru bisa dipakai, token lama invalid (token_version naik).

---

## 7. Fase 3 — Supabase Storage

### 7.1 Bucket

Di Supabase Dashboard → Storage → New bucket:
- Name: `unesarent`
- Public: **OFF** (private)
- File size limit: 10 MB
- Allowed MIME: `image/jpeg,image/png,application/pdf`

Tidak perlu RLS policy untuk client; backend memakai `service_role` yang bypass RLS. Jangan pernah mengirim service role key ke frontend.

### 7.2 Adapter

Buat `backend/internal/objectstore/supabase.go`:

```go
package objectstore

type Supabase struct {
    baseURL    string
    serviceKey string
    bucket     string
    http       *http.Client
}

func NewSupabase(baseURL, serviceKey, bucket string) *Supabase

func (s *Supabase) Put(ctx context.Context, key string, data []byte, contentType string) error
func (s *Supabase) Get(ctx context.Context, key string) (io.ReadCloser, error)
func (s *Supabase) Delete(ctx context.Context, key string) error
```

REST:
- `PUT {SUPABASE_URL}/storage/v1/object/{bucket}/{key}` dengan header `Authorization: Bearer <service_role>`, `Content-Type: <mime>`, `x-upsert: false`. Body = bytes.
- `GET {SUPABASE_URL}/storage/v1/object/{bucket}/{key}` dengan header Authorization. Return `resp.Body`.
- `DELETE {SUPABASE_URL}/storage/v1/object/{bucket}/{key}` dengan header Authorization.
- 404 → error yang bisa dipetakan ke `ktm.ErrFilePurged` / `catalog.ErrPhotoNotFound`.
- Jangan log service key. Jangan follow redirect ke host lain.

Buat wrapper:
- `backend/internal/ktm/storage_supabase.go`:
  ```go
  type SupabaseStorage struct { client *objectstore.Supabase }
  func NewSupabaseStorage(client *objectstore.Supabase) *SupabaseStorage
  ```
  Validasi key dengan `storageKeyPattern` yang sudah ada.
- `backend/internal/catalog/storage_supabase.go`: sama, validasi `photoKeyPattern`.

### 7.3 Wiring

**`config.Config`** tambah:
```go
StorageDriver        string // "local" | "supabase"
SupabaseURL          string
SupabaseServiceKey   string
SupabaseBucket       string
```

**`cmd/server/main.go`**:
```go
var ktmFiles ktm.Storage
var listingFiles catalog.Storage
switch cfg.StorageDriver {
case "supabase":
    obj := objectstore.NewSupabase(cfg.SupabaseURL, cfg.SupabaseServiceKey, cfg.SupabaseBucket)
    ktmFiles = ktm.NewSupabaseStorage(obj)
    listingFiles = catalog.NewSupabaseStorage(obj)
default:
    ktmFiles, _ = ktm.NewLocalStorage(cfg.KTMStorageDir)
    listingFiles, _ = catalog.NewLocalStorage(cfg.ListingStorageDir)
}
```

### 7.4 Acceptance Fase 3

1. Set `STORAGE_DRIVER=supabase` + kredensial.
2. Upload KTM → object muncul di bucket `unesarent` dengan key `ktm/<user-id>/<32hex>.jpg`.
3. `GET /api/ktm/{id}/file` mengembalikan gambar; admin bisa buka.
4. Upload foto listing → key `listings/<listing-id>/<32hex>.jpg`; katalog menampilkan foto.
5. Hapus foto → object hilang dari bucket.
6. Matikan service role key → endpoint file mengembalikan 500 yang aman, bukan crash.
7. Pastikan tidak ada URL publik langsung ke bucket di response API.

---

## 8. Fase 4 — Chat terenkripsi & bukti

### 8.1 Migrasi `0011_chat_encryption`

Buat `postgres/migrations/0011_chat_encryption.up.sql`:

```sql
-- 0011_chat_encryption.up.sql
-- Isi chat dienkripsi AES-256-GCM. Kolom body lama dipertahankan sementara
-- untuk kompatibilitas; command backfill akan mengosongkannya.

ALTER TABLE rental_messages
    ADD COLUMN IF NOT EXISTS body_ciphertext BYTEA,
    ADD COLUMN IF NOT EXISTS body_nonce BYTEA,
    ADD COLUMN IF NOT EXISTS key_version SMALLINT,
    ADD COLUMN IF NOT EXISTS body_length INTEGER,
    ADD COLUMN IF NOT EXISTS body_sha256 CHAR(64);

ALTER TABLE rental_messages ALTER COLUMN body DROP NOT NULL;

ALTER TABLE rental_messages DROP CONSTRAINT IF EXISTS rental_messages_body_length;
ALTER TABLE rental_messages ADD CONSTRAINT rental_messages_body_length CHECK (
    (body IS NULL OR char_length(btrim(body)) BETWEEN 1 AND 2000)
    AND (body_length IS NULL OR body_length BETWEEN 1 AND 2000)
);

ALTER TABLE rental_messages DROP CONSTRAINT IF EXISTS rental_messages_payload_check;
ALTER TABLE rental_messages ADD CONSTRAINT rental_messages_payload_check CHECK (
    (body IS NOT NULL AND body_ciphertext IS NULL AND body_nonce IS NULL AND key_version IS NULL)
    OR
    (body IS NULL AND body_ciphertext IS NOT NULL AND body_nonce IS NOT NULL
        AND key_version IS NOT NULL AND body_length IS NOT NULL AND body_sha256 IS NOT NULL)
);

CREATE TABLE IF NOT EXISTS chat_access_log (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id   UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    rental_id  UUID NOT NULL REFERENCES rentals(id) ON DELETE CASCADE,
    message_id UUID REFERENCES rental_messages(id) ON DELETE SET NULL,
    action     TEXT NOT NULL,
    reason     TEXT NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chat_access_log_action_check CHECK (
        action IN ('reveal', 'export_transcript')
    ),
    CONSTRAINT chat_access_log_reason_length CHECK (
        char_length(btrim(reason)) BETWEEN 10 AND 1000
    )
);

CREATE INDEX IF NOT EXISTS chat_access_log_admin_idx
    ON chat_access_log (admin_id, created_at DESC);
CREATE INDEX IF NOT EXISTS chat_access_log_rental_idx
    ON chat_access_log (rental_id, created_at DESC);

CREATE OR REPLACE FUNCTION chat_access_log_append_only()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'chat_access_log bersifat append-only' USING ERRCODE = '23514';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS chat_access_log_append_only ON chat_access_log;
CREATE TRIGGER chat_access_log_append_only
BEFORE UPDATE OR DELETE ON chat_access_log
FOR EACH ROW EXECUTE FUNCTION chat_access_log_append_only();

COMMENT ON TABLE chat_access_log IS 'Audit setiap kali admin membuka/mengekspor isi chat.';
```

Down: drop `chat_access_log`, hapus constraint payload, hapus kolom baru, set `body NOT NULL` hanya jika tidak ada baris terenkripsi.

### 8.2 Package `chatcrypto`

Buat `backend/internal/chatcrypto/cipher.go`:

```go
package chatcrypto

type Cipher struct {
    keys   map[int][]byte
    active int
}

func New(keys map[int][]byte, active int) (*Cipher, error)

func (c *Cipher) Encrypt(plaintext string) (ciphertext, nonce []byte, version int, err error)
func (c *Cipher) Decrypt(ciphertext, nonce []byte, version int) (string, error)
```

- AES-256-GCM, nonce 12 byte dari `crypto/rand`.
- `version` = `active`.
- Key dari env JSON: `CHAT_ENC_KEYS={"1":"<base64 32 byte>","2":"..."}`, `CHAT_ENC_ACTIVE_VERSION=1`.
- Validasi: setiap key 32 byte; active version ada.
- Jangan pernah log plaintext atau key.

### 8.3 Ubah rental service

**`backend/internal/rental/model.go`** — `Message` tambah metadata (hanya untuk admin):
```go
BodyLength int    `json:"body_length,omitempty"`
BodySHA256 string `json:"body_sha256,omitempty"`
```
Jangan kirim `Body` ke admin metadata.

**`backend/internal/rental/service.go`**
- `Service` tambah `cipher *chatcrypto.Cipher`.
- `SendMessage`:
  ```go
  ct, nonce, version, err := s.cipher.Encrypt(body)
  sum := sha256.Sum256([]byte(otpPepper + body)) // atau SHA-256(body) saja
  return s.store.CreateMessage(ctx, rentalID, userID, ct, nonce, version, len([]rune(body)), hex(sum[:]))
  ```
- `Messages` (peserta): ambil ciphertext, decrypt, isi `Message.Body`.
- Tambah method admin:
  ```go
  func (s *Service) AdminMessageMetadata(ctx, rentalID string, limit, offset int) ([]Message, int, error)
  func (s *Service) AdminRevealMessage(ctx, adminID, rentalID, messageID, reason, ip, ua string) (string, error)
  func (s *Service) AdminExportTranscript(ctx, adminID, rentalID, reason, ip, ua string) ([]byte, error)
  ```
- `AdminMessageMetadata`: query tanpa `body_ciphertext`; isi `Sender`, `CreatedAt`, `BodyLength`, `BodySHA256`.
- `AdminRevealMessage`: validasi reason ≥10 karakter, ambil ciphertext, decrypt, insert `chat_access_log`, return plaintext.
- `AdminExportTranscript`: decrypt semua pesan rental, susun teks/PDF, simpan ke `rental_documents` tipe `other` dengan `visibility='admin'`, insert `chat_access_log` action `export_transcript`, return bytes.

**`backend/internal/rental/store.go` + `store_postgres.go`**
- Ubah `CreateMessage` menerima ciphertext/nonce/version/length/hash.
- Tambah `AdminMessages`, `AdminMessageByID`, `InsertChatAccessLog`.

### 8.4 Endpoint admin

**`backend/internal/httpapi/rental_handler.go`**

```text
GET  /api/admin/rentals/{id}/messages            → metadata saja
POST /api/admin/rentals/{id}/messages/{msgID}/reveal
     body: { "reason": "...", "password": "..." }
POST /api/admin/rentals/{id}/transcript/export
     body: { "reason": "...", "password": "..." }
```

- `reveal` dan `export` wajib verifikasi ulang password admin (`auth.CheckPassword`).
- Reason minimal 10 karakter; maksimal 1000.
- Catat IP dari `clientIP(r)` (lihat 12.1) dan `User-Agent`.
- Return `403 forbidden` jika bukan admin; `401 invalid_password` jika password salah.
- Jangan pernah mengembalikan isi chat di endpoint metadata.

**`backend/internal/httpapi/router.go`** daftarkan route di blok rental dengan `requireAuth(requireAdmin(...))`.

### 8.5 Frontend

- `frontend/src/pages/Messages.jsx` dan `RentalDetail.jsx`: chat peserta tetap seperti sekarang (body didekripsi server).
- `frontend/src/pages/Admin.jsx` atau halaman baru `AdminRentals.jsx`: tab "Chat" menampilkan metadata: pengirim, waktu, panjang pesan, hash pendek.
- Tombol `Buka isi pesan` membuka modal: alasan wajib, password admin, konfirmasi. Setelah reveal, tampilkan plaintext + banner `Akses ini tercatat di audit log.`
- Tombol `Ekspor transkrip` menghasilkan unduhan file dan tercatat.
- Jangan tampilkan plaintext di list; hanya setelah reveal.

### 8.6 Backfill data dev (opsional, hanya lokal)

Buat `backend/cmd/encryptchat/main.go`:
- Ambil baris `rental_messages WHERE body IS NOT NULL`.
- Encrypt dengan `chatcrypto`, update `body_ciphertext/body_nonce/key_version/body_length/body_sha256`, set `body=NULL`.
- Jalankan setelah migrasi 0011 di DB dev.

Produksi kosong tidak butuh backfill.

### 8.7 Acceptance Fase 4

1. Kirim pesan → di DB `body` NULL, `body_ciphertext` terisi.
2. Peserta lain membuka chat → plaintext benar.
3. Admin buka metadata → tidak ada `body`.
4. Admin reveal tanpa alasan → 422.
5. Admin reveal dengan alasan + password benar → plaintext tampil; `chat_access_log` bertambah.
6. Admin reveal password salah → 401; tidak ada log sukses.
7. Export transkrip → file tersimpan di bucket + log.
8. Restart server → pesan lama masih bisa didekripsi (key env sama).
9. Ganti `CHAT_ENC_ACTIVE_VERSION=2` → pesan baru versi 2, pesan lama versi 1 tetap terbaca.

---

## 9. Fase 5 — COD & dokumen

### 9.1 Migrasi `0012_cod_documents`

Buat `postgres/migrations/0012_cod_documents.up.sql`:

```sql
-- 0012_cod_documents.up.sql
-- Dokumen per rental + state COD.

CREATE TABLE IF NOT EXISTS rental_documents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rental_id       UUID NOT NULL REFERENCES rentals(id) ON DELETE CASCADE,
    type            TEXT NOT NULL,
    storage_key     TEXT NOT NULL UNIQUE,
    mime_type       TEXT NOT NULL,
    file_size_bytes BIGINT NOT NULL,
    checksum_sha256 CHAR(64),
    uploaded_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    visibility      TEXT NOT NULL DEFAULT 'participants',
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT rental_documents_type_check CHECK (
        type IN ('agreement', 'handover_proof', 'return_proof', 'cod_receipt', 'other')
    ),
    CONSTRAINT rental_documents_mime_check CHECK (
        mime_type IN ('application/pdf', 'image/jpeg', 'image/png')
    ),
    CONSTRAINT rental_documents_size_check CHECK (
        file_size_bytes > 0 AND file_size_bytes <= 10 * 1024 * 1024
    ),
    CONSTRAINT rental_documents_visibility_check CHECK (
        visibility IN ('participants', 'admin')
    )
);

CREATE INDEX IF NOT EXISTS rental_documents_rental_idx
    ON rental_documents (rental_id, type, created_at DESC);

CREATE TABLE IF NOT EXISTS rental_cod (
    rental_id             UUID PRIMARY KEY REFERENCES rentals(id) ON DELETE CASCADE,
    agreed_at             TIMESTAMPTZ,
    handover_at           TIMESTAMPTZ,
    handover_by           UUID REFERENCES users(id) ON DELETE SET NULL,
    handover_confirmed_at TIMESTAMPTZ,
    handover_confirmed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    return_at             TIMESTAMPTZ,
    return_by             UUID REFERENCES users(id) ON DELETE SET NULL,
    return_confirmed_at   TIMESTAMPTZ,
    return_confirmed_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    amount_due            INTEGER CHECK (amount_due IS NULL OR amount_due >= 0),
    amount_paid           INTEGER CHECK (amount_paid IS NULL OR amount_paid >= 0),
    payment_method        TEXT CHECK (payment_method IS NULL OR payment_method IN ('cash', 'transfer')),
    settled_at            TIMESTAMPTZ,
    notes                 TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

DROP TRIGGER IF EXISTS rental_cod_set_updated_at ON rental_cod;
CREATE TRIGGER rental_cod_set_updated_at
BEFORE UPDATE ON rental_cod
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE rental_documents IS 'Arsip dokumen privat per rental; file di Supabase Storage.';
COMMENT ON TABLE rental_cod IS 'State serah terima dan pembayaran COD per rental.';
```

Down: drop `rental_cod`, `rental_documents`.

### 9.2 Generator PDF

Tambah dependency:

```bash
cd backend
go get github.com/go-pdf/fpdf@latest
```

Buat `backend/internal/rental/pdf.go`:

```go
func BuildAgreementPDF(r Rental, ownerNIM, renterNIM, location string) ([]byte, error)
func BuildReceiptPDF(r Rental, cod CODSummary) ([]byte, error)
```

Isi minimal:
- Judul: `SURAT PERJANJIAN SEWA BARANG`
- Nomor: `SP-<tahun>-<8 karakter pertama rental ID>`
- Identitas penyedia dan penyewa: nama, NIM, program studi.
- Barang: judul listing, deskripsi singkat.
- Periode sewa, jumlah hari, harga per hari, total.
- Lokasi COD dan ketentuan: cek fisik, KTM, pengembalian tepat waktu, tanggung jawab kerusakan.
- Kolom tanda tangan: nama + timestamp (bukan tanda tangan gambar).
- Footer: `Dokumen dibuat otomatis oleh UNESA Rent.`
- Sanitasi karakter non-Latin1 (fpdf core font): ganti dengan `?` atau transliterasi.

### 9.3 Service & endpoint

**Model:**
```go
type Document struct {
    ID          string    `json:"id"`
    RentalID    string    `json:"rental_id"`
    Type        string    `json:"type"`
    MimeType    string    `json:"mime_type"`
    SizeBytes   int64     `json:"size_bytes"`
    UploadedBy  *Party    `json:"uploaded_by,omitempty"`
    Visibility  string    `json:"visibility"`
    CreatedAt   time.Time `json:"created_at"`
    DownloadURL string    `json:"download_url,omitempty"`
}

type CODState struct {
    AgreedAt            *time.Time `json:"agreed_at,omitempty"`
    HandoverAt          *time.Time `json:"handover_at,omitempty"`
    HandoverConfirmedAt *time.Time `json:"handover_confirmed_at,omitempty"`
    ReturnAt            *time.Time `json:"return_at,omitempty"`
    ReturnConfirmedAt   *time.Time `json:"return_confirmed_at,omitempty"`
    AmountDue           *int       `json:"amount_due,omitempty"`
    AmountPaid          *int       `json:"amount_paid,omitempty"`
    PaymentMethod       *string    `json:"payment_method,omitempty"`
    SettledAt           *time.Time `json:"settled_at,omitempty"`
    Notes               *string    `json:"notes,omitempty"`
}
```

**Endpoint peserta:**

```text
GET  /api/rentals/{id}/documents
GET  /api/rentals/{id}/documents/{docID}/file
POST /api/rentals/{id}/cod/handover          (owner, multipart photo + note)
POST /api/rentals/{id}/cod/handover/confirm  (renter)
POST /api/rentals/{id}/cod/return            (renter, multipart photo + note)
POST /api/rentals/{id}/cod/return/confirm    (owner)
POST /api/rentals/{id}/cod/settle            (owner, amount_paid, method, notes)
GET  /api/rentals/{id}/cod
```

**Endpoint admin:**

```text
GET  /api/admin/rentals/{id}/documents
POST /api/admin/rentals/{id}/documents/{docID}/download
     body: { "reason": "..." }
```

- Semua download admin wajib alasan + audit `moderation_actions` action `download_document`.
- File tidak pernah public; selalu stream lewat backend.

**Aturan transisi:**
- Saat rental `approved`: buat baris `rental_cod` (`agreed_at=now()`), generate agreement PDF, simpan dokumen tipe `agreement`.
- `Complete` hanya boleh jika `handover_confirmed_at IS NOT NULL` dan `return_confirmed_at IS NOT NULL` (jika `COD_REQUIRE_RETURN=true`, default true).
- Jika belum, return `409 cod_incomplete` dengan pesan: `Selesaikan serah terima dan pengembalian barang sebelum menandai selesai.`
- `settle` menghasilkan receipt PDF tipe `cod_receipt`; `amount_paid` boleh berbeda dari `amount_due` (untuk catatan).
- `Cancel` setelah handover ditolak; harus lewat pengembalian.

**`AgreementFile` lama:**
- Ganti isi handler: cari dokumen `agreement` terbaru untuk rental; jika belum ada, generate on-demand; stream.
- Hapus ketergantungan `AgreementPDFPath` dari `main.go` (atau biarkan sebagai fallback dev).
- Hapus `backend/assets/rental-agreement.pdf` setelah tidak dipakai.

### 9.4 Frontend

**`frontend/src/pages/RentalDetail.jsx`**
- Tambah panel `COD & Dokumen` setelah panel status:
  - Timeline: `Disetujui` → `Barang diserahkan` → `Dikonfirmasi penyewa` → `Barang dikembalikan` → `Dikonfirmasi penyedia` → `Pembayaran dicatat`.
  - Tombol aksi sesuai peran dan state.
  - Upload bukti serah terima/pengembalian (foto, maks 10 MB).
  - Unduh `Surat Perjanjian`, `Kuitansi COD`, `Bukti Serah Terima`, `Bukti Pengembalian`.
- Copy Bahasa Indonesia, gunakan komponen `Alert`, `Badge`, `Button` yang sudah ada.
- Jangan tampilkan tombol `Selesai` jika COD belum lengkap; tampilkan hint.

**Admin:**
- Halaman `AdminRentals.jsx` (baru) atau tab di `Admin.jsx`: daftar rental, filter status, detail, dokumen, chat metadata.
- Tombol unduh dokumen membuka modal alasan.

### 9.5 Acceptance Fase 5

1. Ajukan rental → owner approve → dokumen `agreement` muncul; PDF bisa diunduh kedua pihak.
2. PDF berisi nama, NIM, barang, tanggal, harga, lokasi.
3. Owner upload bukti serah terima → renter confirm → state berubah.
4. Renter upload bukti pengembalian → owner confirm.
5. Owner catat pembayaran → kuitansi PDF muncul.
6. Coba `Complete` sebelum handover/return → 409 `cod_incomplete`.
7. Setelah lengkap → `Complete` berhasil; riwayat event bertambah.
8. Admin unduh dokumen tanpa alasan → 422; dengan alasan → file + audit log.
9. File tidak bisa diakses tanpa token peserta/admin.

---

## 10. Fase 6 — Riwayat rental & monitoring admin

### 10.1 API

**Peserta:**
```text
GET /api/rentals/mine?role=renter|owner&status=&from=&to=&q=&limit=&offset=
GET /api/rentals/history?role=renter|owner&year=2026
```

- `role` default: semua rental yang melibatkan user.
- `from`/`to` filter `created_at`.
- `q` mencari judul listing.
- `history` mengembalikan ringkasan per bulan: jumlah, total nilai, selesai, batal.

**Admin:**
```text
GET /api/admin/rentals?status=&from=&to=&q=&overdue=true&limit=&offset=
GET /api/admin/rentals/{id}
GET /api/admin/rentals/stats
GET /api/admin/rentals/export.csv
GET /api/admin/users?q=&role=&status=&verified=
GET /api/admin/users/{id}
GET /api/admin/users/export.csv
GET /api/admin/stats
```

- `overdue=true`: `status='approved' AND end_date < current_date`.
- `stats`: total user, user verified, listing aktif, rental per status, overdue, laporan open, dokumen count, storage bytes (opsional).
- CSV memakai `encoding/csv`, header Bahasa Indonesia, `Content-Disposition: attachment`.
- Semua export admin diaudit (`moderation_actions` action `export_data`, target_type `rental`/`user`).

### 10.2 Store

Tambah query di `backend/internal/rental/store_postgres.go`:
- `AdminList(ctx, filter) (RentalPage, error)` dengan `JOIN listings`, `JOIN users` untuk nama.
- `AdminStats(ctx) (Stats, error)`.
- `History(ctx, userID, role, year)`.
- `ExportRows(ctx, filter)`.

Tambah query di `backend/internal/user/store_postgres.go`:
- `AdminList(ctx, filter) ([]AdminUser, int, error)` dengan `is_ktm_verified`.
- `AdminStats`.

### 10.3 Frontend

- `frontend/src/pages/Rentals.jsx`: tambah filter status, peran (penyewa/penyedia), tanggal, pencarian; tombol ekspor CSV.
- `frontend/src/pages/Dashboard.jsx`: kartu ringkasan rental (aktif, selesai, terlambat).
- `frontend/src/pages/Admin.jsx`: kartu statistik + tabel rental terbaru + tautan ke `AdminRentals`.
- `frontend/src/pages/AdminRentals.jsx` (baru): tabel dengan filter, pagination, detail, ekspor.
- `frontend/src/pages/AdminUsers.jsx` (baru): daftar user, status verifikasi, suspend/reactivate, hapus akun belum verified tanpa aktivitas.
- `frontend/src/App.jsx`: route `/admin/rentals`, `/admin/users`.

### 10.4 Acceptance Fase 6

1. Filter rental per status/peran/tanggal bekerja.
2. `overdue=true` hanya menampilkan rental approved yang `end_date` sudah lewat.
3. Admin melihat semua rental, bukan hanya miliknya.
4. CSV bisa dibuka di Excel, kolom Bahasa Indonesia, tidak ada data sensitif (password, chat).
5. Ekspor tercatat di audit.
6. Statistik admin sesuai isi DB.

---

## 11. Fase 7 — Laporan, suspend, audit, ekspor

### 11.1 Migrasi `0013_admin_governance`

Buat `postgres/migrations/0013_admin_governance.up.sql`:

```sql
-- 0013_admin_governance.up.sql

ALTER TABLE reports DROP CONSTRAINT IF EXISTS reports_type_check;
ALTER TABLE reports ADD CONSTRAINT reports_type_check CHECK (
    type IN ('technical', 'safety', 'fraud', 'harassment', 'other')
);

ALTER TABLE moderation_actions DROP CONSTRAINT IF EXISTS moderation_actions_action_check;
ALTER TABLE moderation_actions ADD CONSTRAINT moderation_actions_action_check CHECK (
    action IN (
        'takedown_listing', 'restore_listing',
        'suspend_user', 'reactivate_user',
        'resolve_report', 'reject_report', 'approve_template',
        'delete_unverified_user', 'download_document', 'export_data', 'revoke_ktm'
    )
);

ALTER TABLE moderation_actions DROP CONSTRAINT IF EXISTS moderation_actions_target_type_check;
ALTER TABLE moderation_actions ADD CONSTRAINT moderation_actions_target_type_check CHECK (
    target_type IN (
        'listing', 'user', 'report', 'letter_template',
        'rental', 'rental_document', 'ktm'
    )
);

CREATE INDEX IF NOT EXISTS rentals_admin_status_created_idx
    ON rentals (status, created_at DESC);
CREATE INDEX IF NOT EXISTS users_created_idx
    ON users (created_at DESC);
CREATE INDEX IF NOT EXISTS reports_created_idx
    ON reports (created_at DESC);
```

Down: kembalikan constraint lama (pastikan tidak ada data dengan nilai baru; jika ada, hapus dulu).

### 11.2 API laporan & moderasi

**Peserta:**
```text
POST /api/reports
     { type, subject, body, rental_id?, listing_id? }
GET  /api/reports/mine
```

**Admin:**
```text
GET  /api/admin/reports?status=&type=&limit=&offset=
GET  /api/admin/reports/{id}
POST /api/admin/reports/{id}/resolve   { resolution_note }
POST /api/admin/reports/{id}/reject    { resolution_note }
POST /api/admin/users/{id}/suspend     { reason }
POST /api/admin/users/{id}/reactivate  { reason }
DELETE /api/admin/users/{id}           { reason }   // hanya belum verified + tanpa aktivitas
GET  /api/admin/audit?actor_id=&action=&from=&to=&limit=&offset=
```

Aturan:
- `suspend`: update `status='suspended'`, `token_version=token_version+1`; tidak boleh suspend diri sendiri; catat `moderation_actions` action `suspend_user`.
- `reactivate`: `status='active'`, `token_version+1`; audit.
- `DELETE user`: cek `is_ktm_verified=false`; cek tidak ada baris di `listings`, `rentals` (renter/owner), `rental_messages`, `reports`; jika ada → 409 `user_has_activity`. Hapus dalam transaksi; audit `delete_unverified_user`. Ini membebaskan NIM + email.
- `resolve/reject report`: update status + `resolution_note`, `resolved_at`; audit; kirim notifikasi ke pelapor.
- `GET /api/admin/audit`: baca `moderation_actions` + `chat_access_log` (union atau dua tab). Jangan izinkan edit/hapus.

### 11.3 Frontend

- `frontend/src/pages/Report.jsx` atau modal dari `ListingDetail`/`RentalDetail`: form laporan.
- `frontend/src/pages/AdminReports.jsx` (baru): antrean laporan, filter, detail, resolve/reject.
- `frontend/src/pages/AdminUsers.jsx`: tombol suspend/reactivate/hapus (hapus hanya muncul jika belum verified).
- `frontend/src/pages/AdminAudit.jsx` (baru): tabel audit, filter, read-only.
- `frontend/src/App.jsx`: route `/admin/reports`, `/admin/audit`.
- Semua copy Bahasa Indonesia; konfirmasi tindakan destruktif memakai modal.

### 11.4 Acceptance Fase 7

1. User melaporkan listing → muncul di antrean admin.
2. Admin resolve dengan catatan → pelapor dapat notifikasi; status berubah.
3. Admin suspend user → user langsung logout (token invalid) dan login berikutnya 403.
4. Admin reactivate → user bisa login lagi.
5. Admin hapus akun belum verified tanpa aktivitas → berhasil; NIM + email bebas.
6. Admin hapus akun dengan rental → 409.
7. Audit log menampilkan semua tindakan, tidak bisa dihapus.
8. Ekspor CSV user/rental tercatat.

---

## 12. Fase 8 — Hardening & operasional

### 12.1 IP di belakang proxy

**`backend/internal/httpapi/middleware.go`** tambah:

```go
func clientIP(r *http.Request, trustedProxy bool) string {
    if trustedProxy {
        if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
            parts := strings.Split(xff, ",")
            return strings.TrimSpace(parts[0])
        }
        if ip := r.Header.Get("X-Real-IP"); ip != "" {
            return strings.TrimSpace(ip)
        }
    }
    host, _, err := net.SplitHostPort(r.RemoteAddr)
    if err != nil {
        return r.RemoteAddr
    }
    return host
}
```

- Ganti pemakaian `r.RemoteAddr` di `RateLimit` dengan `clientIP(r, trustedProxy)`.
- Tambah parameter `trustedProxy` ke `RateLimit` atau bungkus dengan middleware yang menyimpan IP di context.
- Jangan percaya `X-Forwarded-For` jika `TRUSTED_PROXY=false`.

### 12.2 Security headers

Tambah middleware `SecurityHeaders`:

```go
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("X-Frame-Options", "DENY")
w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
```

- HSTS diatur di Vercel/Cloudflare/Render, bukan di Go.
- CSP frontend di `index.html` atau header Vercel:
  `default-src 'self'; img-src 'self' data: blob:; connect-src 'self' https://api.domainmu.com wss://api.domainmu.com; style-src 'self' 'unsafe-inline'; script-src 'self';`

### 12.3 Health check

Ubah `Health` menjadi handler yang mengecek DB:

```go
GET /api/health        → 200 {"status":"ok"} jika DB ping < 2s, else 503
GET /api/health/live   → 200 selalu (untuk Render)
GET /api/health/ready  → 200 jika DB siap
```

- Render health check path: `/api/health/ready`.
- UptimeRobot: `/api/health`.

### 12.4 Rate limit produksi

| Endpoint | Limit |
|---|---|
| `POST /api/auth/register` | 5 / 10 menit / IP |
| `POST /api/auth/login` | 10 / menit / IP |
| `POST /api/auth/otp/request` | 3 / 10 menit / email + 10 / jam / IP |
| `POST /api/auth/otp/verify` | 10 / 10 menit / IP |
| `POST /api/ktm` | 5 / jam / user |
| `POST /api/listings/{id}/rentals` | 10 / jam / user |
| `POST /api/rentals/{id}/messages` | 60 / menit / user |
| `POST /api/admin/.../reveal` | 10 / jam / admin |

Catatan: rate limit in-memory hanya akurat untuk **satu instance**. Render Free menjalankan satu instance, jadi aman. Jika nanti scale >1, pindah ke Redis/Upstash.

### 12.5 Connection pooler Supabase

- `DATABASE_URL` untuk app: pooler **transaction mode**, port `6543`:
  ```text
  postgres://postgres.<ref>:<password>@aws-0-<region>.pooler.supabase.com:6543/postgres?sslmode=require&default_query_exec_mode=simple_protocol&statement_cache_capacity=0
  ```
- `DATABASE_URL_DIRECT` untuk migrasi/backup: port `5432`:
  ```text
  postgres://postgres.<ref>:<password>@db.<ref>.supabase.co:5432/postgres?sslmode=require
  ```
- `db.New` sudah `MaxConns=10`; turunkan ke `5` jika Supabase free pooler membatasi.
- Jangan jalankan migrasi lewat port 6543 (prepared statement/DDL bisa gagal).

### 12.6 Backup, keep-alive, monitoring

Semua langkah operasional — backup harian `pg_dump`, keep-alive Supabase, `pg_cron`, UptimeRobot, dan pemantauan kuota — ada di **`docs/TUTORIAL_DEPLOY.md` bagian 7**. Jangan menduplikasi konfigurasi di runbook ini.

### 12.7 Acceptance Fase 8 (kode)

1. `curl /api/health` → 200 saat DB hidup; 503 saat `DATABASE_URL` salah.
2. Rate limit OTP mengembalikan 429 setelah batas.
3. `X-Forwarded-For` dipakai untuk rate limit saat `TRUSTED_PROXY=true`.
4. Security headers muncul di response.
5. `go test ./...` lulus.
6. Tidak ada panic saat service role key / SMTP tidak dikonfigurasi di mode development.

---

## 13. Fase 9 — Deploy (dokumen terpisah)

Deploy dilakukan **manual oleh pemilik proyek**, bukan oleh sesi AI. Semua langkah Supabase, migrasi, seed, admin bootstrap, Render, Vercel/Cloudflare, domain, Brevo, backup, cron, monitoring, smoke test, dan rollback ada di:

**`docs/TUTORIAL_DEPLOY.md`**

Runbook ini hanya mengatur perubahan kode. Jika sesi AI diminta menyiapkan kode untuk deploy, cukup pastikan env var yang dipakai kode tercatat di tutorial; jangan menjalankan deploy.

---

## 14. Fase 10 — Acceptance test end-to-end

Jalankan berurutan di produksi (atau staging yang identik):

### A. Auth & identitas
- [ ] Daftar dengan `@gmail.com` ditolak.
- [ ] Daftar `@mhs.unesa.ac.id` menerima OTP.
- [ ] Kode salah 5× terkunci.
- [ ] Daftar sukses → JWT → bisa buka profil.
- [ ] NIM ganda → 409 sesuai aturan.
- [ ] Lupa sandi → OTP → password baru → login.
- [ ] Login admin → OTP → masuk dashboard.
- [ ] Admin suspend user → user logout paksa.
- [ ] Admin hapus akun belum verified tanpa aktivitas → NIM/email bebas.

### B. KTM
- [ ] Upload KTM JPG/PNG ≤5 MB.
- [ ] File >5 MB / PDF ditolak.
- [ ] Admin lihat antrean, buka file, verify dengan konfirmasi NIM.
- [ ] User bisa buat listing setelah verified.
- [ ] Admin revoke → user tidak bisa buat listing/rental baru.

### C. Listing & rental
- [ ] Buat listing + foto.
- [ ] Listing muncul di katalog publik.
- [ ] Ajukan rental → owner approve.
- [ ] Tanggal bentrok ditolak.
- [ ] Chat real-time + fallback polling.
- [ ] Riwayat status tampil.
- [ ] Filter riwayat & ekspor CSV.

### D. COD & dokumen
- [ ] Agreement PDF per rental bisa diunduh.
- [ ] Upload bukti serah terima + konfirmasi.
- [ ] Upload bukti pengembalian + konfirmasi.
- [ ] Catat pembayaran → kuitansi PDF.
- [ ] `Complete` tanpa COD lengkap → 409.
- [ ] `Complete` setelah lengkap → sukses.
- [ ] Admin unduh dokumen dengan alasan → audit.

### E. Chat encryption
- [ ] DB `rental_messages.body` NULL, ciphertext terisi.
- [ ] Peserta baca pesan normal.
- [ ] Admin metadata tanpa isi.
- [ ] Admin reveal dengan alasan + password → isi tampil + audit.
- [ ] Export transkrip → file privat + audit.

### F. Admin & laporan
- [ ] Statistik admin sesuai.
- [ ] Laporan user masuk antrean.
- [ ] Resolve/reject + notifikasi.
- [ ] Audit log read-only.
- [ ] Ekspor user/rental.

### G. Operasional
- [ ] `https://api.../api/health` 200.
- [ ] Backup artifact muncul.
- [ ] Keep-alive jalan.
- [ ] Tidak ada error 500 di log selama smoke test.
- [ ] Tidak ada service role key / password di bundle frontend (`grep -R "service_role" frontend/dist`).

---

## 15. Environment variable (ringkas)

Daftar lengkap nama, contoh nilai, dan cara mengisinya ada di **`docs/TUTORIAL_DEPLOY.md` bagian “Referensi environment variable”**.

Aturan:
- Jangan menaruh nilai secret di runbook, README, atau repo.
- Frontend hanya memakai `VITE_API_URL`; tidak ada secret di `VITE_*`.
- Backend memakai `DATABASE_URL` (pooler), `JWT_SECRET`, `CORS_ORIGINS`, `PUBLIC_BASE_URL`, dan env fitur sesuai fase yang sudah dikerjakan.

---

## 16. Referensi migrasi & endpoint

### Migrasi

**Skenario A — urutan penuh Fase 0–10:**

| File | Isi |
|---|---|
| `0010_auth_production` | `email_verified_at`, `last_login_at`, `auth_otps`, NIM lock trigger |
| `0011_chat_encryption` | ciphertext/nonce/key_version, `chat_access_log` |
| `0012_cod_documents` | `rental_documents`, `rental_cod` |
| `0013_admin_governance` | perluasan tipe report/action, index admin |

**Skenario B — mulai dari Fase 5:**

| File | Isi |
|---|---|
| `0010_cod_documents` | `rental_documents`, `rental_cod` |
| `0011_rental_history` | index/filter riwayat rental (jika ada perubahan skema) |
| `0012_admin_governance` | perluasan tipe report/action, index admin |
| `0013_auth_production` | `email_verified_at`, `last_login_at`, `auth_otps`, NIM lock trigger |
| `0014_chat_encryption` | ciphertext/nonce/key_version, `chat_access_log` |

Isi SQL sama; hanya nama file yang berbeda. Jangan mencampur kedua skenario. Jangan pernah mengubah migrasi yang sudah pernah dijalankan di produksi; buat migrasi baru.

### Endpoint baru (ringkas)

```text
POST   /api/auth/otp/request
POST   /api/auth/otp/verify
POST   /api/auth/login/otp
POST   /api/auth/password/forgot
POST   /api/auth/password/reset

GET    /api/rentals/history
GET    /api/rentals/{id}/documents
GET    /api/rentals/{id}/documents/{docID}/file
GET    /api/rentals/{id}/cod
POST   /api/rentals/{id}/cod/handover
POST   /api/rentals/{id}/cod/handover/confirm
POST   /api/rentals/{id}/cod/return
POST   /api/rentals/{id}/cod/return/confirm
POST   /api/rentals/{id}/cod/settle

POST   /api/reports
GET    /api/reports/mine

GET    /api/admin/rentals
GET    /api/admin/rentals/{id}
GET    /api/admin/rentals/stats
GET    /api/admin/rentals/export.csv
GET    /api/admin/rentals/{id}/documents
POST   /api/admin/rentals/{id}/documents/{docID}/download
GET    /api/admin/rentals/{id}/messages
POST   /api/admin/rentals/{id}/messages/{msgID}/reveal
POST   /api/admin/rentals/{id}/transcript/export

GET    /api/admin/users
GET    /api/admin/users/{id}
GET    /api/admin/users/export.csv
POST   /api/admin/users/{id}/suspend
POST   /api/admin/users/{id}/reactivate
DELETE /api/admin/users/{id}

GET    /api/admin/reports
GET    /api/admin/reports/{id}
POST   /api/admin/reports/{id}/resolve
POST   /api/admin/reports/{id}/reject
GET    /api/admin/audit
GET    /api/admin/stats
```

---

## 17. Rollback

- **Kode:** karena tidak ada commit otomatis, rollback dilakukan dengan mengembalikan file yang diubah dari salinan/backup kerja, lalu build ulang. Jangan memakai `git reset --hard` atau `git clean -fd` tanpa izin.
- **Migrasi:** down migration disediakan untuk migrasi baru (0010–0014, sesuai skenario penomoran yang dipilih). Jalankan hanya jika yakin tidak ada data baru:
  ```bash
  migrate -path postgres/migrations -database "$DATABASE_URL_DIRECT" down 1
  ```
- **Database produksi:** restore dari artifact backup:
  ```bash
  pg_restore --clean --no-owner --no-privileges -d "$DATABASE_URL_DIRECT" unesarent.dump
  ```
- **Storage:** object tidak ikut terhapus saat rollback DB. Jika rollback besar, catat daftar object yatim lalu bersihkan manual.
- **Chat encryption:** jangan hapus key lama dari `CHAT_ENC_KEYS` selama masih ada pesan dengan versi itu. Rollback aplikasi ke versi lama tanpa dukungan enkripsi akan membuat chat tidak terbaca; jangan rollback melewati Fase 4 tanpa backup.

---

## 18. Batasan yang harus ditulis di laporan UAS

Tulis jujur di bab "Keterbatasan":

1. Free tier Supabase/Render dapat pause/spin down; bukan SLA produksi resmi kampus.
2. Supabase Free tidak punya backup otomatis/PITR; backup harian adalah mitigasi, bukan jaminan.
3. Rate limit masih in-memory; cocok untuk satu instance, belum untuk multi-instance.
4. OTP email bergantung pada Brevo; jika email masuk spam, user tidak bisa daftar.
5. Verifikasi KTM adalah verifikasi manual admin, bukan verifikasi otomatis ke SIAKADU.
6. Perjanjian COD adalah dokumen komunitas, bukan akta notaris.
7. Admin dapat membuka isi chat untuk bukti; ini kebijakan privasi yang harus disetujui user saat mendaftar.
8. Tidak ada SSO kampus; identitas mahasiswa dibuktikan lewat email + KTM.
9. Data produksi dimulai kosong; tidak ada migrasi data dev.
10. Skala free tier cocok untuk UAS/soft launch, bukan operasi resmi fakultas.

---

## 19. Prompt copy-paste untuk sesi AI berikutnya

```text
Baca docs/RUNBOOK_PRODUKSI.md sampai selesai. Kerjakan fase sesuai prioritas tanpa bertanya; semua keputusan sudah final di dokumen itu.

Aturan:
- Jangan lompat fase. Di akhir setiap fase jalankan acceptance test fase tersebut.
- Cukup edit file. Jangan commit, push, atau membuat branch otomatis kecuali diminta.
- Jangan mengubah desain UI di luar yang disebutkan. Semua copy tetap Bahasa Indonesia.
- Produksi mulai dari database kosong + reference seed; jangan migrasikan data dev.
- Jangan pernah menaruh .env, service role key, password, atau file KTM di dalam repo.
- Jika ada error, perbaiki sampai acceptance test lulus sebelum lanjut.
- Deploy tidak dikerjakan dari sesi AI. Deploy manual ada di docs/TUTORIAL_DEPLOY.md.

Prioritas UAS: Fase 5 (COD Transaction and Documentation), Fase 6 (Rental Status and History), Fase 7 (Admin and Reporting). Prasyarat produksi: Fase 2 (auth/OTP), Fase 3 (storage), Fase 4 (chat encryption). Kerjakan prasyarat yang diperlukan lebih dulu, lalu 5–7.

Jika dimulai dari Fase 5 (Fase 2–4 belum dikerjakan):
- Baca bagian 1.2 runbook.
- Gunakan storage lokal untuk dev; jangan deploy ke Render dengan storage lokal.
- Jangan mengubah auth/OTP/chat encryption di fase ini.
- Penomoran migrasi memakai Skenario B: 0010_cod_documents, 0011_rental_history, 0012_admin_governance, lalu 0013_auth_production, 0014_chat_encryption.
- Jika sesi sedang limit, cukup edit file; jangan jalankan build/test berat. Laporkan perintah verifikasi untuk pemilik.

Setelah selesai, laporkan:
1. Fase yang selesai dan file yang diubah.
2. Hasil acceptance test per fase (atau catatan jika build ditunda karena limit).
3. Env var yang sudah dipasang (tanpa nilai secret).
4. Sisa risiko/pekerjaan.
```
