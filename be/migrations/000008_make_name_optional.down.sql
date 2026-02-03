UPDATE users SET name = split_part(email, '@', 1) WHERE name IS NULL OR name = '';
ALTER TABLE users ALTER COLUMN name SET NOT NULL;
ALTER TABLE users ALTER COLUMN name DROP DEFAULT;
