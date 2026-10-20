-- 000007_create_users.up.sql
-- Pengguna. password_hash = hash bcrypt untuk login sementara sebelum SSO (NULL = tanpa password lokal).
-- token_version naik saat logout, sehingga semua access token lama tidak berlaku.
SET XACT_ABORT ON;
BEGIN TRANSACTION;

CREATE TABLE users (
    id               BIGINT IDENTITY(1,1) NOT NULL,
    position_id      BIGINT         NOT NULL,
    role_id          BIGINT         NOT NULL,
    employee_number  VARCHAR(30)    NOT NULL,
    name             NVARCHAR(150)  NOT NULL,
    email            NVARCHAR(150)  NOT NULL,
    password_hash    VARCHAR(255)   NULL,
    token_version    INT            NOT NULL CONSTRAINT df_users_token_version DEFAULT 1,
    is_active        BIT            NOT NULL CONSTRAINT df_users_is_active  DEFAULT 1,
    created_at       DATETIMEOFFSET NOT NULL CONSTRAINT df_users_created_at DEFAULT SYSDATETIMEOFFSET(),
    updated_at       DATETIMEOFFSET NOT NULL CONSTRAINT df_users_updated_at DEFAULT SYSDATETIMEOFFSET(),
    deleted_at       DATETIMEOFFSET NULL,
    CONSTRAINT pk_users PRIMARY KEY (id),
    CONSTRAINT fk_users_position FOREIGN KEY (position_id) REFERENCES positions (id),
    CONSTRAINT fk_users_role     FOREIGN KEY (role_id)     REFERENCES roles (id),
    CONSTRAINT ck_users_token_version CHECK (token_version >= 1)
);

CREATE UNIQUE INDEX uq_users_employee_number ON users (employee_number) WHERE deleted_at IS NULL;
CREATE INDEX ix_users_position ON users (position_id);
CREATE INDEX ix_users_role     ON users (role_id);

COMMIT TRANSACTION;
