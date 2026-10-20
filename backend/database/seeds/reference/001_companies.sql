-- 001_companies.sql
-- Perusahaan dalam grup. Idempoten (MERGE), aman dijalankan berulang di semua environment.
-- letter_number_format hanya diisi saat perusahaan pertama kali dibuat, supaya format yang
-- sudah diubah administrator tidak tertimpa saat seed dijalankan ulang.
SET NOCOUNT ON;
SET XACT_ABORT ON;

MERGE companies WITH (HOLDLOCK) AS t
USING (VALUES
    ('PSP', N'PT Pupuk Sriwidjaja Palembang', N'Palembang', '{SEQ}/{UNIT}/{COMP}/{MM_ROMAN}/{YYYY}'),
    ('PKG', N'PT Petrokimia Gresik',          N'Gresik',    '{SEQ}/{UNIT}/{COMP}/{MM_ROMAN}/{YYYY}')
) AS s (code, name, city, letter_number_format)
    ON t.code = s.code
WHEN MATCHED AND (t.name <> s.name OR ISNULL(t.city, N'') <> ISNULL(s.city, N'')) THEN
    UPDATE SET name = s.name, city = s.city, updated_at = SYSDATETIMEOFFSET()
WHEN NOT MATCHED BY TARGET THEN
    INSERT (code, name, city, letter_number_format)
    VALUES (s.code, s.name, s.city, s.letter_number_format);
