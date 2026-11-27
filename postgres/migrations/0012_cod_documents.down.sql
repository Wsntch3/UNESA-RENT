-- 0012_cod_documents.down.sql
DROP TRIGGER IF EXISTS rental_cod_set_updated_at ON rental_cod;
DROP TABLE IF EXISTS rental_cod;
DROP TABLE IF EXISTS rental_documents;
