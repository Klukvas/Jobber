-- Revert email_verification_tokens: restore token_hash, remove code/attempts
DROP INDEX IF EXISTS idx_email_verification_tokens_user_code;
ALTER TABLE email_verification_tokens DROP COLUMN IF EXISTS code;
ALTER TABLE email_verification_tokens DROP COLUMN IF EXISTS attempts;
ALTER TABLE email_verification_tokens ADD COLUMN token_hash VARCHAR(255) UNIQUE;

-- Revert password_reset_tokens: restore token_hash, remove code/attempts
DROP INDEX IF EXISTS idx_password_reset_tokens_user_code;
ALTER TABLE password_reset_tokens DROP COLUMN IF EXISTS code;
ALTER TABLE password_reset_tokens DROP COLUMN IF EXISTS attempts;
ALTER TABLE password_reset_tokens ADD COLUMN token_hash VARCHAR(255) UNIQUE;
