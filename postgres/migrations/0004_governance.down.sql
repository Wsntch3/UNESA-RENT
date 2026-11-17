-- 0004_governance.down.sql

DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS moderation_actions;
DROP FUNCTION IF EXISTS moderation_actions_append_only();
DROP TABLE IF EXISTS reports;
