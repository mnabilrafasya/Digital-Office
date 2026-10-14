SET NOCOUNT ON;
SET XACT_ABORT ON;

-- ---------------------------------------------------------------------
-- 1. Perusahaan
-- ---------------------------------------------------------------------
INSERT INTO companies (code, name, city, letter_number_format) VALUES
    ('PSP', N'PT Pupuk Sriwidjaja Palembang', N'Palembang', '{SEQ}/{UNIT}/{COMP}/{MM_ROMAN}/{YYYY}'),
    ('PKG', N'PT Petrokimia Gresik',          N'Gresik',    '{SEQ}/{UNIT}/{COMP}/{MM_ROMAN}/{YYYY}');

-- ---------------------------------------------------------------------
-- 2. Unit kerja (Direktorat -> Kompartemen/Departemen -> Bagian)
-- ---------------------------------------------------------------------
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

-- ---------------------------------------------------------------------
-- 3. Jabatan (is_unit_head = 1 -> pimpinan unit, penerima surat masuk)
-- ---------------------------------------------------------------------
INSERT INTO positions (unit_id, name, is_unit_head)
SELECT u.id, v.name, v.is_unit_head
FROM (VALUES
    ('PSP', 'DU',  N'Direktur Utama',                        1),
    ('PSP', 'DKU', N'Direktur Keuangan dan Umum',            1),
    ('PSP', 'DOP', N'Direktur Operasi dan Produksi',         1),
    ('PSP', 'SP',  N'SVP Sekretariat Perusahaan',            1),
    ('PSP', 'SP',  N'Staf Tata Usaha',                       0),
    ('PSP', 'TI',  N'SVP Teknologi Informasi',               1),
    ('PSP', 'SDM', N'VP Sumber Daya Manusia',                1),
    ('PSP', 'SDM', N'Staf Pengembangan SDM',                 0),
    ('PSP', 'AK',  N'VP Akuntansi dan Keuangan',             1),
    ('PSP', 'AK',  N'Staf Anggaran',                         0),
    ('PSP', 'PRD', N'SVP Produksi',                          1),
    ('PSP', 'PA',  N'AVP Pengembangan Aplikasi',             1),
    ('PSP', 'PA',  N'Staf Pengembangan Aplikasi',            0),
    ('PSP', 'PA',  N'Analis Sistem Informasi',               0),
    ('PSP', 'IKT', N'AVP Infrastruktur dan Keamanan TI',     1),
    ('PSP', 'IKT', N'Staf Infrastruktur TI',                 0),
    ('PSP', 'IKT', N'Administrator Sistem',                  0),
    ('PSP', 'OP',  N'AVP Operasi Pabrik',                    1),
    ('PSP', 'OP',  N'Staf Operasi Pabrik',                   0),
    ('PKG', 'DKU', N'Direktur Keuangan dan Umum',            1),
    ('PKG', 'TI',  N'VP Teknologi Informasi',                1),
    ('PKG', 'TI',  N'Staf Teknologi Informasi',              0)
) AS v (company_code, unit_code, name, is_unit_head)
JOIN units u ON u.company_code = v.company_code AND u.code = v.unit_code;

-- ---------------------------------------------------------------------
-- 4. Role dan permission (1 user = 1 role)
-- ---------------------------------------------------------------------
INSERT INTO roles (code, name) VALUES
    ('ADMIN',    N'Administrator Sistem'),
    ('PIMPINAN', N'Pimpinan / Pejabat Struktural'),
    ('PEGAWAI',  N'Pegawai');

INSERT INTO permissions (role_id, code, name)
SELECT r.id, v.code, v.name
FROM (VALUES
    ('ADMIN',    'LETTER_CREATE',   N'Membuat dan mengajukan surat'),
    ('ADMIN',    'LETTER_REVIEW',   N'Dapat dipilih sebagai reviewer'),
    ('ADMIN',    'TEMPLATE_CREATE', N'Membuat template surat'),
    ('ADMIN',    'AUDIT_VIEW',      N'Melihat log sistem'),
    ('PIMPINAN', 'LETTER_CREATE',   N'Membuat dan mengajukan surat'),
    ('PIMPINAN', 'LETTER_REVIEW',   N'Dapat dipilih sebagai reviewer'),
    ('PIMPINAN', 'LETTER_APPROVE',  N'Dapat dipilih sebagai approver (penandatangan)'),
    ('PIMPINAN', 'INBOX_DISPOSE',   N'Mendisposisikan surat masuk'),
    ('PIMPINAN', 'TEMPLATE_CREATE', N'Membuat template surat'),
    ('PEGAWAI',  'LETTER_CREATE',   N'Membuat dan mengajukan surat'),
    ('PEGAWAI',  'LETTER_REVIEW',   N'Dapat dipilih sebagai reviewer'),
    ('PEGAWAI',  'TEMPLATE_CREATE', N'Membuat template surat')
) AS v (role_code, code, name)
JOIN roles r ON r.code = v.role_code;

-- ---------------------------------------------------------------------
-- 5. Pengguna
-- ---------------------------------------------------------------------
INSERT INTO users (position_id, role_id, employee_number, name, email, password_hash, is_active)
SELECT p.id, r.id, v.employee_number, v.name, v.email, @password_hash, v.is_active
FROM (VALUES
    -- Direksi PSP
    ('PSP', 'DU',  N'Direktur Utama',                    'PIMPINAN', '1887001', N'Ir. Hendra Gunawan, M.T.',     'hendra.gunawan@pusri.test',     1),
    ('PSP', 'DKU', N'Direktur Keuangan dan Umum',        'PIMPINAN', '1889002', N'Dra. Ratna Kusumawati, M.M.',  'ratna.kusumawati@pusri.test',   1),
    ('PSP', 'DOP', N'Direktur Operasi dan Produksi',     'PIMPINAN', '1888003', N'Ir. Bambang Sutrisno',         'bambang.sutrisno@pusri.test',   1),
    -- Sekretariat Perusahaan
    ('PSP', 'SP',  N'SVP Sekretariat Perusahaan',        'PIMPINAN', '1992014', N'Nurul Hidayah, S.H.',          'nurul.hidayah@pusri.test',      1),
    ('PSP', 'SP',  N'Staf Tata Usaha',                   'PEGAWAI',  '2016163', N'Wahyu Hidayat',                'wahyu.hidayat@pusri.test',      1),
    -- Teknologi Informasi
    ('PSP', 'TI',  N'SVP Teknologi Informasi',           'PIMPINAN', '1995021', N'Agus Setiawan, S.Kom., M.T.',  'agus.setiawan@pusri.test',      1),
    ('PSP', 'PA',  N'AVP Pengembangan Aplikasi',         'PIMPINAN', '2003035', N'Dewi Lestari, S.Kom.',         'dewi.lestari@pusri.test',       1),
    ('PSP', 'PA',  N'Staf Pengembangan Aplikasi',        'PEGAWAI',  '2015047', N'Andi Kurniawan',               'andi.kurniawan@pusri.test',     1),
    ('PSP', 'PA',  N'Staf Pengembangan Aplikasi',        'PEGAWAI',  '2017052', N'Rina Marlina',                 'rina.marlina@pusri.test',       1),
    ('PSP', 'PA',  N'Analis Sistem Informasi',           'PEGAWAI',  '2012060', N'Fajar Nugroho',                'fajar.nugroho@pusri.test',      1),
    ('PSP', 'PA',  N'Staf Pengembangan Aplikasi',        'PEGAWAI',  '1990170', N'Slamet Riyadi',                'slamet.riyadi@pusri.test',      0),
    ('PSP', 'IKT', N'AVP Infrastruktur dan Keamanan TI', 'PIMPINAN', '2004071', N'Yudi Pratama, S.T.',           'yudi.pratama@pusri.test',       1),
    ('PSP', 'IKT', N'Staf Infrastruktur TI',             'PEGAWAI',  '2018083', N'Eka Fitriani',                 'eka.fitriani@pusri.test',       1),
    ('PSP', 'IKT', N'Administrator Sistem',              'ADMIN',    '2014090', N'Rudi Hartono',                 'rudi.hartono@pusri.test',       1),
    -- SDM
    ('PSP', 'SDM', N'VP Sumber Daya Manusia',            'PIMPINAN', '1998102', N'Sri Wahyuni, S.Psi.',          'sri.wahyuni@pusri.test',        1),
    ('PSP', 'SDM', N'Staf Pengembangan SDM',             'PEGAWAI',  '2019110', N'Maya Anggraini',               'maya.anggraini@pusri.test',     1),
    -- Akuntansi dan Keuangan
    ('PSP', 'AK',  N'VP Akuntansi dan Keuangan',         'PIMPINAN', '1997118', N'Joko Susilo, S.E., Ak.',       'joko.susilo@pusri.test',        1),
    ('PSP', 'AK',  N'Staf Anggaran',                     'PEGAWAI',  '2020125', N'Indah Permatasari',            'indah.permatasari@pusri.test',  1),
    -- Produksi
    ('PSP', 'PRD', N'SVP Produksi',                      'PIMPINAN', '1993130', N'Ir. Teguh Wibowo',             'teguh.wibowo@pusri.test',       1),
    ('PSP', 'OP',  N'AVP Operasi Pabrik',                'PIMPINAN', '2001142', N'Hadi Purnomo, S.T.',           'hadi.purnomo@pusri.test',       1),
    ('PSP', 'OP',  N'Staf Operasi Pabrik',               'PEGAWAI',  '2021150', N'Dimas Saputra',                'dimas.saputra@pusri.test',      1),
    -- Petrokimia Gresik (untuk uji pemisahan data antar perusahaan)
    ('PKG', 'DKU', N'Direktur Keuangan dan Umum',        'PIMPINAN', '1990201', N'Drs. Suryo Atmojo, M.M.',      'suryo.atmojo@petrokimia.test',  1),
    ('PKG', 'TI',  N'VP Teknologi Informasi',            'PIMPINAN', '2008202', N'Bayu Aji Pamungkas, S.T.',     'bayu.pamungkas@petrokimia.test',1),
    ('PKG', 'TI',  N'Staf Teknologi Informasi',          'PEGAWAI',  '2019203', N'Lestari Ningsih',              'lestari.ningsih@petrokimia.test',1)
) AS v (company_code, unit_code, position_name, role_code, employee_number, name, email, is_active)
JOIN units     u ON u.company_code = v.company_code AND u.code = v.unit_code
JOIN positions p ON p.unit_id = u.id AND p.name = v.position_name
JOIN roles     r ON r.code = v.role_code;

-- ---------------------------------------------------------------------
-- 6. Lookups 
-- ---------------------------------------------------------------------
INSERT INTO lookups (company_code, category, code, name, color, sort_order, note) VALUES
    -- Sifat
    (NULL, 'CONFIDENTIALITY', 'BIASA',    N'Biasa',    '#6B7280', 1, N'Boleh diketahui umum di internal perusahaan'),
    (NULL, 'CONFIDENTIALITY', 'TERBATAS', N'Terbatas', '#F59E0B', 2, N'Hanya untuk unit atau pejabat terkait'),
    (NULL, 'CONFIDENTIALITY', 'RAHASIA',  N'Rahasia',  '#DC2626', 3, N'Hanya untuk penerima yang dituju'),
    -- Kecepatan
    (NULL, 'URGENCY', 'BIASA',         N'Biasa',         '#6B7280', 1, N'Ditanggapi sesuai antrean, maksimal 5 hari kerja'),
    (NULL, 'URGENCY', 'SEGERA',        N'Segera',        '#F59E0B', 2, N'Ditanggapi maksimal 2x24 jam'),
    (NULL, 'URGENCY', 'SANGAT_SEGERA', N'Sangat Segera', '#DC2626', 3, N'Ditanggapi maksimal 1x24 jam'),
    -- Jenis surat
    (NULL, 'LETTER_TYPE', 'NOTA_DINAS',       N'Nota Dinas',       NULL, 1, N'Komunikasi resmi antar unit atau pejabat'),
    (NULL, 'LETTER_TYPE', 'MEMO',             N'Memo Internal',    NULL, 2, N'Pesan singkat, misalnya dari atasan ke bawahan'),
    (NULL, 'LETTER_TYPE', 'SURAT_UNDANGAN',   N'Surat Undangan',   NULL, 3, N'Mengundang rapat atau acara'),
    (NULL, 'LETTER_TYPE', 'SURAT_EDARAN',     N'Surat Edaran',     NULL, 4, N'Pemberitahuan atau kebijakan untuk banyak unit'),
    (NULL, 'LETTER_TYPE', 'SURAT_PERMOHONAN', N'Surat Permohonan', NULL, 5, N'Meminta dana, akses, atau dukungan'),
    (NULL, 'LETTER_TYPE', 'SURAT_TUGAS',      N'Surat Tugas',      NULL, 6, N'Menugaskan pegawai untuk kegiatan tertentu'),
    (NULL, 'LETTER_TYPE', 'SURAT_KEPUTUSAN',  N'Surat Keputusan',  NULL, 7, N'Menetapkan keputusan resmi'),
    (NULL, 'LETTER_TYPE', 'BERITA_ACARA',     N'Berita Acara',     NULL, 8, N'Mencatat kejadian, serah terima, atau hasil pemeriksaan'),
    -- Status surat 
    (NULL, 'LETTER_STATUS', 'DRAFT',                N'Draf',                     '#6B7280', 1, NULL),
    (NULL, 'LETTER_STATUS', 'IN_REVIEW',            N'Sedang Direview',          '#2563EB', 2, NULL),
    (NULL, 'LETTER_STATUS', 'RETURNED_TO_DRAFTER',  N'Dikembalikan ke Drafter',  '#D97706', 3, NULL),
    (NULL, 'LETTER_STATUS', 'IN_APPROVAL',          N'Menunggu Persetujuan',     '#7C3AED', 4, NULL),
    (NULL, 'LETTER_STATUS', 'RETURNED_TO_REVIEWER', N'Dikembalikan ke Reviewer', '#EA580C', 5, NULL),
    (NULL, 'LETTER_STATUS', 'SENT',                 N'Terkirim',                 '#16A34A', 6, NULL),
    (NULL, 'LETTER_STATUS', 'CANCELLED',            N'Dibatalkan',               '#DC2626', 7, NULL),
    -- Status inbox
    (NULL, 'INBOX_STATUS', 'UNREAD',     N'Belum Dibaca',      '#2563EB', 1, NULL),
    (NULL, 'INBOX_STATUS', 'READ',       N'Sudah Dibaca',      '#6B7280', 2, NULL),
    (NULL, 'INBOX_STATUS', 'DISPOSED',   N'Sudah Didisposisi', '#16A34A', 3, NULL),
    (NULL, 'INBOX_STATUS', 'REDIRECTED', N'Dialihkan',         '#7C3AED', 4, NULL);

-- Klasifikasi arsip (per perusahaan)
INSERT INTO lookups (company_code, category, code, name, sort_order) VALUES
    ('PSP', 'CLASSIFICATION', 'UM.01',  N'Persuratan dan Kearsipan',               1),
    ('PSP', 'CLASSIFICATION', 'UM.02',  N'Rapat dan Koordinasi',                   2),
    ('PSP', 'CLASSIFICATION', 'UM.03',  N'Fasilitas dan Rumah Tangga Kantor',      3),
    ('PSP', 'CLASSIFICATION', 'SDM.01', N'Rekrutmen dan Penempatan',               4),
    ('PSP', 'CLASSIFICATION', 'SDM.02', N'Pelatihan dan Pengembangan',             5),
    ('PSP', 'CLASSIFICATION', 'SDM.03', N'Kesejahteraan dan Penggajian',           6),
    ('PSP', 'CLASSIFICATION', 'KU.01',  N'Anggaran',                               7),
    ('PSP', 'CLASSIFICATION', 'KU.02',  N'Pembayaran dan Perbendaharaan',          8),
    ('PSP', 'CLASSIFICATION', 'KU.03',  N'Akuntansi dan Pelaporan',                9),
    ('PSP', 'CLASSIFICATION', 'PG.01',  N'Perencanaan Pengadaan',                 10),
    ('PSP', 'CLASSIFICATION', 'PG.02',  N'Pelaksanaan Pengadaan Barang dan Jasa', 11),
    ('PSP', 'CLASSIFICATION', 'PG.03',  N'Pengelolaan Vendor',                    12),
    ('PSP', 'CLASSIFICATION', 'HK.01',  N'Peraturan dan Kebijakan Internal',      13),
    ('PSP', 'CLASSIFICATION', 'HK.02',  N'Perjanjian dan Kontrak',                14),
    ('PSP', 'CLASSIFICATION', 'HK.03',  N'Kepatuhan dan Tata Kelola (GCG)',       15),
    ('PSP', 'CLASSIFICATION', 'TI.01',  N'Tata Kelola dan Kebijakan TI',          16),
    ('PSP', 'CLASSIFICATION', 'TI.02',  N'Pengembangan Sistem Aplikasi',          17),
    ('PSP', 'CLASSIFICATION', 'TI.03',  N'Infrastruktur dan Jaringan',            18),
    ('PSP', 'CLASSIFICATION', 'TI.04',  N'Keamanan Informasi',                    19),
    ('PSP', 'CLASSIFICATION', 'PW.01',  N'Audit Internal',                        20),
    ('PSP', 'CLASSIFICATION', 'PW.02',  N'Manajemen Risiko',                      21),
    ('PSP', 'CLASSIFICATION', 'PR.01',  N'Perencanaan Produksi',                  22),
    ('PSP', 'CLASSIFICATION', 'PR.02',  N'Pemeliharaan Pabrik',                   23),
    ('PSP', 'CLASSIFICATION', 'PR.03',  N'Pengendalian Mutu',                     24),
    ('PSP', 'CLASSIFICATION', 'PS.01',  N'Penjualan',                             25),
    ('PSP', 'CLASSIFICATION', 'PS.02',  N'Distribusi dan Logistik',               26),
    ('PSP', 'CLASSIFICATION', 'K3.01',  N'Keselamatan dan Kesehatan Kerja',       27),
    ('PSP', 'CLASSIFICATION', 'K3.02',  N'Pengelolaan Lingkungan',                28),
    ('PKG', 'CLASSIFICATION', 'UM.01',  N'Persuratan dan Kearsipan',               1),
    ('PKG', 'CLASSIFICATION', 'KU.01',  N'Anggaran',                               2),
    ('PKG', 'CLASSIFICATION', 'TI.01',  N'Tata Kelola dan Kebijakan TI',           3);