-- Riwayat dokumen. Tidak boleh di-soft delete (dijaga di Go).
CREATE TABLE letter_versions (
    id                        BIGINT IDENTITY(1,1) NOT NULL,
    letter_id                 BIGINT           NOT NULL,
    version_number            INT              NOT NULL,
    version_type              VARCHAR(10)      NOT NULL,
    object_key                UNIQUEIDENTIFIER NOT NULL,
    file_name                 NVARCHAR(255)    NOT NULL,
    file_extension            VARCHAR(10)      NOT NULL,
    file_size                 BIGINT           NOT NULL,
    checksum                  VARCHAR(64)      NOT NULL,
    created_by                BIGINT           NOT NULL,
    created_by_role           VARCHAR(20)      NOT NULL,
    created_at                DATETIMEOFFSET NOT NULL CONSTRAINT df_letter_versions_created_at DEFAULT SYSDATETIMEOFFSET(),
    updated_at                DATETIMEOFFSET NOT NULL CONSTRAINT df_letter_versions_updated_at DEFAULT SYSDATETIMEOFFSET(),
    deleted_at                DATETIMEOFFSET NULL,
    CONSTRAINT pk_letter_versions PRIMARY KEY (id),
    CONSTRAINT fk_letter_versions_letter     FOREIGN KEY (letter_id)  REFERENCES letters (id),
    CONSTRAINT fk_letter_versions_created_by FOREIGN KEY (created_by) REFERENCES users (id),
    CONSTRAINT uq_letter_versions_letter_version UNIQUE (letter_id, version_number),
    CONSTRAINT uq_letter_versions_object_key     UNIQUE (object_key),
    CONSTRAINT ck_letter_versions_type      CHECK (version_type IN ('DRAFT','FINAL')),
    CONSTRAINT ck_letter_versions_role      CHECK (created_by_role IN ('DRAFTER','REVIEWER','APPROVER')),
    CONSTRAINT ck_letter_versions_number    CHECK (version_number >= 1),
    CONSTRAINT ck_letter_versions_file_size CHECK (file_size >= 0)
);

-- Dicari saat verifikasi keaslian (QR / unggah file)
CREATE INDEX ix_letter_versions_checksum ON letter_versions (checksum);
