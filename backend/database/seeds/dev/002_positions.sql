-- 002_positions.sql
-- DATA DUMMY DEVELOPMENT: jabatan beserta level jabatan (POSITION_LEVEL perusahaan yang sama).
-- Pemetaan level: nama diawali Direktur -> DIREKTUR, SVP -> SVP, VP -> VP, AVP -> AVP, selain itu STAF.
-- is_unit_head = 1 -> pimpinan unit (penerima surat masuk dan anggota alias).
SET NOCOUNT ON;
SET XACT_ABORT ON;

INSERT INTO positions (unit_id, name, is_unit_head, level_id)
SELECT u.id, v.name, v.is_unit_head, lv.id
FROM (VALUES
    ('PSP', 'DU',  N'Direktur Utama',                    1, 'DIREKTUR'),
    ('PSP', 'DKU', N'Direktur Keuangan dan Umum',        1, 'DIREKTUR'),
    ('PSP', 'DOP', N'Direktur Operasi dan Produksi',     1, 'DIREKTUR'),
    ('PSP', 'SP',  N'SVP Sekretariat Perusahaan',        1, 'SVP'),
    ('PSP', 'SP',  N'Staf Tata Usaha',                   0, 'STAF'),
    ('PSP', 'TI',  N'SVP Teknologi Informasi',           1, 'SVP'),
    ('PSP', 'SDM', N'VP Sumber Daya Manusia',            1, 'VP'),
    ('PSP', 'SDM', N'Staf Pengembangan SDM',             0, 'STAF'),
    ('PSP', 'AK',  N'VP Akuntansi dan Keuangan',         1, 'VP'),
    ('PSP', 'AK',  N'Staf Anggaran',                     0, 'STAF'),
    ('PSP', 'PRD', N'SVP Produksi',                      1, 'SVP'),
    ('PSP', 'PA',  N'AVP Pengembangan Aplikasi',         1, 'AVP'),
    ('PSP', 'PA',  N'Staf Pengembangan Aplikasi',        0, 'STAF'),
    ('PSP', 'PA',  N'Analis Sistem Informasi',           0, 'STAF'),
    ('PSP', 'IKT', N'AVP Infrastruktur dan Keamanan TI', 1, 'AVP'),
    ('PSP', 'IKT', N'Staf Infrastruktur TI',             0, 'STAF'),
    ('PSP', 'IKT', N'Administrator Sistem',              0, 'STAF'),
    ('PSP', 'OP',  N'AVP Operasi Pabrik',                1, 'AVP'),
    ('PSP', 'OP',  N'Staf Operasi Pabrik',               0, 'STAF'),
    ('PKG', 'DKU', N'Direktur Keuangan dan Umum',        1, 'DIREKTUR'),
    ('PKG', 'TI',  N'VP Teknologi Informasi',            1, 'VP'),
    ('PKG', 'TI',  N'Staf Teknologi Informasi',          0, 'STAF')
) AS v (company_code, unit_code, name, is_unit_head, level_code)
JOIN units u    ON u.company_code = v.company_code AND u.code = v.unit_code
JOIN lookups lv ON lv.company_code = v.company_code AND lv.category = 'POSITION_LEVEL'
               AND lv.code = v.level_code AND lv.deleted_at IS NULL;

IF (SELECT COUNT(*) FROM positions) <> 22 THROW 50102, N'Seed dev gagal: jumlah positions harus 22 (sudah menjalankan seed reference?)', 1;
