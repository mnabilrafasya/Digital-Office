-- Login sementara sebelum SSO tersedia. Berisi hash bcrypt, BUKAN password asli.
-- NULL = user tanpa password lokal (nanti login lewat SSO).
ALTER TABLE users ADD password_hash VARCHAR(255) NULL;
