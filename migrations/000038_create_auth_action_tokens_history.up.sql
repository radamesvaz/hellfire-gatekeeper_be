-- Append-only audit trail for auth_action_tokens lifecycle (invite + password_reset).
-- Aligned with project *_history tables: tenant, actor when known, timestamps, action.

CREATE TABLE auth_action_tokens_history (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    auth_action_token_id BIGINT NOT NULL,
    purpose VARCHAR(32) NOT NULL,
    action VARCHAR(48) NOT NULL,
    modified_on TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    modified_by_user_id BIGINT NULL,
    subject_user_id BIGINT NULL,
    metadata_json JSONB NULL,
    CONSTRAINT fk_auth_action_tokens_history_tenant
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_auth_action_tokens_history_token
        FOREIGN KEY (auth_action_token_id)
        REFERENCES auth_action_tokens(id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_auth_action_tokens_history_modified_by
        FOREIGN KEY (modified_by_user_id) REFERENCES users(id_user) ON DELETE SET NULL,
    CONSTRAINT fk_auth_action_tokens_history_subject_user
        FOREIGN KEY (subject_user_id) REFERENCES users(id_user) ON DELETE SET NULL,
    CONSTRAINT chk_auth_action_tokens_history_purpose
        CHECK (purpose IN ('invite', 'password_reset')),
    CONSTRAINT chk_auth_action_tokens_history_action
        CHECK (
            action IN (
                'invite_created',
                'invite_revoked',
                'invite_accepted',
                'password_reset_issued',
                'password_reset_completed'
            )
        )
);

CREATE INDEX idx_auth_action_tokens_history_tenant_modified
    ON auth_action_tokens_history (tenant_id, modified_on DESC);

CREATE INDEX idx_auth_action_tokens_history_token_id
    ON auth_action_tokens_history (auth_action_token_id);
