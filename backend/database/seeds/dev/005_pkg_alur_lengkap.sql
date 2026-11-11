-- 005_pkg_alur_lengkap.sql
-- DATA DUMMY DEVELOPMENT: melengkapi PKG (Petrokimia Gresik, fiktif) supaya seluruh alur bisa diuji
-- seperti di PSP: admin perusahaan, 2 VP (alias "Seluruh VP"), AVP sebagai approver, unit Kepada dan
-- Tembusan, unit tanpa pimpinan, user nonaktif, dan watermark sendiri. Semua nama, NIK, dan email FIKTIF.
--
-- Idempoten: hanya menambah yang belum ada, sehingga juga dijalankan "go run ./cmd/seed dev" pada
-- database yang sudah berisi data (tanpa reset). @password_hash dikirim oleh cmd/seed.
--
-- Struktur PKG setelah seed ini:
--   Direktorat Keuangan dan Umum (DKU)        Direktur Keuangan dan Umum*    Suryo (DIREKTUR)
--   +- Departemen Teknologi Informasi (TI)    VP Teknologi Informasi*        Bayu (VP)
--   |                                         Staf Teknologi Informasi       Lestari, Hendro (nonaktif)
--   |                                         Administrator Sistem           Rizky (ADMIN)
--   +- Departemen Sumber Daya Manusia (SDM)   VP Sumber Daya Manusia*        Wulan (VP)
--   |                                         Staf Sumber Daya Manusia       Agung
--   +- Departemen Keuangan (KEU)              AVP Keuangan*                  Dian (AVP)
--   |                                         Staf Keuangan                  Fikri
--   +- Bagian Arsip dan Dokumentasi (ARSIP)   Staf Arsip (tanpa pimpinan)    Nadia
--   (* = pimpinan unit)
SET NOCOUNT ON;
SET XACT_ABORT ON;

IF NOT EXISTS (SELECT 1 FROM units WHERE company_code = 'PKG' AND code = 'DKU' AND deleted_at IS NULL)
    THROW 50110, N'Unit dasar PKG (DKU) belum ada: jalankan seed dev dari database kosong dulu', 1;

INSERT INTO units (company_code, parent_id, unit_type, code, name)
SELECT 'PKG', p.id, v.unit_type, v.code, v.name
FROM (VALUES
    ('DEPARTEMEN', 'SDM',   N'Departemen Sumber Daya Manusia'),
    ('DEPARTEMEN', 'KEU',   N'Departemen Keuangan'),
    ('BAGIAN',     'ARSIP', N'Bagian Arsip dan Dokumentasi')
) AS v (unit_type, code, name)
JOIN units p ON p.company_code = 'PKG' AND p.code = 'DKU' AND p.deleted_at IS NULL
WHERE NOT EXISTS (SELECT 1 FROM units x WHERE x.company_code = 'PKG' AND x.code = v.code);

INSERT INTO positions (unit_id, name, is_unit_head, level_id)
SELECT u.id, v.name, v.is_unit_head, lv.id
FROM (VALUES
    ('TI',    N'Administrator Sistem',      0, 'STAF'),
    ('SDM',   N'VP Sumber Daya Manusia',    1, 'VP'),
    ('SDM',   N'Staf Sumber Daya Manusia',  0, 'STAF'),
    ('KEU',   N'AVP Keuangan',              1, 'AVP'),
    ('KEU',   N'Staf Keuangan',             0, 'STAF'),
    ('ARSIP', N'Staf Arsip',                0, 'STAF')
) AS v (unit_code, name, is_unit_head, level_code)
JOIN units u    ON u.company_code = 'PKG' AND u.code = v.unit_code AND u.deleted_at IS NULL
JOIN lookups lv ON lv.company_code = 'PKG' AND lv.category = 'POSITION_LEVEL' AND lv.code = v.level_code
               AND lv.deleted_at IS NULL
WHERE NOT EXISTS (SELECT 1 FROM positions x WHERE x.unit_id = u.id AND x.name = v.name AND x.deleted_at IS NULL);

INSERT INTO users (position_id, role_id, employee_number, name, email, password_hash, is_active)
SELECT p.id, r.id, v.employee_number, v.name, v.email, @password_hash, v.is_active
FROM (VALUES
    ('TI',    N'Administrator Sistem',     'ADMIN',   '2010204', N'Rizky Ramadhan',          'rizky.ramadhan@petrokimia.test', 1),
    ('TI',    N'Staf Teknologi Informasi', 'PEGAWAI', '2016210', N'Hendro Wibisono',         'hendro.wibisono@petrokimia.test', 0),
    ('SDM',   N'VP Sumber Daya Manusia',   'PEGAWAI', '2005205', N'Ir. Wulan Sari, M.M.',    'wulan.sari@petrokimia.test',     1),
    ('SDM',   N'Staf Sumber Daya Manusia', 'PEGAWAI', '2018206', N'Agung Prasetyo',          'agung.prasetyo@petrokimia.test', 1),
    ('KEU',   N'AVP Keuangan',             'PEGAWAI', '2007207', N'Dian Puspita, S.E.',      'dian.puspita@petrokimia.test',   1),
    ('KEU',   N'Staf Keuangan',            'PEGAWAI', '2020208', N'Fikri Hidayat',           'fikri.hidayat@petrokimia.test',  1),
    ('ARSIP', N'Staf Arsip',               'PEGAWAI', '2021209', N'Nadia Putri',             'nadia.putri@petrokimia.test',    1)
) AS v (unit_code, position_name, role_code, employee_number, name, email, is_active)
JOIN units u     ON u.company_code = 'PKG' AND u.code = v.unit_code AND u.deleted_at IS NULL
JOIN positions p ON p.unit_id = u.id AND p.name = v.position_name AND p.deleted_at IS NULL
JOIN roles r     ON r.code = v.role_code AND r.deleted_at IS NULL
WHERE NOT EXISTS (SELECT 1 FROM users x WHERE x.employee_number = v.employee_number AND x.deleted_at IS NULL);

-- Watermark PKG berbeda dari PSP (hanya bila belum diatur)
UPDATE companies SET watermark_text = N'PETROKIMIA GRESIK', updated_at = SYSDATETIMEOFFSET()
WHERE code = 'PKG' AND watermark_text IS NULL;

IF (SELECT COUNT(*) FROM users
    WHERE deleted_at IS NULL AND employee_number IN
          ('1990201', '2008202', '2019203', '2010204', '2016210', '2005205', '2018206', '2007207', '2020208', '2021209')) <> 10
    THROW 50111, N'Seed dev gagal: 10 user PKG harus ada (cek ejaan unit/jabatan/role)', 1;
