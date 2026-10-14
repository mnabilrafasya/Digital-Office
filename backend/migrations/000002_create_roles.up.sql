CREATE TABLE roles (
    id          BIGINT IDENTITY(1,1) NOT NULL,
    code        VARCHAR(30)    NOT NULL,
    name        NVARCHAR(100)  NOT NULL,
    is_active   BIT            NOT NULL CONSTRAINT df_roles_is_active  DEFAULT 1,
    created_at  DATETIMEOFFSET NOT NULL CONSTRAINT df_roles_created_at DEFAULT SYSDATETIMEOFFSET(),
    updated_at  DATETIMEOFFSET NOT NULL CONSTRAINT df_roles_updated_at DEFAULT SYSDATETIMEOFFSET(),
    deleted_at  DATETIMEOFFSET NULL,
    CONSTRAINT pk_roles PRIMARY KEY (id)
);
CREATE UNIQUE INDEX uq_roles_code ON roles (code) WHERE deleted_at IS NULL;