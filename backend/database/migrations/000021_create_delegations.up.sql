-- 000021_create_delegations.up.sql
-- Delegasi tugas: selama start_date..end_date (tanggal penuh, zona waktu aplikasi) tugas review/
-- approval user_id dikerjakan delegate_id atas namanya, dan surat yang masuk ke inbox user_id
-- ditembuskan ke delegate_id. Diisi pegawai sendiri atau admin perusahaan (created_by).
-- Jadwal satu pegawai tidak boleh tumpang-tindih dan tidak boleh berantai (dijaga aplikasi).
SET XACT_ABORT ON;
BEGIN TRANSACTION;

CREATE TABLE delegations (
    id            BIGINT IDENTITY(1,1) NOT NULL,
    company_code  VARCHAR(20)    NOT NULL,
    user_id       BIGINT         NOT NULL,
    delegate_id   BIGINT         NOT NULL,
    start_date    DATE           NOT NULL,
    end_date      DATE           NOT NULL,
    note          NVARCHAR(200)  NULL,
    created_by    BIGINT         NOT NULL,
    created_at    DATETIMEOFFSET NOT NULL CONSTRAINT df_delegations_created_at DEFAULT SYSDATETIMEOFFSET(),
    updated_at    DATETIMEOFFSET NOT NULL CONSTRAINT df_delegations_updated_at DEFAULT SYSDATETIMEOFFSET(),
    deleted_at    DATETIMEOFFSET NULL,
    CONSTRAINT pk_delegations PRIMARY KEY (id),
    CONSTRAINT fk_delegations_company    FOREIGN KEY (company_code) REFERENCES companies (code),
    CONSTRAINT fk_delegations_user       FOREIGN KEY (user_id)      REFERENCES users (id),
    CONSTRAINT fk_delegations_delegate   FOREIGN KEY (delegate_id)  REFERENCES users (id),
    CONSTRAINT fk_delegations_created_by FOREIGN KEY (created_by)   REFERENCES users (id),
    CONSTRAINT ck_delegations_period     CHECK (start_date <= end_date),
    CONSTRAINT ck_delegations_not_self   CHECK (user_id <> delegate_id)
);

-- Delegasi milik seorang pegawai, dan siapa saja yang sedang diwakili seorang pengganti
CREATE INDEX ix_delegations_user     ON delegations (user_id, start_date, end_date)     WHERE deleted_at IS NULL;
CREATE INDEX ix_delegations_delegate ON delegations (delegate_id, start_date, end_date) WHERE deleted_at IS NULL;
-- Daftar delegasi perusahaan (admin)
CREATE INDEX ix_delegations_company  ON delegations (company_code, end_date)            WHERE deleted_at IS NULL;

COMMIT TRANSACTION;
