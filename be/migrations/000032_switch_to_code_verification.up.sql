-- Delete all existing link-based tokens (they can't work with the new code system)
DELETE FROM email_verification_tokens;
DELETE FROM password_reset_tokens;

-- email_verification_tokens: switch from token_hash to 6-digit code with attempts
ALTER TABLE email_verification_tokens ADD COLUMN code VARCHAR(6) NOT NULL;
ALTER TABLE email_verification_tokens ADD COLUMN attempts INT NOT NULL DEFAULT 0;

-- Drop old token_hash column (this also drops its unique constraint and index)
ALTER TABLE email_verification_tokens DROP COLUMN IF EXISTS token_hash;

-- Add index for lookup by user_id + code
CREATE INDEX idx_email_verification_tokens_user_code ON email_verification_tokens (user_id, code);

-- password_reset_tokens: switch from token_hash to 6-digit code with attempts
ALTER TABLE password_reset_tokens ADD COLUMN code VARCHAR(6) NOT NULL;
ALTER TABLE password_reset_tokens ADD COLUMN attempts INT NOT NULL DEFAULT 0;

-- Drop old token_hash column (this also drops its unique constraint and index)
ALTER TABLE password_reset_tokens DROP COLUMN IF EXISTS token_hash;

-- Add index for lookup by user_id + code
CREATE INDEX idx_password_reset_tokens_user_code ON password_reset_tokens (user_id, code);
