-- 0001_identity.down.sql

DROP TABLE IF EXISTS user_contacts;

DROP TRIGGER IF EXISTS users_resolve_study_program ON users;
DROP FUNCTION IF EXISTS users_resolve_study_program();

DROP TRIGGER IF EXISTS users_set_updated_at ON users;
DROP TABLE IF EXISTS users;

DROP TABLE IF EXISTS study_programs;
DROP FUNCTION IF EXISTS set_updated_at();
