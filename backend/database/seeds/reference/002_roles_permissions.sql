-- 002_roles_permissions.sql
-- Role (1 user = 1 role) dan permission per role. Idempoten (MERGE), aman dijalankan berulang.
--
-- Hak akses user = permission role + permission dari posisi (level jabatan). SUPERADMIN punya seluruh hak.
-- Pada role lain, approve dan disposisi otomatis dimiliki posisi DIREKTUR, SVP, VP, dan AVP
-- (lihat internal/auth/hakakses.go). Role hanya menentukan akses sistem:
--   SUPERADMIN  seluruh permission: master data, surat, approve, disposisi, audit
--   ADMIN       kelola internal perusahaannya (pengaturan, unit, jabatan, user, template) + audit
--   PEGAWAI     pemakai biasa
SET NOCOUNT ON;
SET XACT_ABORT ON;

MERGE roles WITH (HOLDLOCK) AS t
USING (VALUES
    ('SUPERADMIN', N'Super Administrator'),
    ('ADMIN',      N'Administrator Sistem'),
    ('PEGAWAI',    N'Pegawai')
) AS s (code, name)
    ON t.code = s.code AND t.deleted_at IS NULL
WHEN MATCHED AND t.name <> s.name THEN
    UPDATE SET name = s.name, updated_at = SYSDATETIMEOFFSET()
WHEN NOT MATCHED BY TARGET THEN
    INSERT (code, name) VALUES (s.code, s.name);

MERGE permissions WITH (HOLDLOCK) AS t
USING (
    SELECT r.id AS role_id, v.code, v.name
    FROM (VALUES
        ('SUPERADMIN', 'LETTER_CREATE',    N'Membuat dan mengajukan surat'),
        ('SUPERADMIN', 'LETTER_REVIEW',    N'Dapat dipilih sebagai reviewer'),
        ('SUPERADMIN', 'TEMPLATE_CREATE',  N'Membuat template surat'),
        ('SUPERADMIN', 'COMPANY_MANAGE',   N'Mengelola perusahaan dan admin pertamanya'),
        ('SUPERADMIN', 'COMPANY_SETTINGS', N'Mengatur perusahaan'),
        ('SUPERADMIN', 'ORG_MANAGE',       N'Mengelola unit dan jabatan'),
        ('SUPERADMIN', 'USER_MANAGE',      N'Mengelola user'),
        ('SUPERADMIN', 'TEMPLATE_MANAGE',  N'Mengelola status template'),
        ('SUPERADMIN', 'AUDIT_VIEW',       N'Melihat log sistem'),
        ('SUPERADMIN', 'LETTER_APPROVE',   N'Menyetujui surat yang ditugaskan'),
        ('SUPERADMIN', 'INBOX_DISPOSE',    N'Mendisposisikan surat masuk'),
        ('ADMIN',      'LETTER_CREATE',    N'Membuat dan mengajukan surat'),
        ('ADMIN',      'LETTER_REVIEW',    N'Dapat dipilih sebagai reviewer'),
        ('ADMIN',      'TEMPLATE_CREATE',  N'Membuat template surat'),
        ('ADMIN',      'AUDIT_VIEW',       N'Melihat log sistem'),
        ('ADMIN',      'COMPANY_SETTINGS', N'Mengatur watermark dan format nomor surat perusahaan'),
        ('ADMIN',      'ORG_MANAGE',       N'Mengelola unit dan jabatan'),
        ('ADMIN',      'USER_MANAGE',      N'Mengelola user'),
        ('ADMIN',      'TEMPLATE_MANAGE',  N'Mengaktifkan dan menonaktifkan template surat'),
        ('PEGAWAI',    'LETTER_CREATE',    N'Membuat dan mengajukan surat'),
        ('PEGAWAI',    'LETTER_REVIEW',    N'Dapat dipilih sebagai reviewer'),
        ('PEGAWAI',    'TEMPLATE_CREATE',  N'Membuat template surat')
    ) AS v (role_code, code, name)
    JOIN roles r ON r.code = v.role_code AND r.deleted_at IS NULL
) AS s
    ON t.role_id = s.role_id AND t.code = s.code AND t.deleted_at IS NULL
WHEN MATCHED AND t.name <> s.name THEN
    UPDATE SET name = s.name, updated_at = SYSDATETIMEOFFSET()
WHEN NOT MATCHED BY TARGET THEN
    INSERT (role_id, code, name) VALUES (s.role_id, s.code, s.name);

-- Role PIMPINAN (versi lama) dihapus: kewenangan pimpinan kini datang dari posisi. Pemegangnya
-- dipindah ke PEGAWAI, lalu role dan permission-nya dinonaktifkan (tidak dihapus, demi audit lama).
UPDATE u
SET role_id = peg.id, updated_at = SYSDATETIMEOFFSET()
FROM users u
JOIN roles old ON old.id = u.role_id AND old.code = 'PIMPINAN'
CROSS JOIN (SELECT id FROM roles WHERE code = 'PEGAWAI' AND deleted_at IS NULL) AS peg;

UPDATE pm
SET deleted_at = SYSDATETIMEOFFSET(), updated_at = SYSDATETIMEOFFSET()
FROM permissions pm
JOIN roles old ON old.id = pm.role_id AND old.code = 'PIMPINAN'
WHERE pm.deleted_at IS NULL;

UPDATE roles
SET is_active = 0, deleted_at = SYSDATETIMEOFFSET(), updated_at = SYSDATETIMEOFFSET()
WHERE code = 'PIMPINAN' AND deleted_at IS NULL;
