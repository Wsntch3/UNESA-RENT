-- Extension yang dibutuhkan schema UNESARENT.
-- File ini hanya dijalankan PostgreSQL saat volume database masih kosong.
-- Migration 0001 juga membuat extension yang sama agar aman dijalankan manual.

CREATE EXTENSION IF NOT EXISTS btree_gist; -- exclusion constraint tanggal sewa
CREATE EXTENSION IF NOT EXISTS pg_trgm;    -- pencarian judul listing (NFR-01)
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_salt/crypt untuk seed admin dev
