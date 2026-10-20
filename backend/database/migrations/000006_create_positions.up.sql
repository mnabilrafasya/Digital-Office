-- 000006_create_positions.up.sql
-- Jabatan di unit. is_unit_head = 1 -> pimpinan unit (penerima surat masuk).
-- level_id menunjuk lookup POSITION_LEVEL milik perusahaan yang sama (dijaga di service dan seeder).
SET XACT_ABORT ON;
BEGIN TRANSACTION;

CREATE TABLE positions (
    id            BIGINT IDENTITY(1,1) NOT NULL,
    unit_id       BIGINT         NOT NULL,
    name          NVARCHAR(150)  NOT NULL,
    is_unit_head  BIT            NOT NULL CONSTRAINT df_positions_is_unit_head DEFAULT 0,
    level_id      BIGINT         NULL,
    is_active     BIT            NOT NULL CONSTRAINT df_positions_is_active    DEFAULT 1,
    created_at    DATETIMEOFFSET NOT NULL CONSTRAINT df_positions_created_at   DEFAULT SYSDATETIMEOFFSET(),
    updated_at    DATETIMEOFFSET NOT NULL CONSTRAINT df_positions_updated_at   DEFAULT SYSDATETIMEOFFSET(),
    deleted_at    DATETIMEOFFSET NULL,
    CONSTRAINT pk_positions PRIMARY KEY (id),
    CONSTRAINT fk_positions_unit  FOREIGN KEY (unit_id)  REFERENCES units (id),
    CONSTRAINT fk_positions_level FOREIGN KEY (level_id) REFERENCES lookups (id)
);

CREATE INDEX ix_positions_unit  ON positions (unit_id);
CREATE INDEX ix_positions_level ON positions (level_id) WHERE deleted_at IS NULL;

COMMIT TRANSACTION;
