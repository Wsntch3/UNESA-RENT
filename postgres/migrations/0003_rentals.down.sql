-- 0003_rentals.down.sql

DROP TABLE IF EXISTS security_letters;
DROP FUNCTION IF EXISTS security_letters_require_approved();
DROP TABLE IF EXISTS rental_events;

DROP TRIGGER IF EXISTS rentals_before_write ON rentals;
DROP FUNCTION IF EXISTS rentals_before_write();

DROP TABLE IF EXISTS rentals;
DROP TABLE IF EXISTS letter_templates;
