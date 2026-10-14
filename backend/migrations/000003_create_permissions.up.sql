CREATE TABLE permissions (
    id          BIGINT IDENTITY(1,1) NOT NULL,
    role_id     BIGINT         NOT NULL,
    code        VARCHAR(50)    NOT NULL,
    name        NVARCHAR(150)  NOT NULL,
    created_at  DATETIMEOFFSET NOT NULL CONSTRAINT df_permissions_created_at DEFAULT SYSDATETIMEOFFSET(),
    updated_at  DATETIMEOFFSET NOT NULL CONSTRAINT df_permissions_updated_at DEFAULT SYSDATETIMEOFFSET(),
    deleted_at  DATETIMEOFFSET NULL,
    CONSTRAINT pk_permissions PRIMARY KEY (id),
    CONSTRAINT fk_permissions_role FOREIGN KEY (role_id) REFERENCES roles (id)
);
CREATE UNIQUE INDEX uq_permissions_role_code ON permissions (role_id, code) WHERE deleted_at IS NULL;