CREATE TABLE mcp_audit_log (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_token_id UUID REFERENCES api_tokens(id) ON DELETE SET NULL,
    tool VARCHAR(64) NOT NULL,
    ok BOOLEAN NOT NULL,
    error_code VARCHAR(64),
    duration_ms INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX mcp_audit_log_user_created_at_idx ON mcp_audit_log (user_id, created_at DESC);
CREATE INDEX mcp_audit_log_api_token_id_idx ON mcp_audit_log (api_token_id) WHERE api_token_id IS NOT NULL;
