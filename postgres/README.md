# Database UNESARENT

Schema dan migration PostgreSQL 17. Berisi identitas, katalog, rental, verifikasi KTM, chat terenkripsi, dokumen COD, dan governance admin. Versi migrasi terakhir: `13`.

## Jalankan lokal

```bash
docker compose up -d
```

Migration:

```bash
export DATABASE_URL_DIRECT='postgres://...@localhost:5432/unesarent?sslmode=disable'
migrate -path migrations -database "$DATABASE_URL_DIRECT" up
migrate -path migrations -database "$DATABASE_URL_DIRECT" version
```

Seed lokal:

```bash
psql "$DATABASE_URL_DIRECT" -v ON_ERROR_STOP=1 -f seed/dev_seed.sql
```

## Produksi

Seed yang dipakai hanya:

```bash
psql "$DATABASE_URL_DIRECT" -v ON_ERROR_STOP=1 -f seed/reference.sql
```

Jangan jalankan `dev_seed.sql` di produksi.

Aplikasi memakai transaction pooler port `6543`, bukan direct `5432`. Tambahkan pada `DATABASE_URL`:

```text
?sslmode=require&default_query_exec_mode=simple_protocol&statement_cache_capacity=0
```

Tanpa itu, pgx bisa menemui error `prepared statement already exists` (SQLSTATE `42P05`).

## Isi folder

```text
initdb/       extension, dijalankan saat volume kosong
migrations/   up/down 0001 sampai 0013
seed/         reference.sql (produksi) dan dev_seed.sql (lokal)
```

Aturan penting seperti format NIM, status KTM, kepemilikan rental, dan audit sudah menjadi constraint/trigger di database, bukan hanya di aplikasi.
