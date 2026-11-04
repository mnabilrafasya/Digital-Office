-- 001_units.sql
-- DATA DUMMY DEVELOPMENT: unit kerja.
-- Unit PSP berasal dari HC (go run ./cmd/sync-hc) dan wajib sudah ada; di sini hanya unit PKG
-- (Petrokimia Gresik, fiktif) untuk uji pemisahan data antar perusahaan.
SET NOCOUNT ON;
SET XACT_ABORT ON;

IF NOT EXISTS (SELECT 1 FROM units WHERE company_code = 'PSP' AND is_active = 1)
    THROW 50100, N'Unit PSP belum ada: jalankan "go run ./cmd/sync-hc" dulu, lalu ulangi seed dev', 1;

INSERT INTO units (company_code, parent_id, unit_type, code, name) VALUES
    ('PKG', NULL, 'DIREKTORAT', 'DKU', N'Direktorat Keuangan dan Umum');

INSERT INTO units (company_code, parent_id, unit_type, code, name)
SELECT 'PKG', p.id, 'DEPARTEMEN', 'TI', N'Departemen Teknologi Informasi'
FROM units p
WHERE p.company_code = 'PKG' AND p.code = 'DKU';

IF (SELECT COUNT(*) FROM units WHERE company_code = 'PKG') <> 2 THROW 50101, N'Seed dev gagal: jumlah unit PKG harus 2', 1;
