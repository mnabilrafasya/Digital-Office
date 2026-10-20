-- 001_units.sql
-- DATA DUMMY DEVELOPMENT: unit kerja (Direktorat -> Kompartemen/Departemen -> Bagian).
-- Hanya dijalankan oleh "go run ./cmd/seed dev" saat APP_ENV=development.
SET NOCOUNT ON;
SET XACT_ABORT ON;

-- Level 1: Direktorat (tanpa induk)
INSERT INTO units (company_code, parent_id, unit_type, code, name) VALUES
    ('PSP', NULL, 'DIREKTORAT', 'DU',  N'Direktorat Utama'),
    ('PSP', NULL, 'DIREKTORAT', 'DKU', N'Direktorat Keuangan dan Umum'),
    ('PSP', NULL, 'DIREKTORAT', 'DOP', N'Direktorat Operasi dan Produksi'),
    ('PKG', NULL, 'DIREKTORAT', 'DKU', N'Direktorat Keuangan dan Umum');

-- Level 2: Kompartemen / Departemen
INSERT INTO units (company_code, parent_id, unit_type, code, name)
SELECT v.company_code, p.id, v.unit_type, v.code, v.name
FROM (VALUES
    ('PSP', 'DU',  'KOMPARTEMEN', 'SP',  N'Kompartemen Sekretariat Perusahaan'),
    ('PSP', 'DKU', 'KOMPARTEMEN', 'TI',  N'Kompartemen Teknologi Informasi'),
    ('PSP', 'DKU', 'DEPARTEMEN',  'SDM', N'Departemen Sumber Daya Manusia'),
    ('PSP', 'DKU', 'DEPARTEMEN',  'AK',  N'Departemen Akuntansi dan Keuangan'),
    ('PSP', 'DOP', 'KOMPARTEMEN', 'PRD', N'Kompartemen Produksi'),
    ('PKG', 'DKU', 'DEPARTEMEN',  'TI',  N'Departemen Teknologi Informasi')
) AS v (company_code, parent_code, unit_type, code, name)
JOIN units p ON p.company_code = v.company_code AND p.code = v.parent_code;

-- Level 3: Bagian
INSERT INTO units (company_code, parent_id, unit_type, code, name)
SELECT v.company_code, p.id, v.unit_type, v.code, v.name
FROM (VALUES
    ('PSP', 'TI',  'BAGIAN', 'PA',  N'Bagian Pengembangan Aplikasi'),
    ('PSP', 'TI',  'BAGIAN', 'IKT', N'Bagian Infrastruktur dan Keamanan TI'),
    ('PSP', 'PRD', 'BAGIAN', 'OP',  N'Bagian Operasi Pabrik')
) AS v (company_code, parent_code, unit_type, code, name)
JOIN units p ON p.company_code = v.company_code AND p.code = v.parent_code;

IF (SELECT COUNT(*) FROM units) <> 13 THROW 50101, N'Seed dev gagal: jumlah units harus 13', 1;
