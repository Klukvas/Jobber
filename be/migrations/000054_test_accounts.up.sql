-- Administrative analytics metadata only; existing accounts remain included.
ALTER TABLE users ADD COLUMN is_test boolean NOT NULL DEFAULT false;
