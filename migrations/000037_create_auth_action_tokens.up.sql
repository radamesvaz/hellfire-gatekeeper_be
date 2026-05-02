CREATE TABLE auth_action_tokens (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    email VARCHAR(320) NOT NULL,
    purpose VARCHAR(32) NOT NULL,
    token_hash VARCHAR(128) NOT NULL,
    subject_user_id BIGINT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ NULL,
    revoked_at TIMESTAMPTZ NULL,
    created_by_user_id BIGINT NULL,
    metadata_json JSONB NULL,
    created_on TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_on TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_auth_action_tokens_purpose
        CHECK (purpose IN ('invite', 'password_reset')),
    CONSTRAINT ux_auth_action_tokens_token_hash UNIQUE (token_hash),
    CONSTRAINT fk_auth_action_tokens_tenant
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_auth_action_tokens_subject_user
        FOREIGN KEY (subject_user_id) REFERENCES users(id_user) ON DELETE SET NULL,
    CONSTRAINT fk_auth_action_tokens_created_by_user
        FOREIGN KEY (created_by_user_id) REFERENCES users(id_user) ON DELETE SET NULL
);

CREATE INDEX idx_auth_action_tokens_tenant_purpose_email
    ON auth_action_tokens (tenant_id, purpose, email);

CREATE INDEX idx_auth_action_tokens_status
    ON auth_action_tokens (used_at, revoked_at, expires_at);
