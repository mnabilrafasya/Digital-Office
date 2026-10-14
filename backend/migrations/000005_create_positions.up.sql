CREATE TABLE positions (
    id            BIGINT IDENTITY(1,1) NOT NULL,
    unit_id       BIGINT         NOT NULL,
    name          NVARCHAR(150)  NOT NULL,
    is_unit_head  BIT            NOT NULL CONSTRAINT df_positions_is_unit_head DEFAULT 0,
    is_active     BIT            NOT NULL CONSTRAINT df_positions_is_active    DEFAULT 1,
    created_at    DATETIMEOFFSET NOT NULL CONSTRAINT df_positions_created_at   DEFAULT SYSDATETIMEOFFSET(),
    updated_at    DATETIMEOFFSET NOT NULL CONSTRAINT df_positions_updated_at   DEFAULT SYSDATETIMEOFFSET(),
    deleted_at    DATETIMEOFFSET NULL,
    CONSTRAINT pk_positions PRIMARY KEY (id),
    CONSTRAINT fk_positions_unit FOREIGN KEY (unit_id) REFERENCES units (id)
);
CREATE INDEX ix_positions_unit ON positions (unit_id);