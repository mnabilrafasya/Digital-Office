-- 003_users.sql
-- DATA DUMMY DEVELOPMENT: pengguna. Semua nama, NIK, dan email FIKTIF.
-- NIK sama dengan daftar "Akses Cepat" front-end (frontend/src/lib/dev-accounts.ts).
-- Aman diulang: hanya menambah NIK yang belum ada, tanpa mengubah akun/password lama.
-- @password_hash dikirim oleh cmd/seed (hasil bcrypt dari SEED_PASSWORD di .env).
SET NOCOUNT ON;
SET XACT_ABORT ON;

DECLARE @seedUsers TABLE (
    company_code VARCHAR(20) NOT NULL,
    unit_ref NVARCHAR(150) NOT NULL,
    position_name NVARCHAR(150) NOT NULL,
    role_code VARCHAR(30) NOT NULL,
    employee_number VARCHAR(30) NOT NULL PRIMARY KEY,
    name NVARCHAR(150) NOT NULL,
    email NVARCHAR(150) NOT NULL,
    is_active BIT NOT NULL
);

INSERT INTO @seedUsers (company_code, unit_ref, position_name, role_code, employee_number, name, email, is_active)
VALUES
    -- Direksi PSP
    ('PSP', N'Direktorat Utama',                              N'Direktur Utama',                  'PEGAWAI',  '1887001', N'Ir. Hendra Gunawan, M.T.',     'hendra.gunawan@pusri.test',       1),
    ('PSP', N'Direktorat Keuangan & Umum',                    N'Direktur Keuangan & Umum',        'PEGAWAI',  '1889002', N'Dra. Ratna Kusumawati, M.M.',  'ratna.kusumawati@pusri.test',     1),
    ('PSP', N'Direktorat Operasi & Produksi',                 N'Direktur Operasi & Produksi',     'PEGAWAI',  '1888003', N'Ir. Bambang Sutrisno',         'bambang.sutrisno@pusri.test',     1),
    -- Sekretaris Perusahaan
    ('PSP', N'Kompartemen Sekretaris Perusahaan',             N'SVP Sekretaris Perusahaan',       'PEGAWAI',  '1992014', N'Nurul Hidayah, S.H.',          'nurul.hidayah@pusri.test',        1),
    ('PSP', N'Departemen Komunikasi & Administrasi Korporat', N'Staf Tata Usaha',                 'PEGAWAI',  '2016163', N'Wahyu Hidayat',                'wahyu.hidayat@pusri.test',        1),
    -- Teknologi Informasi (alur surat utama)
    ('PSP', N'Kompartemen Teknologi & K3LH',                  N'SVP Teknologi & K3LH',            'PEGAWAI',  '2004071', N'Yudi Pratama, S.T.',           'yudi.pratama@pusri.test',         1),
    ('PSP', N'Departemen Teknologi Informasi PSP',            N'VP Teknologi Informasi',          'PEGAWAI',  '1995021', N'Agus Setiawan, S.Kom., M.T.',  'agus.setiawan@pusri.test',        1),
    ('PSP', N'Fungsional Digitalisasi TI',                    N'AVP Digitalisasi TI',             'PEGAWAI',  '2003035', N'Dewi Lestari, S.Kom.',         'dewi.lestari@pusri.test',         1),
    ('PSP', N'Departemen Teknologi Informasi PSP',            N'Staf Pengembangan Aplikasi',      'PEGAWAI',  '2015047', N'Andi Kurniawan',               'andi.kurniawan@pusri.test',       1),
    ('PSP', N'Departemen Teknologi Informasi PSP',            N'Staf Pengembangan Aplikasi',      'PEGAWAI',  '2017052', N'Rina Marlina',                 'rina.marlina@pusri.test',         1),
    ('PSP', N'Fungsional Digitalisasi TI',                    N'Analis Sistem Informasi',         'PEGAWAI',  '2012060', N'Fajar Nugroho',                'fajar.nugroho@pusri.test',        1),
    ('PSP', N'Departemen Teknologi Informasi PSP',            N'Staf Pengembangan Aplikasi',      'PEGAWAI',  '1990170', N'Slamet Riyadi',                'slamet.riyadi@pusri.test',        0),
    ('PSP', N'Departemen Teknologi Informasi PSP',            N'Staf Infrastruktur TI',           'PEGAWAI',  '2018083', N'Eka Fitriani',                 'eka.fitriani@pusri.test',         1),
    ('PSP', N'Departemen Teknologi Informasi PSP',            N'Administrator Sistem',            'ADMIN',    '2014090', N'Rudi Hartono',                 'rudi.hartono@pusri.test',         1),
    ('PSP', N'Departemen Teknologi Informasi PSP',            N'Administrator Sistem',            'SUPERADMIN', '2014091', N'Arif Wicaksono',             'arif.wicaksono@pusri.test',       1),
    -- SDM dan Keuangan (penerima surat dan disposisi)
    ('PSP', N'Departemen Manajemen & Pengembangan SDM',       N'VP Manajemen & Pengembangan SDM', 'PEGAWAI',  '1998102', N'Sri Wahyuni, S.Psi.',          'sri.wahyuni@pusri.test',          1),
    ('PSP', N'Departemen Manajemen & Pengembangan SDM',       N'Staf Pengembangan SDM',           'PEGAWAI',  '2019110', N'Maya Anggraini',               'maya.anggraini@pusri.test',       1),
    ('PSP', N'Departemen Akuntansi',                          N'VP Akuntansi',                    'PEGAWAI',  '1997118', N'Joko Susilo, S.E., Ak.',       'joko.susilo@pusri.test',          1),
    ('PSP', N'Departemen Anggaran',                           N'Staf Anggaran',                   'PEGAWAI',  '2020125', N'Indah Permatasari',            'indah.permatasari@pusri.test',    1),
    -- Operasi pabrik
    ('PSP', N'Kompartemen Operasi',                           N'SVP Operasi',                     'PEGAWAI',  '1993130', N'Ir. Teguh Wibowo',             'teguh.wibowo@pusri.test',         1),
    ('PSP', N'Departemen Operasi Pabrik I',                   N'VP Operasi Pabrik I',             'PEGAWAI',  '2001142', N'Hadi Purnomo, S.T.',           'hadi.purnomo@pusri.test',         1),
    ('PSP', N'Departemen Operasi Pabrik I',                   N'Staf Operasi Pabrik',             'PEGAWAI',  '2021150', N'Dimas Saputra',                'dimas.saputra@pusri.test',        1),
    -- Petrokimia Gresik (fiktif, untuk uji pemisahan data antar perusahaan)
    ('PKG', 'DKU',                                            N'Direktur Keuangan dan Umum',      'PEGAWAI',  '1990201', N'Drs. Suryo Atmojo, M.M.',      'suryo.atmojo@petrokimia.test',    1),
    ('PKG', 'TI',                                             N'VP Teknologi Informasi',          'PEGAWAI',  '2008202', N'Bayu Aji Pamungkas, S.T.',     'bayu.pamungkas@petrokimia.test',  1),
    ('PKG', 'TI',                                             N'Staf Teknologi Informasi',        'PEGAWAI',  '2019203', N'Lestari Ningsih',              'lestari.ningsih@petrokimia.test', 1);

INSERT INTO users (position_id, role_id, employee_number, name, email, password_hash, is_active)
SELECT p.id, r.id, v.employee_number, v.name, v.email, @password_hash, v.is_active
FROM @seedUsers AS v
JOIN units u ON u.company_code = v.company_code AND u.is_active = 1 AND u.deleted_at IS NULL
    AND ((v.company_code = 'PSP' AND u.name = v.unit_ref) OR (v.company_code = 'PKG' AND u.code = v.unit_ref))
JOIN positions p ON p.unit_id = u.id AND p.name = v.position_name AND p.deleted_at IS NULL
JOIN roles r ON r.code = v.role_code AND r.deleted_at IS NULL
WHERE NOT EXISTS (
    SELECT 1 FROM users existing WITH (UPDLOCK, HOLDLOCK)
    WHERE existing.employee_number = v.employee_number
);

-- Periksa akun dalam daftar seed, bukan jumlah seluruh user di database.
-- User tambahan, mutasi, perubahan role/password, dan akun yang sudah dihapus tidak ditimpa.
IF EXISTS (
    SELECT 1 FROM @seedUsers v
    WHERE NOT EXISTS (SELECT 1 FROM users existing WHERE existing.employee_number = v.employee_number)
)
    THROW 50103, N'Seed user gagal: ada akun yang belum dapat dibuat. Periksa unit, jabatan, dan role pada data acuan.', 1;
