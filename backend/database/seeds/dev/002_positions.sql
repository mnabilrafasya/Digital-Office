-- 002_positions.sql
-- DATA DUMMY DEVELOPMENT: jabatan beserta level jabatan (POSITION_LEVEL perusahaan yang sama).
-- Unit PSP dirujuk lewat nama unit aktif hasil sinkronisasi HC; unit PKG lewat kode.
-- Level mengikuti tingkat unit: Direktorat -> DIREKTUR, Kompartemen -> SVP, Departemen -> VP,
-- Bagian/Fungsional -> AVP, selain pimpinan -> STAF.
-- is_unit_head = 1 -> pimpinan unit (penerima surat masuk dan anggota alias).
SET NOCOUNT ON;
SET XACT_ABORT ON;

INSERT INTO positions (unit_id, name, is_unit_head, level_id)
SELECT u.id, v.name, v.is_unit_head, lv.id
FROM (VALUES
    -- Direksi
    ('PSP', N'Direktorat Utama',                               N'Direktur Utama',                  1, 'DIREKTUR'),
    ('PSP', N'Direktorat Keuangan & Umum',                     N'Direktur Keuangan & Umum',        1, 'DIREKTUR'),
    ('PSP', N'Direktorat Operasi & Produksi',                  N'Direktur Operasi & Produksi',     1, 'DIREKTUR'),
    -- Sekretaris Perusahaan
    ('PSP', N'Kompartemen Sekretaris Perusahaan',              N'SVP Sekretaris Perusahaan',       1, 'SVP'),
    ('PSP', N'Departemen Komunikasi & Administrasi Korporat',  N'Staf Tata Usaha',                 0, 'STAF'),
    -- Teknologi Informasi
    ('PSP', N'Kompartemen Teknologi & K3LH',                   N'SVP Teknologi & K3LH',            1, 'SVP'),
    ('PSP', N'Departemen Teknologi Informasi PSP',             N'VP Teknologi Informasi',          1, 'VP'),
    ('PSP', N'Departemen Teknologi Informasi PSP',             N'Staf Pengembangan Aplikasi',      0, 'STAF'),
    ('PSP', N'Departemen Teknologi Informasi PSP',             N'Staf Infrastruktur TI',           0, 'STAF'),
    ('PSP', N'Departemen Teknologi Informasi PSP',             N'Administrator Sistem',            0, 'STAF'),
    ('PSP', N'Fungsional Digitalisasi TI',                     N'AVP Digitalisasi TI',             1, 'AVP'),
    ('PSP', N'Fungsional Digitalisasi TI',                     N'Analis Sistem Informasi',         0, 'STAF'),
    -- SDM dan Keuangan
    ('PSP', N'Departemen Manajemen & Pengembangan SDM',        N'VP Manajemen & Pengembangan SDM', 1, 'VP'),
    ('PSP', N'Departemen Manajemen & Pengembangan SDM',        N'Staf Pengembangan SDM',           0, 'STAF'),
    ('PSP', N'Departemen Akuntansi',                           N'VP Akuntansi',                    1, 'VP'),
    ('PSP', N'Departemen Anggaran',                            N'Staf Anggaran',                   0, 'STAF'),
    -- Operasi pabrik
    ('PSP', N'Kompartemen Operasi',                            N'SVP Operasi',                     1, 'SVP'),
    ('PSP', N'Departemen Operasi Pabrik I',                    N'VP Operasi Pabrik I',             1, 'VP'),
    ('PSP', N'Departemen Operasi Pabrik I',                    N'Staf Operasi Pabrik',             0, 'STAF'),
    -- Petrokimia Gresik (fiktif)
    ('PKG', 'DKU',                                             N'Direktur Keuangan dan Umum',      1, 'DIREKTUR'),
    ('PKG', 'TI',                                              N'VP Teknologi Informasi',          1, 'VP'),
    ('PKG', 'TI',                                              N'Staf Teknologi Informasi',        0, 'STAF')
) AS v (company_code, unit_ref, name, is_unit_head, level_code)
JOIN units u    ON u.company_code = v.company_code AND u.is_active = 1 AND u.deleted_at IS NULL
               AND ((v.company_code = 'PSP' AND u.name = v.unit_ref) OR (v.company_code = 'PKG' AND u.code = v.unit_ref))
JOIN lookups lv ON lv.company_code = v.company_code AND lv.category = 'POSITION_LEVEL'
               AND lv.code = v.level_code AND lv.deleted_at IS NULL;

IF (SELECT COUNT(*) FROM positions) <> 22
    THROW 50102, N'Seed dev gagal: jumlah positions harus 22 (sudah menjalankan seed reference dan sync-hc?)', 1;
