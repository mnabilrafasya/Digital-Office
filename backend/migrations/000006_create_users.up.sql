CREATE TABLE users (
    id               BIGINT IDENTITY(1,1) NOT NULL,
    position_id      BIGINT         NOT NULL,
    role_id          BIGINT         NOT NULL,
    employee_number  VARCHAR(30)    NOT NULL,
    name             NVARCHAR(150)  NOT NULL,
    email            NVARCHAR(150)  NOT NULL,
    is_active        BIT            NOT NULL CONSTRAINT df_users_is_active  DEFAULT 1,
    created_at       DATETIMEOFFSET NOT NULL CONSTRAINT df_users_created_at DEFAULT SYSDATETIMEOFFSET(),
    updated_at       DATETIMEOFFSET NOT NULL CONSTRAINT df_users_updated_at DEFAULT SYSDATETIMEOFFSET(),
    deleted_at       DATETIMEOFFSET NULL,
    CONSTRAINT pk_users PRIMARY KEY (id),
    CONSTRAINT fk_users_position FOREIGN KEY (position_id) REFERENCES positions (id),
    CONSTRAINT fk_users_role     FOREIGN KEY (role_id)     REFERENCES roles (id)
);
CREATE UNIQUE INDEX uq_users_employee_number ON users (employee_number) WHERE deleted_at IS NULL;
CREATE INDEX ix_users_position ON users (position_id);
CREATE INDEX ix_users_role     ON users (role_id);