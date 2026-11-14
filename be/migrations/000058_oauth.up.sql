CREATE TABLE oauth_clients (
    client_id TEXT PRIMARY KEY,
    client_name VARCHAR(100) NOT NULL DEFAULT '',
    redirect_uris TEXT[] NOT NULL,
    grant_types TEXT[] NOT NULL,
    response_types TEXT[] NOT NULL,
    is_metadata_document BOOLEAN NOT NULL DEFAULT false,
    fetched_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE oauth_grants (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id TEXT NOT NULL REFERENCES oauth_clients(client_id),
    scope TEXT NOT NULL CHECK (scope = 'mcp'),
    redirect_uri TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX oauth_grants_active_idx ON oauth_grants(user_id, client_id) WHERE revoked_at IS NULL;
CREATE TABLE oauth_authorization_codes (
    code_hash CHAR(64) PRIMARY KEY,
    grant_id UUID NOT NULL REFERENCES oauth_grants(id) ON DELETE CASCADE,
    redirect_uri TEXT NOT NULL,
    code_challenge VARCHAR(43) NOT NULL,
    resource TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ
);
CREATE INDEX oauth_codes_grant_idx ON oauth_authorization_codes(grant_id);
CREATE INDEX oauth_codes_expiry_idx ON oauth_authorization_codes(expires_at);
CREATE TABLE oauth_tokens (
    token_hash CHAR(64) PRIMARY KEY,
    kind TEXT NOT NULL CHECK (kind IN ('access', 'refresh')),
    grant_id UUID NOT NULL REFERENCES oauth_grants(id) ON DELETE CASCADE,
    resource TEXT NOT NULL,
    scope TEXT NOT NULL CHECK (scope = 'mcp'),
    expires_at TIMESTAMPTZ NOT NULL,
    rotated_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ
);
CREATE INDEX oauth_tokens_grant_idx ON oauth_tokens(grant_id);
CREATE INDEX oauth_tokens_expiry_idx ON oauth_tokens(expires_at);
ALTER TABLE mcp_audit_log ADD COLUMN oauth_grant_id UUID REFERENCES oauth_grants(id) ON DELETE SET NULL;
CREATE INDEX mcp_audit_log_oauth_grant_id_idx ON mcp_audit_log(oauth_grant_id) WHERE oauth_grant_id IS NOT NULL;
