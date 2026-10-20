-- 003_users.sql
-- DATA DUMMY DEVELOPMENT: pengguna. Semua nama, NIK, dan email FIKTIF.
-- @password_hash dikirim oleh cmd/seed (hasil bcrypt dari SEED_PASSWORD di .env).
SET NOCOUNT ON;
SET XACT_ABORT ON;

INSERT INTO users (position_id, role_id, employee_number, name, email, password_hash, is_active)
SELECT p.id, r.id, v.employee_number, v.name, v.email, @password_hash, v.is_active
FROM (VALUES
    -- Direksi PSP
    ('PSP', 'DU',  N'Direktur Utama',                    'PIMPINAN', '1887001', N'Ir. Hendra Gunawan, M.T.',     'hendra.gunawan@pusri.test',      1),
    ('PSP', 'DKU', N'Direktur Keuangan dan Umum',        'PIMPINAN', '1889002', N'Dra. Ratna Kusumawati, M.M.',  'ratna.kusumawati@pusri.test',    1),
    ('PSP', 'DOP', N'Direktur Operasi dan Produksi',     'PIMPINAN', '1888003', N'Ir. Bambang Sutrisno',         'bambang.sutrisno@pusri.test',    1),
    -- Sekretariat Perusahaan
    ('PSP', 'SP',  N'SVP Sekretariat Perusahaan',        'PIMPINAN', '1992014', N'Nurul Hidayah, S.H.',          'nurul.hidayah@pusri.test',       1),
    ('PSP', 'SP',  N'Staf Tata Usaha',                   'PEGAWAI',  '2016163', N'Wahyu Hidayat',                'wahyu.hidayat@pusri.test',       1),
    -- Teknologi Informasi
    ('PSP', 'TI',  N'SVP Teknologi Informasi',           'PIMPINAN', '1995021', N'Agus Setiawan, S.Kom., M.T.',  'agus.setiawan@pusri.test',       1),
    ('PSP', 'PA',  N'AVP Pengembangan Aplikasi',         'PIMPINAN', '2003035', N'Dewi Lestari, S.Kom.',         'dewi.lestari@pusri.test',        1),
    ('PSP', 'PA',  N'Staf Pengembangan Aplikasi',        'PEGAWAI',  '2015047', N'Andi Kurniawan',               'andi.kurniawan@pusri.test',      1),
    ('PSP', 'PA',  N'Staf Pengembangan Aplikasi',        'PEGAWAI',  '2017052', N'Rina Marlina',                 'rina.marlina@pusri.test',        1),
    ('PSP', 'PA',  N'Analis Sistem Informasi',           'PEGAWAI',  '2012060', N'Fajar Nugroho',                'fajar.nugroho@pusri.test',       1),
    ('PSP', 'PA',  N'Staf Pengembangan Aplikasi',        'PEGAWAI',  '1990170', N'Slamet Riyadi',                'slamet.riyadi@pusri.test',       0),
    ('PSP', 'IKT', N'AVP Infrastruktur dan Keamanan TI', 'PIMPINAN', '2004071', N'Yudi Pratama, S.T.',           'yudi.pratama@pusri.test',        1),
    ('PSP', 'IKT', N'Staf Infrastruktur TI',             'PEGAWAI',  '2018083', N'Eka Fitriani',                 'eka.fitriani@pusri.test',        1),
    ('PSP', 'IKT', N'Administrator Sistem',              'ADMIN',    '2014090', N'Rudi Hartono',                 'rudi.hartono@pusri.test',        1),
    -- SDM
    ('PSP', 'SDM', N'VP Sumber Daya Manusia',            'PIMPINAN', '1998102', N'Sri Wahyuni, S.Psi.',          'sri.wahyuni@pusri.test',         1),
    ('PSP', 'SDM', N'Staf Pengembangan SDM',             'PEGAWAI',  '2019110', N'Maya Anggraini',               'maya.anggraini@pusri.test',      1),
    -- Akuntansi dan Keuangan
    ('PSP', 'AK',  N'VP Akuntansi dan Keuangan',         'PIMPINAN', '1997118', N'Joko Susilo, S.E., Ak.',       'joko.susilo@pusri.test',         1),
    ('PSP', 'AK',  N'Staf Anggaran',                     'PEGAWAI',  '2020125', N'Indah Permatasari',            'indah.permatasari@pusri.test',   1),
    -- Produksi
    ('PSP', 'PRD', N'SVP Produksi',                      'PIMPINAN', '1993130', N'Ir. Teguh Wibowo',             'teguh.wibowo@pusri.test',        1),
    ('PSP', 'OP',  N'AVP Operasi Pabrik',                'PIMPINAN', '2001142', N'Hadi Purnomo, S.T.',           'hadi.purnomo@pusri.test',        1),
    ('PSP', 'OP',  N'Staf Operasi Pabrik',               'PEGAWAI',  '2021150', N'Dimas Saputra',                'dimas.saputra@pusri.test',       1),
    -- Petrokimia Gresik (untuk uji pemisahan data antar perusahaan)
    ('PKG', 'DKU', N'Direktur Keuangan dan Umum',        'PIMPINAN', '1990201', N'Drs. Suryo Atmojo, M.M.',      'suryo.atmojo@petrokimia.test',   1),
    ('PKG', 'TI',  N'VP Teknologi Informasi',            'PIMPINAN', '2008202', N'Bayu Aji Pamungkas, S.T.',     'bayu.pamungkas@petrokimia.test', 1),
    ('PKG', 'TI',  N'Staf Teknologi Informasi',          'PEGAWAI',  '2019203', N'Lestari Ningsih',              'lestari.ningsih@petrokimia.test',1)
) AS v (company_code, unit_code, position_name, role_code, employee_number, name, email, is_active)
JOIN units     u ON u.company_code = v.company_code AND u.code = v.unit_code
JOIN positions p ON p.unit_id = u.id AND p.name = v.position_name
JOIN roles     r ON r.code = v.role_code AND r.deleted_at IS NULL;

IF (SELECT COUNT(*) FROM users) <> 24 THROW 50103, N'Seed dev gagal: jumlah users harus 24 (cek ejaan unit/jabatan/role)', 1;
