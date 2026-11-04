-- 000017_add_letter_template_description.down.sql
SET XACT_ABORT ON;
BEGIN TRANSACTION;

DROP INDEX IF EXISTS uq_letter_templates_company_name ON letter_templates;
ALTER TABLE letter_templates DROP COLUMN IF EXISTS description;

COMMIT TRANSACTION;
