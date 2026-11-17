-- 0008_letter_governance.down.sql

DROP TRIGGER IF EXISTS security_letters_immutable ON security_letters;
DROP FUNCTION IF EXISTS security_letters_immutable();

DROP TRIGGER IF EXISTS security_letters_require_active_template ON security_letters;
DROP FUNCTION IF EXISTS security_letters_require_active_template();

DROP TRIGGER IF EXISTS letter_templates_no_delete_activated ON letter_templates;
DROP FUNCTION IF EXISTS letter_templates_no_delete_activated();

DROP TRIGGER IF EXISTS letter_templates_guard ON letter_templates;
DROP FUNCTION IF EXISTS letter_templates_guard();

ALTER TABLE letter_templates DROP CONSTRAINT IF EXISTS letter_templates_active_requires_approval;

ALTER TABLE letter_templates
    DROP COLUMN IF EXISTS activated_at,
    DROP COLUMN IF EXISTS approved_offline_at,
    DROP COLUMN IF EXISTS approved_by_name,
    DROP COLUMN IF EXISTS approval_reference;

COMMENT ON TABLE reports IS 'Pengaduan pengguna; technical ditangani admin_dev, safety ditangani admin_bem.';
