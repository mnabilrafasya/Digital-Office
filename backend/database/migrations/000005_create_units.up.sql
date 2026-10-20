-- 000005_create_units.up.sql
-- Unit kerja bertingkat (Direktorat -> Kompartemen/Departemen -> Bagian) per perusahaan.
SET XACT_ABORT ON;
BEGIN TRANSACTION;

CREATE TABLE units (
    id            BIGINT IDENTITY(1,1) NOT NULL,
    company_code  VARCHAR(20)    NOT NULL,
    parent_id     BIGINT         NULL,
    unit_type     VARCHAR(30)    NOT NULL,
    code          VARCHAR(20)    NOT NULL,
    name          NVARCHAR(150)  NOT NULL,
    is_active     BIT            NOT NULL CONSTRAINT df_units_is_active  DEFAULT 1,
    created_at    DATETIMEOFFSET NOT NULL CONSTRAINT df_units_created_at DEFAULT SYSDATETIMEOFFSET(),
    updated_at    DATETIMEOFFSET NOT NULL CONSTRAINT df_units_updated_at DEFAULT SYSDATETIMEOFFSET(),
    deleted_at    DATETIMEOFFSET NULL,
    CONSTRAINT pk_units PRIMARY KEY (id),
    CONSTRAINT fk_units_company FOREIGN KEY (company_code) REFERENCES companies (code),
    CONSTRAINT fk_units_parent  FOREIGN KEY (parent_id)    REFERENCES units (id),
    CONSTRAINT uq_units_company_code UNIQUE (company_code, code)
);

CREATE INDEX ix_units_parent ON units (parent_id);

COMMIT TRANSACTION;
