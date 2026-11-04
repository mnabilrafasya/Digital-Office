-- 000017_add_letter_template_description.up.sql
-- Template bisa dibuat dan diganti semua pengguna: deskripsi untuk pilihan di form, dan nama
-- template aktif unik per perusahaan supaya pilihan tidak membingungkan.
SET XACT_ABORT ON;
BEGIN TRANSACTION;

ALTER TABLE letter_templates ADD description NVARCHAR(500) NULL;

CREATE UNIQUE INDEX uq_letter_templates_company_name
    ON letter_templates (company_code, name) WHERE deleted_at IS NULL;

COMMIT TRANSACTION;
